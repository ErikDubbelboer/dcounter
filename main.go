package main

import (
	"flag"
	"fmt"
	"os"
)

func usageExit() {
	fmt.Fprintf(os.Stderr, "Usage: %s [cli|server]\n", os.Args[0])
	os.Exit(2)
}

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		usageExit()
	}

	if args[0] == "cli" {
		cli(args[1:])
	} else if args[0] == "server" {
		server(args[1:])
	} else {
		usageExit()
	}
}
