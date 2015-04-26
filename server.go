package main

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/atomx/dcounter/server"
	"github.com/atomx/syslog"
	"github.com/codegangsta/cli"
)

func waitForSignal() {
	// Wait for SIGINT or SIGQUIT to stop.
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGQUIT)

	<-c

	// Stop listening for signals, this cases a
	// second signal to force a quit.
	signal.Stop(c)
}

func init() {
	hostname, err := os.Hostname()
	if err != nil {
		log.Printf("[ERR] %v", err)
		os.Exit(1)
	}

	app.Commands = append(app.Commands, cli.Command{
		Name:  "server",
		Usage: "Run a server.",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "name, n",
				Value: hostname,
				Usage: "Name of this node in the cluster. This must be unique within the cluster.",
			},
			cli.StringFlag{
				Name:  "bind, b",
				Value: "127.0.0.1:9373",
				Usage: "The address that should be bound to for internal cluster communications.",
			},
			cli.StringFlag{
				Name:  "advertise, a",
				Value: "",
				Usage: "The address that should be used to broadcast, can be different than the bind address in case of a NAT.",
			},
			cli.StringFlag{
				Name:  "client, c",
				Value: "127.0.0.1:9374",
				Usage: "The address that should be bound to for client communications.",
			},
			cli.StringSliceFlag{
				Name:  "join, j",
				Value: &cli.StringSlice{},
				Usage: "Join these hosts after starting.",
			},
			cli.StringFlag{
				Name:  "load, l",
				Value: "",
				Usage: "Load data from this json file.",
			},
			cli.StringFlag{
				Name:  "save, s",
				Value: "",
				Usage: "Persist data to this json file.",
			},
			cli.DurationFlag{
				Name:  "save-interval, i",
				Value: time.Minute,
				Usage: "Persist data this often.",
			},
			cli.StringFlag{
				Name:  "pidfile, p",
				Value: "",
				Usage: "Create a pidfile at this path.",
			},
			cli.StringFlag{
				Name:  "syslog",
				Value: "",
				Usage: "Log to syslog, protocol:address, for example udp:localhost:514",
			},
		},
		Action: func(c *cli.Context) {
			var logWriter io.Writer = os.Stderr

			if c.String("syslog") != "" {
				parts := strings.SplitN(c.String("syslog"), ":", 2)

				if len(parts) != 2 {
					log.Printf("[ERR] invalid --syslog, expected network:addr")
				}

				logWriter = syslog.New(parts[0], parts[1], "dcounter")
				log.SetOutput(logWriter)
			}

			if c.String("pidfile") != "" {
				if err := ioutil.WriteFile(c.String("pidfile"), []byte(strconv.FormatInt(int64(os.Getpid()), 10)), 0666); err != nil {
					log.Printf("[ERR] %v", err)
					return
				}
				defer os.Remove(c.String("pidfile"))
			}

			join := c.StringSlice("join")

			if len(join) > 0 && c.String("load") != "" {
				log.Printf("[ERR] Can not use --load and --join at the same time")
				return
			}

			advertise := c.String("advertise")

			if advertise == "" {
				advertise = c.String("bind")
			}

			s, err := server.New(c.String("name"), c.String("bind"), advertise, c.String("client"))
			if err != nil {
				log.Printf("[ERR] %v", err)
				return
			}

			s.Config.LogOutput = logWriter

			if err := s.Start(); err != nil {
				log.Printf("[ERR] %v", err)
				return
			}
			defer func() {
				if err := s.Stop(); err != nil {
					log.Printf("[ERR] %v", err)
				}
			}()

			if len(join) > 0 {
				for i, s := range join {
					if !strings.ContainsRune(s, ':') {
						join[i] += ":9373"
					}
				}

				if err := s.Join(join); err != nil {
					log.Printf("[ERR] %v", err)
				}
			} else if c.String("load") != "" {
				if err := s.Load(c.String("load")); err != nil {
					log.Printf("[ERR] %v", err)
				}
			}

			if c.String("save") != "" {
				go func(filename string, interval time.Duration) {
					for {
						time.Sleep(interval)

						if err := s.Save(filename); err != nil {
							log.Printf("[ERR] %v", err)
						}
					}
				}(c.String("save"), c.Duration("save-interval"))
			}

			waitForSignal()

			// After this the defer from above will stop the server gracefully.
		},
	})
}
