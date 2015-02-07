package main

import (
	"flag"
	"fmt"
	"os"
)

const VERSION = "0.1.0"

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
	} else if args[0] == "bench" {
		bench(args[1:])
	} else if args[0] == "version" {
		fmt.Println(VERSION)
	} else {
		usageExit()
	}
}
