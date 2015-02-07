package main

import (
	"flag"
	"fmt"
	"strconv"

	"github.com/atomx/dcounter/api"
)

func cli(arguments []string) {
	flags := flag.NewFlagSet("cli", flag.ExitOnError)
	connect := flags.String("connect", "127.0.0.1:9374", "connect to this ip:port combination")
	flags.Parse(arguments)

	api, err := dcounter.Dial("tcp", *connect)
	if err != nil {
		panic(err)
	}
	defer api.Close()

	if flags.Arg(0) == "get" {
		amount, consistent, err := api.Get(flags.Arg(1))
		if err != nil {
			fmt.Println(err)
		} else {
			state := "CONSISTENT"
			if !consistent {
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
	} else if flags.Arg(0) == "set" {
		if value, err := strconv.ParseFloat(flags.Arg(2), 64); err != nil {
			fmt.Println(err)
		} else if err := api.Set(flags.Arg(1), value); err != nil {
			fmt.Println(err)
		}
	} else if flags.Arg(0) == "list" {
		if list, err := api.List(); err != nil {
			fmt.Println(err)
		} else {
			for name, value := range list {
				fmt.Printf("%s: %f\n", name, value)
			}
		}
	} else if flags.Arg(0) == "join" {
		if err := api.Join(flags.Args()[1:]); err != nil {
			fmt.Println(err)
		}
	} else if flags.Arg(0) == "save" {
		if data, err := api.Save(); err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(data)
		}
	} else if flags.Arg(0) == "members" {
		if members, err := api.Members(); err != nil {
			fmt.Println(err)
		} else {
			for _, member := range members {
				fmt.Printf("%s\t%s\n", member.Addr, member.Name)
			}
		}
	}
}
