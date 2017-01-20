package main

import (
	"os"
	"os/signal"

	"github.com/daohoangson/go-sitemirror/engine"
)

func main() {
	config := engine.ParseConfig(os.Args[0], os.Args[1:])
	e := engine.FromConfig(config)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	for sig := range c {
		switch sig {
		case os.Interrupt:
			e.Stop()
			os.Exit(0)
		}
	}
}
