package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	dc "github.com/atomx/dcounter/server"
)

func server(arguments []string) {
	flags := flag.NewFlagSet("server", flag.ExitOnError)
	bind := flags.String("bind", "127.0.0.1:9373", "Sets the bind address for cluster communication")
	client := flags.String("client", "127.0.0.1:9374", "Sets the address to bind for client access")
	flags.Parse(arguments)

	s := dc.New(*bind, *client)

	go s.Start()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGQUIT)

	<-c

	if err := s.Stop(); err != nil {
		log.Printf("[ERR] %v", err)
	}
}
