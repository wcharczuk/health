package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"./lib"

	"github.com/blendlabs/go-request"
)

const (
	RED    = "31"
	BLUE   = "94"
	GREEN  = "32"
	YELLOW = "33"
	WHITE  = "37"
	GRAY   = "90"

	MAX_STATS    = 1000
	TIMEOUT_MSEC = 2500
)

type hostData struct {
	Host      string
	IsUp      bool
	DownAt    *time.Time
	Errors    []time.Time
	Stats     *lib.DurationQueue
	PingCount int
}

var _lock *sync.Mutex
var _host_data map[string]*hostData

func main() {
	_lock = &sync.Mutex{}
	_host_data = map[string]*hostData{}

	config := parseFlags()

	longest_host_name := 0
	for x := 0; x < len(config.Hosts); x++ {
		l := len(config.Hosts[x])
		if l > longest_host_name {
			longest_host_name = l
		}
	}

	for x := 0; x < len(config.Hosts); x++ {
		host := config.Hosts[x]

		_host_data[host] = &hostData{Host: host, Stats: &lib.DurationQueue{}}

		go pingServer(host, time.Duration(config.PollInterval)*time.Millisecond)
	}

	for {
		clear()
		for x := 0; x < len(config.Hosts); x++ {
			status(longest_host_name, _host_data[config.Hosts[x]])
		}

		time.Sleep(time.Duration(config.PollInterval/2) * time.Millisecond)
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

func pushError(host string, errorTime time.Time) {
	_lock.Lock()
	defer _lock.Unlock()

	_host_data[host].Errors = append(_host_data[host].Errors, errorTime)
}

func getEffectivePollInterval(poll_interval time.Duration) time.Duration {
	effective_interval := time.Duration(0)
	timeout_msec := TIMEOUT_MSEC * time.Millisecond
	if timeout_msec > poll_interval {
		effective_interval = timeout_msec
	} else {
		effective_interval = poll_interval
	}

	return effective_interval
}

func pingServer(host string, poll_interval time.Duration) {
	for {
		before := time.Now()
		res, res_err := request.NewRequest().AsGet().WithUrl(host).WithTimeout(TIMEOUT_MSEC).FetchRawResponse()
		after := time.Now()
		elapsed := after.Sub(before)

		incrementPingCount(host)
		if res_err != nil {
			pushError(host, time.Now())
			setStatus(host, false)
		} else {
			defer res.Body.Close()

			if res.StatusCode != 200 {
				pushError(host, time.Now())
				setStatus(host, false)
			} else {
				pushStats(host, elapsed)
				setStatus(host, true)
			}
		}

		time.Sleep(getEffectivePollInterval(poll_interval))
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
		last := *host_data.Stats.Peek()
		avg := host_data.Stats.Mean()
		percentile_99 := host_data.Stats.Percentile(99.0)
		percentile_90 := host_data.Stats.Percentile(90.0)
		percentile_75 := host_data.Stats.Percentile(75.0)

		full_text = fmt.Sprintf("%s %6s %s: %-6s %s: %-6s %s: %-6s %s: %-6s %s: %-6s", full_host, status, last_label, formatDuration(last), avg_label, formatDuration(avg), label_99th, formatDuration(percentile_99), label_90th, formatDuration(percentile_90), label_75th, formatDuration(percentile_75))
	} else if !is_up && host_data.DownAt != nil {
		down_at := *host_data.DownAt
		down_for := time.Now().Sub(down_at)
		full_text = fmt.Sprintf("%s %6s Down For: %s", full_host, status, formatDuration(down_for))
	} else if host_data.PingCount > 0 {
		full_text = fmt.Sprintf("%s %6s %s: %-6s %s: %-6s %s: %-6s %s: %-6ss %s: %-6s", full_host, status, last_label, "--", avg_label, "--", label_99th, "--", label_90th, "--", label_75th, "--")
	} else {
		full_text = fmt.Sprintf("%s %s", full_host, unknown_status)
	}

	fmt.Println(full_text)
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
}

func loadFromPath(file_path string) (*Config, error) {
	var config Config

	config_file, read_err := os.Open(file_path)
	if read_err != nil {
		return &config, read_err
	}

	decoder := json.NewDecoder(config_file)
	decode_err := decoder.Decode(&config)

	return &config, decode_err
}

func parseFlags() *Config {

	var poll_interval_msec int
	flag.IntVar(&poll_interval_msec, "interval", 30000, "Server polling interval in milliseconds")

	var hosts hostsFlag
	flag.Var(&hosts, "host", "Host(s) to ping.")

	var show_notification bool
	flag.BoolVar(&show_notification, "notification", true, "Show OS X Notification on `down`")

	var config_file_path string
	flag.StringVar(&config_file_path, "config", "", "Load configuration from a file.")

	//parse the arguments
	flag.Parse()

	conf := Config{}
	if config_file_path != "" {
		read_conf, conf_err := loadFromPath(config_file_path)
		if conf_err != nil {
			fmt.Printf("%v\n", conf_err)
			os.Exit(1)
		}
		conf = *read_conf
	} else {
		conf.PollInterval = poll_interval_msec
		conf.Hosts = hosts[:]
		conf.ShowNotification = show_notification
	}

	return &conf
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
		return fmt.Sprintf("%ds%dms", seconds, milliseconds)
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
