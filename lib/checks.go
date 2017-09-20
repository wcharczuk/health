package health

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/blendlabs/go-util"
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
	ticker := time.NewTicker(c.config.PollInterval)
	for {
		select {
		case <-c.abort:
			c.aborted <- true
			return
		case <-ticker.C:
			c.PingAll()
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

// PingAll pings all the hosts.
func (c *Checks) PingAll() {
	wg := sync.WaitGroup{}
	wg.Add(len(c.hosts))
	for index := range c.hosts {
		go func(x int) {
			defer wg.Done()
			host := c.hosts[x]
			err := c.Ping(host)
			if err != nil {
				host.errs.Enqueue(err)
			}
		}(index)
	}
	wg.Wait()
}

// Ping performs a ping and marks the host up or down.
func (c *Checks) Ping(h *Host) error {
	elapsed, err := h.Ping()
	if err != nil {
		h.SetDown(time.Now())
	} else {
		h.SetUp()
	}
	h.AddTiming(elapsed)
	return err
}

// HasErrors returns if the checks collection has a host with errors.
func (c *Checks) HasErrors() bool {
	for index := range c.hosts {
		if c.hosts[index].errs.Len() > 0 {
			return true
		}
	}
	return false
}

// MaxElapsed returns the maximum elapsed for the entire checks list.
func (c *Checks) MaxElapsed() time.Duration {
	var elapsed time.Duration
	for index := range c.hosts {
		c.hosts[index].stats.Each(func(v interface{}) {
			if typed, isTyped := v.(time.Duration); isTyped {
				if typed > elapsed {
					elapsed = typed
				}
			}
		})
	}
	return elapsed
}

// WriteStatus writes the statuses for all the hosts.
func (c *Checks) WriteStatus(writer io.Writer) error {
	var err error

	maxElapsed := c.MaxElapsed()

	for index := range c.hosts {
		err = c.hosts[index].WriteStatus(c.longestHost, maxElapsed, writer)
		if err != nil {
			return err
		}
	}

	if !c.HasErrors() {
		return nil
	}

	fmt.Fprintf(writer, "\n")
	fmt.Fprintf(writer, "%s\n", util.ColorYellow.Apply("Downtime:"))

	for index := range c.hosts {
		err = c.hosts[index].WriteDowntimeStatus(c.longestHost, writer)
		if err != nil {
			return err
		}
	}

	fmt.Fprintf(writer, "\n")
	fmt.Fprintf(writer, "%s\n", util.ColorRed.Apply("Errors:"))

	for index := range c.hosts {
		err = c.hosts[index].WriteErrorStatus(c.longestHost, writer)
		if err != nil {
			return err
		}
	}
	return nil
}
