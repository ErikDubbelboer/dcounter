package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/atomx/dcounter/api"
	"github.com/codegangsta/cli"
	"github.com/ryanuber/columnize"
)

func init() {
	connect := []cli.Flag{
		cli.StringFlag{
			Name:  "connect, c",
			Value: "127.0.0.1:9374",
			Usage: "connect to this ip:port combination",
		},
	}

	app.Commands = append(app.Commands, cli.Command{
		Name:  "cli",
		Usage: "run commands",
		Subcommands: []cli.Command{
			{
				Name:  "get",
				Usage: "get the value of a counter",
				Flags: connect,
				Action: func(c *cli.Context) {
					api, err := dcounter.Dial("tcp", c.String("connect"))
					if err != nil {
						log.Printf("[ERR] %v", err)
						return
					}
					defer api.Close()

					amount, consistent, err := api.Get(c.Args().Get(0))
					if err != nil {
						log.Printf("[ERR] %v", err)
					} else {
						state := "CONSISTENT"
						if !consistent {
							state = "INCONSISTENT"
						}

						fmt.Printf("%f (%s)\n", amount, state)
					}
				},
			},
			{
				Name:  "inc",
				Usage: "increment a counter",
				Flags: connect,
				Action: func(c *cli.Context) {
					api, err := dcounter.Dial("tcp", c.String("connect"))
					if err != nil {
						log.Printf("[ERR] %v", err)
						return
					}
					defer api.Close()

					if amount, err := strconv.ParseFloat(c.Args().Get(1), 64); err != nil {
						fmt.Println(err)
					} else if err := api.Inc(c.Args().Get(0), amount); err != nil {
						fmt.Println(err)
					} else {
						fmt.Println("OK")
					}
				},
			},
			{
				Name:  "set",
				Usage: "set the value of a counter",
				Flags: connect,
				Action: func(c *cli.Context) {
					api, err := dcounter.Dial("tcp", c.String("connect"))
					if err != nil {
						log.Printf("[ERR] %v", err)
						return
					}
					defer api.Close()

					if amount, err := strconv.ParseFloat(c.Args().Get(1), 64); err != nil {
						fmt.Println(err)
					} else if err := api.Set(c.Args().Get(0), amount); err != nil {
						fmt.Println(err)
					} else {
						fmt.Println("OK")
					}
				},
			},
			{
				Name:  "list",
				Usage: "list all counters and their values",
				Flags: connect,
				Action: func(c *cli.Context) {
					api, err := dcounter.Dial("tcp", c.String("connect"))
					if err != nil {
						log.Printf("[ERR] %v", err)
						return
					}
					defer api.Close()

					if list, err := api.List(); err != nil {
						log.Printf("[ERR] %v", err)
					} else {
						for name, value := range list {
							fmt.Printf("%s: %f\n", name, value)
						}
					}
				},
			},
			{
				Name:  "join",
				Usage: "join a cluster",
				Flags: connect,
				Action: func(c *cli.Context) {
					api, err := dcounter.Dial("tcp", c.String("connect"))
					if err != nil {
						log.Printf("[ERR] %v", err)
						return
					}
					defer api.Close()

					if err := api.Join(c.Args()); err != nil {
						log.Printf("[ERR] %v", err)
					} else {
						fmt.Println("OK")
					}
				},
			},
			{
				Name:  "members",
				Usage: "list the members in the cluster",
				Flags: connect,
				Action: func(c *cli.Context) {
					api, err := dcounter.Dial("tcp", c.String("connect"))
					if err != nil {
						log.Printf("[ERR] %v", err)
						return
					}
					defer api.Close()

					if members, err := api.Members(); err != nil {
						log.Printf("[ERR] %v", err)
					} else {
						result := []string{"Name|Address"}

						for _, member := range members {
							result = append(result, fmt.Sprintf("%s|%s\n", member.Name, member.Addr))
						}

						fmt.Println(columnize.SimpleFormat(result))
					}
				},
			},
			{
				Name:  "save",
				Usage: "return the state of the server as json",
				Flags: connect,
				Action: func(c *cli.Context) {
					api, err := dcounter.Dial("tcp", c.String("connect"))
					if err != nil {
						log.Printf("[ERR] %v", err)
						return
					}
					defer api.Close()

					if data, err := api.Save(); err != nil {
						log.Printf("[ERR] %v", err)
					} else {
						fmt.Println(data)
					}
				},
			},
		},
	})
}
