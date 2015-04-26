package main

import (
	"os"

	"github.com/atomx/dcounter/server"
	"github.com/codegangsta/cli"
)

var app = &cli.App{
	Name:         "dcounter",
	Usage:        "high performance, eventually consisten, distributed counters.",
	Version:      server.VERSION,
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
