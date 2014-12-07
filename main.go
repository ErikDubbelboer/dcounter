package main

import (
	"flag"
	"fmt"
	"os"
)

func usageExit() {
	fmt.Fprintf(os.Stderr, "Usage: %s [cli|server]\n")
	os.Exit(2)
}

func main() {
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
