package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"sync"
	"time"

	"github.com/blendlabs/go-exception"
	"github.com/blendlabs/go-request"
)

const (
	RED    = "31"
	BLUE   = "94"
	GREEN  = "32"
	YELLOW = "33"
	WHITE  = "37"
	GRAY   = "90"

	MAX_STATS            = 100
	DEFAULT_TIMEOUT_MSEC = 5000
)

type hostData struct {
	Host      string
	IsUp      bool
	DownAt    *time.Time
	Stats     *DurationQueue
	Error     error
	PingCount int
}

var _lock *sync.Mutex = &sync.Mutex{}
var _host_data map[string]*hostData = map[string]*hostData{}
var _transportsLock *sync.Mutex = &sync.Mutex{}
var _transports map[string]*http.Transport = map[string]*http.Transport{}

var _config_file_path *string
var _poll_interval_msec *int
var _should_show_notifications *bool
var _hosts hostsFlag

var _should_watch_config bool = false
var _config_last_write *time.Time
var _config_did_change bool = false

var _logger *log.Logger

func main() {
	config := parseFlags()

	for {
		longest_host_name := 0
		for x := 0; x < len(config.Hosts); x++ {
			l := len(config.Hosts[x])
			if l > longest_host_name {
				longest_host_name = l
			}
		}

		latch := sync.WaitGroup{}
		latch.Add(len(config.Hosts))

		polling_interval := time.Duration(config.PollInterval) * time.Millisecond
		for x := 0; x < len(config.Hosts); x++ {
			host := config.Hosts[x]

			_host_data[host] = &hostData{Host: host, Stats: &DurationQueue{}}

			go func() {
				pingServer(host, polling_interval)
				latch.Done()
			}()
		}

		for !_config_did_change {
			didLabelError := false
			clear()
			for x := 0; x < len(config.Hosts); x++ {
				status(longest_host_name, _host_data[config.Hosts[x]])
			}

			for x := 0; x < len(config.Hosts); x++ {
				host := config.Hosts[x]
				hostError := _host_data[host].Error

				if hostError != nil && !didLabelError {
					fmt.Printf("\n%s\n", color("Errors:", RED))
					didLabelError = true
				}

				showError(host, hostError)
			}
			time.Sleep(500 * time.Millisecond)
			checkConfig()
		}

		latch.Wait()
		config = reloadConfig(config)
	}
}

func incrementPingCount(host string) {
	_lock.Lock()
	defer _lock.Unlock()
	_host_data[host].PingCount = _host_data[host].PingCount + 1
}

func setStatus(host string, is_up bool) {
	_lock.Lock()
	defer _lock.Unlock()

	_host_data[host].IsUp = is_up

	if is_up {
		_host_data[host].DownAt = nil
	} else {
		if _host_data[host].DownAt == nil {
			notification(fmt.Sprintf("Down at %s", time.Now().Format(time.RFC3339)), fmt.Sprintf("%s is down.", host))
			now := time.Now()
			_host_data[host].DownAt = &now
		}
	}
}

func pushStats(host string, elapsed time.Duration) {
	_lock.Lock()
	defer _lock.Unlock()

	_host_data[host].Stats.Push(elapsed)

	if _host_data[host].Stats.Length > MAX_STATS {
		_host_data[host].Stats.Pop()
	}
}

func pushError(host string, err error) {
	_lock.Lock()
	defer _lock.Unlock()

	_host_data[host].Error = err
}

func getEffectivePollInterval(poll_interval time.Duration) time.Duration {
	timeout_msec := DEFAULT_TIMEOUT_MSEC * time.Millisecond
	if timeout_msec < poll_interval {
		return timeout_msec
	} else {
		return poll_interval
	}
}

func pingServer(host string, poll_interval time.Duration) {
	effective_poll_interval := getEffectivePollInterval(poll_interval)
	for !_config_did_change {
		before := time.Now()
		req := request.NewHTTPRequest().AsGet().WithKeepAlives().WithURL(host).WithTimeout(effective_poll_interval)
		if hasTransportForHost(host) {
			transport, _ := getTransportForHost(host)
			req = req.WithTransport(transport)
		} else {
			req = req.OnCreateTransport(func(h *url.URL, t *http.Transport) {
				if h != nil {
					addTransportForHost(*h, t)
				}
			})
		}

		if _logger != nil {
			req = req.WithLogger(request.HTTPRequestLogLevelDebug, _logger)
		}

		res, res_err := req.FetchRawResponse()

		if res != nil && res.Body != nil {
			io.Copy(ioutil.Discard, res.Body)
			res.Body.Close()
		}

		after := time.Now()
		elapsed := after.Sub(before)

		remaining_poll_interval := effective_poll_interval - elapsed

		incrementPingCount(host)
		if res_err != nil {
			setStatus(host, false)
			pushError(host, res_err)
		} else {
			if res.StatusCode != 200 {
				setStatus(host, false)
				pushError(host, exception.Newf("Non 200 Status Returned: %d", res.StatusCode))
			} else {
				pushStats(host, elapsed)
				setStatus(host, true)
			}
		}

		time.Sleep(remaining_poll_interval)
	}
}

func hasTransportForHost(host string) bool {
	_transportsLock.Lock()
	defer _transportsLock.Unlock()

	hostUrl, _ := url.Parse(host)
	hostKey := fmt.Sprintf("%s://%s", hostUrl.Scheme, hostUrl.Host)
	_, hasTransport := _transports[hostKey]
	return hasTransport
}

func getTransportForHost(host string) (*http.Transport, error) {
	_transportsLock.Lock()
	defer _transportsLock.Unlock()

	hostUrl, _ := url.Parse(host)
	hostKey := fmt.Sprintf("%s://%s", hostUrl.Scheme, hostUrl.Host)
	if transport, hasTransport := _transports[hostKey]; hasTransport {
		return transport, nil
	}
	return nil, nil
}

func addTransportForHost(hostUrl url.URL, transport *http.Transport) {
	_transportsLock.Lock()
	defer _transportsLock.Unlock()

	hostKey := fmt.Sprintf("%s://%s", hostUrl.Scheme, hostUrl.Host)
	if _, hasTransport := _transports[hostKey]; !hasTransport {
		_transports[hostKey] = transport
	}
}

func status(host_width int, host_data *hostData) {
	is_up := host_data.IsUp

	label_99th := color("99th", GRAY)
	label_90th := color("90th", GRAY)
	label_75th := color("75th", GRAY)
	avg_label := color("Average", GRAY)
	last_label := color("Last", GRAY)

	unknown_status := color("UNKNOWN", GRAY)
	status := color("UP", GREEN)
	if !is_up {
		status = color("DOWN", RED)
	}

	fixed_token := fmt.Sprintf("%%-%ds", host_width+2)
	full_host := fmt.Sprintf(fixed_token, host_data.Host)

	var full_text string
	if is_up && host_data.Stats.Length > 1 {

		last := *host_data.Stats.PeekBack()
		avg := host_data.Stats.Mean()
		percentile_99 := host_data.Stats.Percentile(99.0)
		percentile_90 := host_data.Stats.Percentile(90.0)
		percentile_75 := host_data.Stats.Percentile(75.0)

		full_text = fmt.Sprintf("%s %6s %s: %-6s %s: %-6s %s: %-7s %s: %-6s %s: %-6s", full_host, status, last_label, formatDuration(last), avg_label, formatDuration(avg), label_99th, formatDuration(percentile_99), label_90th, formatDuration(percentile_90), label_75th, formatDuration(percentile_75))
	} else if !is_up && host_data.DownAt != nil {
		down_at := *host_data.DownAt
		down_for := time.Now().Sub(down_at)
		full_text = fmt.Sprintf("%s %6s Down For: %s", full_host, status, formatDuration(down_for))
	} else if host_data.PingCount > 0 {
		full_text = fmt.Sprintf("%s %6s %s: %-6s %s: %-6s %s: %-7s %s: %-6s %s: %-6s", full_host, status, last_label, "--", avg_label, "--", label_99th, "--", label_90th, "--", label_75th, "--")
	} else {
		full_text = fmt.Sprintf("%s %s", full_host, unknown_status)
	}

	fmt.Println(full_text)
}

func showError(host string, err error) {
	if err != nil {
		fmt.Printf("%s: %s\n", host, err.Error())
	}
}

//********************************************************************************
// Console Arguments / Config
//********************************************************************************

type hostsFlag []string

func (h *hostsFlag) String() string {
	return "Hosts to ping."
}

func (h *hostsFlag) Set(value string) error {
	*h = append(*h, value)
	return nil
}

type Config struct {
	PollInterval     int      `json:"interval"`
	Hosts            []string `json:"hosts"`
	ShowNotification bool     `json:"show_notification"`
	Verbose          bool     `json:"verbose"`
}

func loadFromPath(file_path string) (*Config, *time.Time, error) {
	var config Config
	var last_write time.Time
	if info, stat_err := os.Stat(file_path); stat_err == nil {
		last_write = info.ModTime()
	} else {
		return &config, nil, stat_err
	}

	config_file, read_err := os.Open(file_path)
	if read_err != nil {
		return &config, &last_write, read_err
	}

	decoder := json.NewDecoder(config_file)
	decode_err := decoder.Decode(&config)

	return &config, &last_write, decode_err
}

func parseFlags() *Config {

	flag.Var(&_hosts, "host", "Host(s) to ping.")

	_poll_interval_msec = flag.Int("interval", 30000, "Server polling interval in milliseconds")
	_should_show_notifications = flag.Bool("notification", true, "Show OS X Notification on `down`")
	_config_file_path = flag.String("config", "", "Load configuration from a file.")

	//parse the arguments
	flag.Parse()

	conf := Config{}
	if _config_file_path != nil && *_config_file_path != "" {
		read_conf, last_write, conf_err := loadFromPath(*_config_file_path)
		if conf_err != nil {
			fmt.Printf("%v\n", conf_err)
			os.Exit(1)
		}

		_config_last_write = last_write
		_should_watch_config = true
		_config_did_change = false

		conf = *read_conf
	} else {
		if _poll_interval_msec != nil {
			conf.PollInterval = *_poll_interval_msec
		}
		if len(_hosts) != 0 {
			conf.Hosts = append(conf.Hosts, _hosts[:]...)
		}

		if _should_show_notifications != nil {
			conf.ShowNotification = *_should_show_notifications
		}
	}

	return &conf
}

func checkConfig() {
	if _should_watch_config && _config_file_path != nil {
		var last_write time.Time
		if info, stat_err := os.Stat(*_config_file_path); stat_err == nil {
			last_write = info.ModTime()

			if _config_last_write != nil {
				if last_write.After(*_config_last_write) {
					_config_did_change = true
				}
			}
		}
	}
}

func reloadConfig(old *Config) *Config {
	read_conf, last_write, conf_err := loadFromPath(*_config_file_path)
	if conf_err != nil {
		return old
	}

	_config_last_write = last_write
	_config_did_change = false

	return read_conf
}

//********************************************************************************
// Utility
//********************************************************************************

func formatDuration(d time.Duration) string {
	if d > time.Hour {
		hours := d / time.Hour
		hours_remainder := d - (hours * time.Hour)
		minutes := hours_remainder / time.Minute
		minute_remainder := hours_remainder - (minutes * time.Minute)
		seconds := minute_remainder / time.Second

		return fmt.Sprintf("%dh%dm%ds", hours, minutes, seconds)
	} else if d > time.Minute {
		minutes := d / time.Minute
		minute_remainder := d - (minutes * time.Minute)
		seconds := minute_remainder / time.Second

		return fmt.Sprintf("%dm%ds", minutes, seconds)
	} else if d > time.Second {
		seconds := d / time.Second
		seconds_remainder := d - (seconds * time.Second)
		milliseconds := seconds_remainder / time.Millisecond
		return fmt.Sprintf("%d.%ds", seconds, milliseconds)
	} else if d > time.Millisecond {
		milliseconds := d / time.Millisecond
		return fmt.Sprintf("%dms", milliseconds)
	} else {
		microseconds := d / time.Microsecond
		return fmt.Sprintf("%dµs", microseconds)
	}
}

func notification(message, title string) {
	cmd_name := "osascript"
	full_cmd_name, path_err := exec.LookPath(cmd_name)
	if path_err != nil {
		return
	}

	arg_body := fmt.Sprintf("display notification \"%s\" with title \"%s\" sound name \"Basso\"", message, title)
	cmd := exec.Command(full_cmd_name, "-e", arg_body)
	cmd.Run()
}

func clear() {
	fmt.Print("\033[H\033[2J")
}

func color(input string, colorCode string) string {
	return fmt.Sprintf("\033[%s;01m%s\033[0m", colorCode, input)
}

//********************************************************************************
// Duration Queue
//********************************************************************************

type durationList []time.Duration

func (dl durationList) Len() int {
	return len(dl)
}

func (dl durationList) Less(i, j int) bool {
	return dl[i] < dl[j]
}

func (dl durationList) Swap(i, j int) {
	dl[i], dl[j] = dl[j], dl[i]
}

type durationNode struct {
	Value    time.Duration
	Previous *durationNode
	Next     *durationNode
}

type DurationQueue struct {
	Head   *durationNode
	Tail   *durationNode
	Length int
}

func (dq *DurationQueue) ToArray() []time.Duration {
	if dq.Head == nil {
		return []time.Duration{}
	}

	results := []time.Duration{}
	node_ptr := dq.Head
	for node_ptr != nil {
		results = append(results, node_ptr.Value)
		node_ptr = node_ptr.Previous
	}

	return results
}

func (dq *DurationQueue) Push(value time.Duration) {
	new_node := durationNode{Value: value}

	if dq.Tail != nil {
		dq.Tail.Previous = &new_node
	}
	new_node.Next = dq.Tail

	if dq.Head == nil {
		dq.Head = &new_node
	}

	dq.Tail = &new_node
	dq.Length = dq.Length + 1
}

func (dq *DurationQueue) Pop() *time.Duration {
	if dq.Head == nil {
		return nil
	}

	old_head := dq.Head
	value := old_head.Value

	dq.Head = dq.Head.Previous
	if dq.Head == nil {
		dq.Tail = nil
	} else {
		dq.Head.Next = nil
	}

	dq.Length = dq.Length - 1

	return &value
}

func (dq *DurationQueue) Peek() *time.Duration {
	if dq.Head == nil {
		return nil
	}

	return &dq.Head.Value
}

func (dq *DurationQueue) PeekBack() *time.Duration {
	if dq.Tail == nil {
		return nil
	}

	return &dq.Tail.Value
}

func (dq *DurationQueue) Mean() time.Duration {
	if dq.Head == nil {
		return 0
	}

	accum := time.Duration(0)

	node_ptr := dq.Head
	for node_ptr != nil {
		accum = accum + node_ptr.Value
		node_ptr = node_ptr.Previous
	}

	return accum / time.Duration(dq.Length)
}

func (dq *DurationQueue) Percentile(percentile float64) time.Duration {
	if dq.Head == nil {
		return time.Duration(0)
	}

	values := dq.ToArray()
	sort.Sort(durationList(values))

	index := (percentile / 100.0) * float64(len(values))
	if index == float64(int64(index)) {
		i := float64ToInt(index)

		if i < 1 {
			return time.Duration(0)
		}

		value_1 := float64(values[i-1])
		value_2 := float64(values[i])
		to_average := []float64{value_1, value_2}
		averaged := mean(to_average)

		return time.Duration(int64(averaged))
	} else {
		i := float64ToInt(index)
		if i < 1 {
			return time.Duration(0)
		}

		return values[i-1]
	}
}

func mean(input []float64) float64 {
	accum := 0.0
	input_len := len(input)
	for i := 0; i < input_len; i++ {
		v := input[i]
		accum = accum + float64(v)
	}
	return accum / float64(input_len)
}

func round(input float64, places int) (rounded float64, err error) {
	if math.IsNaN(input) {
		return 0.0, errors.New("Not a number")
	}

	sign := 1.0
	if input < 0 {
		sign = -1
		input *= -1
	}

	precision := math.Pow(10, float64(places))
	digit := input * precision
	_, decimal := math.Modf(digit)

	if decimal >= 0.5 {
		rounded = math.Ceil(digit)
	} else {
		rounded = math.Floor(digit)
	}

	return rounded / precision * sign, nil
}

func float64ToInt(input float64) (output int) {
	r, _ := round(input, 0)
	return int(r)
}
