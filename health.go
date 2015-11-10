package main

import (
	"flag"
	"fmt"
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

type config struct {
	PollInterval time.Duration
	Hosts        []string
}

func parseFlags() *config {

	var poll_interval_msec int
	flag.IntVar(&poll_interval_msec, "interval", 30000, "Server polling interval in milliseconds")

	var hosts hostsFlag
	flag.Var(&hosts, "host", "Host(s) to ping.")

	//parse the arguments
	flag.Parse()

	conf := config{}
	conf.PollInterval = time.Duration(poll_interval_msec) * time.Millisecond
	conf.Hosts = hosts[:]

	return &conf
}

func main() {
	conf := parseFlags()

	var latch sync.WaitGroup
	latch.Add(len(conf.Hosts))
	for x := 0; x < len(conf.Hosts); x++ {
		host := conf.Hosts[x]
		go func() {
			pingServer(host, conf.PollInterval)
			latch.Done()
		}()
	}
	latch.Wait()
}

func pingServer(host string, poll_interval time.Duration) {
	for {
		before := time.Now()
		res, res_err := request.NewRequest().AsGet().WithUrl(host).FetchRawResponse()
		after := time.Now()
		elapsed := after.Sub(before)
		if res_err != nil {
			down(host, elapsed)
		} else {
			defer res.Body.Close()

			if res.StatusCode != 200 {
				down(host, elapsed)
			} else {
				up(host, elapsed)
			}
		}

		time.Sleep(poll_interval)
	}
}

func up(host string, elapsed time.Duration) {
	status(host, Color("up", COLOR_GREEN), elapsed)
}

func down(host string, elapsed time.Duration) {
	status(host, Color("down", COLOR_RED), elapsed)
}

func status(host string, status string, elapsed time.Duration) {
	fmt.Printf("%s %s is %s (%s)\n", Color(time.Now().Format(time.RFC3339), COLOR_GRAY), host, status, elapsed)
}
