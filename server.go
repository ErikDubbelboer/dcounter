package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

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
	bindAddr := flags.String("bind-addr", "127.0.0.1:9373", "Sets the bind address for cluster communication")
	bindPort := flags.Int("bind-port", 9373, "Sets the bind port address for cluster communication")
	client := flags.String("client", "127.0.0.1:9374", "Sets the address to bind for client access")
	join := make(hosts, 0)
	flags.Var(&join, "join", "Join these hosts after starting")
	load := flags.String("load", "", "json file to load data from")
	save := flags.String("save", "", "json file to save data to")
	saveInterval := flags.Duration("save-interval", time.Minute, "how often to save the data")
	flags.Parse(arguments)

	if *name == "" {
		*name = *bindAddr
	}

	if ips, err := net.LookupIP(*bindAddr); err != nil {
		log.Printf("[ERR] %v", err)
		return
	} else {
		*bindAddr = ips[0].String()
	}

	bind := *bindAddr + ":" + strconv.FormatInt(int64(*bindPort), 10)

	s := dc.New(*name, bind, *client)

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
		if err := s.Join(join); err != nil {
			log.Printf("[ERR] %v", err)
		}
	} else if *load != "" {
		if err := s.Load(*load); err != nil {
			log.Printf("[ERR] %v", err)
		}
	}

	if *save != "" {
		go func(filename string, interval time.Duration) {
			for {
				time.Sleep(interval)

				if err := s.Save(filename); err != nil {
					log.Printf("[ERR] %v", err)
				}
			}
		}(*save, *saveInterval)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGQUIT)

	<-c

	signal.Stop(c)
}
