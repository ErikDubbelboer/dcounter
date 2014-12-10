package main

import (
	"flag"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/atomx/dcounter/api"
)

func runBench(cb func()) {
	ops := uint32(0)

	go func() {
		for {
			cb()

			atomic.AddUint32(&ops, 1)
		}
	}()

	for {
		time.Sleep(time.Second)

		n := atomic.SwapUint32(&ops, 0)

		fmt.Printf("%10.d per second\n", n)
	}
}

func bench(arguments []string) {
	flags := flag.NewFlagSet("bench", flag.ExitOnError)
	connect := flags.String("connect", "127.0.0.1:9374", "connect to this ip:port combination")
	flags.Parse(arguments)

	api, err := api.Dial("tcp", *connect)
	if err != nil {
		panic(err)
	}
	defer api.Close()

	if flags.Arg(0) == "get" {
		runBench(func() {
			if _, _, err := api.Get(flags.Arg(1)); err != nil {
				panic(err)
			}
		})
	} else if flags.Arg(0) == "inc" {
		if amount, err := strconv.ParseFloat(flags.Arg(2), 64); err != nil {
			fmt.Println(err)
		} else {
			runBench(func() {
				if err := api.Inc(flags.Arg(1), amount); err != nil {
					panic(err)
				}
			})
		}
	}
}
