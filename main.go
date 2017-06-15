package main

import (
	"fmt"
	"log"

	lib "github.com/medyagh/health/lib"
	tm "github.com/buger/goterm"
)

func main() {
	config, err := lib.NewConfigFromFlags()

	if err != nil {
		log.Fatal(err)
	}

	checks, err := lib.NewChecksFromConfig(config)
	if err != nil {
		log.Fatal(err)
	}
	checks.OnInterval(func(c *lib.Checks) {
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
	tm.Clear() // Clear current screen
	tm.Flush()
}
