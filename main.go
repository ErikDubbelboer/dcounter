package main

import (
	"os"

	"github.com/codegangsta/cli"
)

const VERSION = "0.1.0"

var app = &cli.App{
	Name:         "dcounter",
	Usage:        "high performance, eventually consisten, distributed counters.",
	Version:      VERSION,
	BashComplete: cli.DefaultAppComplete,
	Author:       "Erik Dubbelboer",
	Email:        "erik@dubbelboer.com",
	Commands:     []cli.Command{},
	Writer:       os.Stdout,
}

func init() {
	app.Action = cli.NewApp().Action
}

func main() {
	app.Run(os.Args)
}
