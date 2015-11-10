package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/blendlabs/go-request"
)

func main() {
	raw_args := os.Args[:]

	if len(raw_args) < 2 {
		fmt.Println("Need to specify a host.")
		os.Exit(1)
	}

	raw_host := raw_args[1]

	if strings.Contains(raw_host, ",") {
		hosts := strings.Split(raw_host, ",")

		var latch sync.WaitGroup
		latch.Add(len(hosts))
		for x := 0; x < len(hosts); x++ {
			host := hosts[x]
			go func() {
				pingServer(host)
				latch.Done()
			}()
		}
		latch.Wait()
	} else {
		pingServer(raw_host)
	}
}

func pingServer(host string) {
	for {
		res, res_err := request.NewRequest().AsGet().WithUrl(host).FetchRawResponse()
		defer res.Body.Close()

		if res.StatusCode != 200 || res_err != nil {
			fmt.Println()
			fmt.Printf("%s: %s is down.\n", time.Now().Format(time.RFC3339), host)
		} else {
			//fmt.Printf("%s: %s is up\n", time.Now().Format(time.RFC3339), host)
			fmt.Print(".")
		}

		time.Sleep(30 * time.Second)
	}
}
