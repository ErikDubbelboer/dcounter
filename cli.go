package main

import (
	"flag"
	"fmt"
	"strconv"

	"github.com/spotmx/dcounter/api"
)

func cli(arguments []string) {
	flags := flag.NewFlagSet("cli", flag.ExitOnError)
	connect := flags.String("connect", "127.0.0.1:9374", "connect to this ip:port combination")
	flags.Parse(arguments)

	api, err := api.Dial("tcp", *connect)
	if err != nil {
		panic(err)
	}
	defer api.Close()

	if flags.Arg(0) == "get" {
		amount, inconsistent, err := api.Get(flags.Arg(1))
		if err != nil {
			fmt.Println(err)
		} else {
			state := "CONSITENT"
			if !inconsistent {
				state = "INCONSISTENT"
			}

			fmt.Printf("%f (%s)\n", amount, state)
		}
	} else if flags.Arg(0) == "inc" {
		if amount, err := strconv.ParseFloat(flags.Arg(2), 64); err != nil {
			fmt.Println(err)
		} else if err := api.Inc(flags.Arg(1), amount); err != nil {
			fmt.Println(err)
		}
	} else if flags.Arg(0) == "reset" {
		err := api.Reset(flags.Arg(1))
		if err != nil {
			fmt.Println(err)
		}
	} else if flags.Arg(0) == "join" {
		if err := api.Join(flags.Args()[1:]); err != nil {
			fmt.Println(err)
		}
	}
}
