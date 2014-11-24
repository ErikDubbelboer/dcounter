package main

import (
	"flag"
	"fmt"
	"strconv"

	"github.com/spotmx/dcounter/api"
)

func main() {
	connect := flag.String("connect", "127.0.0.1:9373", "connect to this ip:port combination")
	flag.Parse()

	api, err := api.Dial("tcp", *connect)
	if err != nil {
		panic(err)
	}
	defer api.Close()

	if flag.Arg(0) == "get" {
		amount, err := api.Get(flag.Arg(1))
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(amount)
		}
	} else if flag.Arg(0) == "inc" {
		if amount, err := strconv.ParseFloat(flag.Arg(2), 64); err != nil {
			fmt.Println(err)
		} else if err := api.Inc(flag.Arg(1), amount); err != nil {
			fmt.Println(err)
		}
	} else if flag.Arg(0) == "replicate" {
		if err := api.Replicate(flag.Arg(1), flag.Arg(2)); err != nil {
			fmt.Println(err)
		}
	} else if flag.Arg(0) == "sync" {
		if err := api.Sync(); err != nil {
			fmt.Println(err)
		}
	}
}
