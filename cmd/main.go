package main

import (
	"fmt"
	"log"
	"time"

	"github.com/wcharczuk/health"
)

const (
	// DefaultMaxStats is the default number of deltas to keep per host.
	DefaultMaxStats = 100
	// DefaultTimeout is the connection timeout.
	DefaultTimeout = 5000 * time.Millisecond
)

func main() {
	config, err := health.NewConfigFromFlags()

	if err != nil {
		log.Fatal(err)
	}

	checks := health.NewChecksFromConfig(config)
	checks.OnInterval(func(c *health.Checks) {
		clear()
		fmt.Printf(c.Status())
	})
	checks.Start()
}

func clear() {
	fmt.Print("\033[H\033[2J")
}
