package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	dc "github.com/atomx/dcounter/server"
)

type hosts []string

func (h *hosts) String() string {
	return strings.Join(*h, ",")
}

func (h *hosts) Set(value string) error {
	*h = append(*h, value)
	return nil
}

func server(arguments []string) {
	flags := flag.NewFlagSet("server", flag.ExitOnError)
	name := flags.String("name", "", "Name for this instance")
	bind := flags.String("bind", "127.0.0.1:9373", "Sets the bind address for cluster communication")
	client := flags.String("client", "127.0.0.1:9374", "Sets the address to bind for client access")
	join := make(hosts, 0)
	flags.Var(&join, "join", "Join these hosts after starting")
	flags.Parse(arguments)

	if *name == "" {
		*name = *bind
	}

	s := dc.New(*name, *bind, *client)

	go func() {
		s.Start()

		if len(join) > 0 {
			if err := s.Join(join); err != nil {
				log.Printf("[ERR] %v", err)
			}
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGQUIT)

	<-c

	signal.Stop(c)

	if err := s.Stop(); err != nil {
		log.Printf("[ERR] %v", err)
	}
}
