package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/wcharczuk/health"
)

const (
	byteNewLine = byte('\n')
	byteTab     = byte('\t')
)

func main() {
	// set the term to raw mode
	initialSettings, tty := initTerm()
	// on quit, put the term back in interactive mode
	defer restoreTerm(initialSettings, tty)

	config, err := health.NewConfigFromFlags()
	if err != nil {
		log.Fatal(err)
	}

	checks, err := health.NewChecksFromConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	// handle os signals
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		<-sigc
		checks.Stop()
		os.Exit(0)
	}()

	// handle ctrl-c like inputs
	go func() {
		var c = make([]byte, 1)
		for {
			os.Stdin.Read(c)
			switch c[0] {
			case health.ANSI.ETX:
				checks.Stop()
				os.Exit(0)
			}
		}
	}()

	fmt.Fprintf(tty, "gathering metrics...\n")
	checks.OnInterval(func(c *health.Checks) {
		err = render(tty, c)
		if err != nil {
			log.Fatal(err)
		}
	})
	checks.Start()
}

func initTerm() (*health.Termios, *os.File) {
	tty := os.Stdout
	initialSettings, err := health.MakeRaw(os.Stdout.Fd())
	if err != nil {
		log.Fatal(err)
	}

	tty.Write(health.ANSI.Clear)
	tty.Write(health.ANSI.HideCursor)
	tty.Write(health.ANSI.MoveCursor(0, 0))

	return initialSettings, tty
}

func restoreTerm(initialSettings *health.Termios, tty *os.File) {
	tty.Write(health.ANSI.ShowCursor)
	err := health.TcSetAttr(tty.Fd(), initialSettings)
	tty.Close()
	if err != nil {
		log.Fatal(err)
	}
}

func render(tty *os.File, c *health.Checks) (err error) {
	tty.Write(health.ANSI.Clear)
	tty.Write(health.ANSI.HideCursor)
	tty.Write(health.ANSI.MoveCursor(0, 0))
	tty.Write(health.ANSI.ColorReset)

	if len(c.Hosts()) > 0 {
		err = c.WriteStatus(tty)
	} else {
		err = fmt.Errorf("no hosts configured")
	}
	return
}
