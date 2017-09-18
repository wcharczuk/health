package health

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// CheckIntervalAction is a func(c *Checks)
type CheckIntervalAction func(c *Checks)

// NewChecksFromConfig initializes a check set from a config.
func NewChecksFromConfig(config *Config) (*Checks, error) {
	c := &Checks{
		config:  config,
		abort:   make(chan bool),
		aborted: make(chan bool),
	}
	var longestHost int
	for _, h := range config.Hosts {
		host, err := NewHost(h, config.PingTimeout, config.MaxStats)
		if err != nil {
			return nil, err
		}
		c.hosts = append(
			c.hosts,
			host,
		)
		if len(h) > longestHost {
			longestHost = len(h)
		}
	}
	c.longestHost = longestHost
	return c, nil
}

// Checks is the entrypoint for running healthchecks.
type Checks struct {
	config         *Config
	hosts          []*Host
	abort          chan bool
	aborted        chan bool
	intervalAction CheckIntervalAction
	longestHost    int
}

// Hosts returns the hosts for the checks collection.
func (c Checks) Hosts() []*Host {
	return c.hosts
}

// OnInterval registers a hook to be run before the ping sleep.
func (c *Checks) OnInterval(action CheckIntervalAction) {
	c.intervalAction = action
}

// Start starts the healthcheck
func (c *Checks) Start() {
	wg := sync.WaitGroup{}
	ticker := time.NewTicker(c.config.PollInterval)
	for {
		select {
		case <-c.abort:
			c.aborted <- true
			return
		case <-ticker.C:
			wg.Add(len(c.hosts))
			for index := range c.hosts {
				go func(x int) {
					defer wg.Done()
					host := c.hosts[x]
					doPing(host)
				}(index)
			}
			wg.Wait()
			if c.intervalAction != nil {
				c.intervalAction(c)
			}
		}
	}
}

// Stop stops the healthcheck loop.
func (c *Checks) Stop() {
	c.abort <- true
	<-c.aborted
}

// WriteStatus writes the statuses for all the hosts.
func (c *Checks) WriteStatus(writer io.Writer) error {
	var err error
	for index := range c.hosts {
		err = c.hosts[index].WriteStatus(c.longestHost, writer)
		if err != nil {
			return err
		}
	}
	_, err = fmt.Fprintf(writer, "\n")
	return err
}

func doPing(h *Host) {
	elapsed, err := h.Ping()
	if err != nil {
		h.SetDown(time.Now())
	} else {
		h.SetUp()
	}
	h.AddTiming(elapsed)
}
