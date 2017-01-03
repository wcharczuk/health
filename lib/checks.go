package health

import (
	"strings"
	"sync"
	"time"
)

// CheckIntervalAction is a func(c *Checks)
type CheckIntervalAction func(c *Checks)

// NewChecksFromConfig initializes a check set from a config.
func NewChecksFromConfig(config *Config) (*Checks, error) {
	c := &Checks{
		config: config,
		lock:   &sync.RWMutex{},
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
	running        bool
	intervalAction CheckIntervalAction
	longestHost    int
	lock           *sync.RWMutex
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
	c.running = true
	for c.running {
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
		time.Sleep(c.config.PollInterval.AsTimeDuration())
	}
}

// Stop stops the healthcheck loop.
func (c *Checks) Stop() {
	c.running = false
}

// Status returns the statuses for all the hosts.
func (c *Checks) Status() string {
	c.lock.RLock()
	defer c.lock.RUnlock()

	var statuses []string
	for index := range c.hosts {
		statuses = append(statuses, c.hosts[index].Status(c.longestHost))
	}
	return strings.Join(statuses, "\n")
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
