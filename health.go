package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/blendlabs/go-request"
)

const (
	COLOR_RED    = "31"
	COLOR_BLUE   = "94"
	COLOR_GREEN  = "32"
	COLOR_YELLOW = "33"
	COLOR_WHITE  = "37"
	COLOR_GRAY   = "90"
)

func Color(input string, colorCode string) string {
	return fmt.Sprintf("\033[%s;01m%s\033[0m", colorCode, input)
}

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
			errorMessage(conf_err.Error())
		}
		conf = *read_conf
	} else {
		conf.PollInterval = poll_interval_msec
		conf.Hosts = hosts[:]
		conf.ShowNotification = show_notification
	}

	return &conf
}

func main() {
	config := parseFlags()

	var latch sync.WaitGroup
	latch.Add(len(config.Hosts))
	for x := 0; x < len(config.Hosts); x++ {
		host := config.Hosts[x]
		go func() {
			pingServer(host, config)
			latch.Done()
		}()
	}
	latch.Wait()
}

func pingServer(host string, config *Config) {
	for {
		before := time.Now()
		res, res_err := request.NewRequest().AsGet().WithUrl(host).FetchRawResponse()
		after := time.Now()
		elapsed := after.Sub(before)
		if res_err != nil {
			down(host, elapsed, config.ShowNotification)
		} else {
			defer res.Body.Close()

			if res.StatusCode != 200 {
				down(host, elapsed, config.ShowNotification)
			} else {
				up(host, elapsed)
			}
		}

		time.Sleep(time.Duration(config.PollInterval) * time.Millisecond)
	}
}

func up(host string, elapsed time.Duration) {
	status(host, Color("up", COLOR_GREEN), elapsed)
}

func down(host string, elapsed time.Duration, show_notification bool) {
	status(host, Color("down", COLOR_RED), elapsed)
	if show_notification {
		message := fmt.Sprintf("Last request took %v", elapsed)
		title := fmt.Sprintf("%s is down", host)
		notification(message, title)
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
	cmd_err := cmd.Run()
	if cmd_err != nil {
		errorMessage(cmd_err.Error())
	}
}

func errorMessage(message string) {
	fmt.Printf("%s %s %s\n", Color(time.Now().Format(time.RFC3339), COLOR_GRAY), Color("error", COLOR_RED), message)
}

func status(host string, status string, elapsed time.Duration) {
	fmt.Printf("%s %s is %s (%s)\n", Color(time.Now().Format(time.RFC3339), COLOR_GRAY), host, status, elapsed)
}
