package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/wcharczuk/health/lib"

	"github.com/blendlabs/go-request"
)

const (
	RED    = "31"
	BLUE   = "94"
	GREEN  = "32"
	YELLOW = "33"
	WHITE  = "37"
	GRAY   = "90"

	MAX_STATS = 1000
)

type statusUpdate struct {
	Host   string
	Status bool
}

type statsUpdate struct {
	Host    string
	Elapsed time.Duration
}

type errorUpdate struct {
	Host      string
	Timestamp time.Time
}

var _lock *sync.Mutex

var statuses map[string]bool
var stats_queues map[string]*lib.DurationQueue
var errors map[string][]time.Time

func main() {
	_lock = &sync.Mutex{}
	config := parseFlags()

	longest_host := 0

	statuses = map[string]bool{}
	stats_queues = map[string]*lib.DurationQueue{}
	errors = map[string][]time.Time{}

	for x := 0; x < len(config.Hosts); x++ {
		host := config.Hosts[x]

		statuses[host] = false
		stats_queues[host] = &lib.DurationQueue{}
		errors[host] = []time.Time{}

		if len(host) > longest_host {
			longest_host = len(host)
		}

		go pingServer(host, time.Duration(config.PollInterval)*time.Millisecond)
	}

	for {
		clear()
		for x := 0; x < len(config.Hosts); x++ {
			host := config.Hosts[x]

			_lock.Lock()
			is_up, _ := statuses[host]
			stats, _ := stats_queues[host]

			if stats.Length > 1 {
				last := *stats.PeekBack()
				status(host, longest_host, is_up, last, stats.Mean(), stats.StdDev())
			} else {
				fmt.Printf("Pinging %s ...\n", host)
			}
			_lock.Unlock()
		}

		time.Sleep(time.Duration(config.PollInterval/2) * time.Millisecond)
	}
}

func setStatus(host string, is_up bool) {
	_lock.Lock()
	defer _lock.Unlock()
	statuses[host] = is_up
}

func pushStats(host string, elapsed time.Duration) {
	_lock.Lock()
	defer _lock.Unlock()
	stats_queues[host].Push(elapsed)

	if stats_queues[host].Length > MAX_STATS {
		stats_queues[host].Pop()
	}
}

func pushError(host string, errorTime time.Time) {
	_lock.Lock()
	defer _lock.Unlock()

	errors[host] = append(errors[host], errorTime)
}

func pingServer(host string, poll_interval time.Duration) {
	for {
		before := time.Now()
		res, res_err := request.NewRequest().AsGet().WithUrl(host).FetchRawResponse()
		after := time.Now()
		elapsed := after.Sub(before)

		pushStats(host, elapsed)

		if res_err != nil {
			pushError(host, time.Now())
			setStatus(host, false)
		} else {
			defer res.Body.Close()

			if res.StatusCode != 200 {
				pushError(host, time.Now())
				setStatus(host, false)
			} else {
				setStatus(host, true)
			}
		}

		time.Sleep(poll_interval)
	}
}

func status(host string, host_width int, is_up bool, last time.Duration, avg time.Duration, stddev time.Duration) {
	std_dev_label := color("StdDev", GRAY)
	avg_label := color("Average", GRAY)
	last_label := color("Last", GRAY)

	status := color("UP", GREEN)
	if !is_up {
		status = color("DOWN", RED)
	}

	fixed_token := fmt.Sprintf("%%-%ds", host_width+2)
	full_host := fmt.Sprintf(fixed_token, host)

	full_text := fmt.Sprintf("%s %s %s: %7s %s: %7s %s: %7s", full_host, status, last_label, formatDuration(last), avg_label, formatDuration(avg), std_dev_label, formatDuration(stddev))
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
