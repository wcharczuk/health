package main

import (
	"fmt"
	"log"

	. "github.com/wcharczuk/health/lib"
)

func main() {
	config, err := NewConfigFromFlags()

	if err != nil {
		log.Fatal(err)
	}

	checks := NewChecksFromConfig(config)
	checks.OnInterval(func(c *Checks) {
		clear()
		if len(c.Hosts()) > 0 {
			fmt.Printf(c.Status())
		} else {
			fmt.Println("No hosts configured.")
		}
	})
	checks.Start()
}

func clear() {
	fmt.Print("\033[H\033[2J")
}
