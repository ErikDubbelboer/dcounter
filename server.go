package main

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/spotmx/dcounter/proto"
)

type Server struct {
	members *memberlist.Memberlist

	m Meta

	inconsistent bool

	stop chan struct{}
	wg   sync.WaitGroup

	l sync.RWMutex

	counters map[string]*Counter
	replicas map[int]map[string]float64

	db *sql.DB
}

func (s Server) NodeMeta(limit int) []byte {
	return s.m.Encode(limit)
}

func (s Server) NotifyMsg(msg []byte) {
	reader := bytes.NewReader(msg)
	decoder := gob.NewDecoder(reader)

	var id int
	if err := decoder.Decode(&id); err != nil {
		log.Printf("[ERR] %v\n", err)
		return
	}

	s.l.Lock()
	defer s.l.Unlock()

	counters, ok := s.replicas[id]
	if !ok {
		counters = make(map[string]float64, 0)
		s.replicas[id] = counters
	}

	for {
		var c Counter
		if err := decoder.Decode(&c); err != nil {
			if err != io.EOF {
				log.Printf("[ERR] %v\n", err)
			}

			break
		}

		if myc, ok := s.counters[c.Name]; ok {
			if c.Revision > myc.Revision {
				myc.Value = 0
				myc.Revision = c.Revision

				for _, ctrs := range s.replicas {
					ctrs[c.Name] = 0
				}
			}
		} else {
			s.counters[c.Name] = &Counter{
				Name:     c.Name,
				Value:    0,
				Revision: c.Revision,
			}
		}

		counters[c.Name] = c.Value
	}
}

func (s Server) GetBroadcasts(overhead, limit int) [][]byte {
	buffer := make(Buffer, 0, limit-overhead)
	encoder := gob.NewEncoder(&buffer)

	if err := encoder.Encode(s.m.id); err != nil {
		log.Printf("[ERR] %v\n", err)
		return nil
	}

	s.l.RLock()
	defer s.l.RUnlock()

	for _, c := range s.counters {
		if err := encoder.Encode(c); err != nil {
			log.Printf("[ERR] %v\n", err)
			break
		}
	}

	return [][]byte{buffer}
}

func (s Server) LocalState(join bool) []byte {
	return nil
}

func (s Server) MergeRemoteState(buf []byte, join bool) {
}

func (s Server) NotifyJoin(node *memberlist.Node) {
}

func (s Server) NotifyLeave(node *memberlist.Node) {
	m := &Meta{}
	if err := m.Decode(node.Meta); err != nil {
		log.Printf("[ERR] %v\n", err)
		return
	}

	if !m.leaving {
		s.inconsistent = true

		log.Printf("[WARNING] %d is not reachable\n", m.id)
	} else {
		log.Printf("[INFO] %d left\n", m.id)
	}
}

func (s Server) NotifyUpdate(node *memberlist.Node) {
}

func (s Server) get(name string) (float64, bool) {
	value := float64(0)

	s.l.RLock()
	defer s.l.RUnlock()

	if v, ok := s.counters[name]; ok {
		value += v.Value
	}

	for _, counters := range s.replicas {
		if v, ok := counters[name]; ok {
			value += v
		}
	}

	return value, s.inconsistent
}

func (s Server) inc(name string, diff float64) {
	s.l.Lock()
	defer s.l.Unlock()

	if c, ok := s.counters[name]; ok {
		c.Value += diff
	} else {
		s.counters[name] = &Counter{
			Name:     name,
			Value:    diff,
			Revision: 1,
		}
	}
}

func (s Server) reset(name string) {
	s.l.Lock()
	defer s.l.Unlock()

	if c, ok := s.counters[name]; ok {
		c.Value = 0
		c.Revision += 1
	} else {
		s.counters[name] = &Counter{
			Name:     name,
			Value:    0,
			Revision: 2,
		}
	}
}

func (s Server) handle(conn net.Conn) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("[ERR] %v\n", err)
		}

		if err := conn.Close(); err != nil {
			log.Printf("[ERR] %v\n", err)
		}
	}()

	p := proto.New(conn)

	for {
		cmd, args, err := p.Read()
		if err == io.EOF {
			return
		} else if err != nil {
			if opErr, ok := err.(*net.OpError); ok {
				if opErr.Err == syscall.ECONNRESET {
					return
				}
			}

			panic(err)
		}

		switch cmd {
		case "PING":
			if err := p.Write("PONG", []string{}); err != nil {
				panic(err)
			}
		case "GET":
			if len(args) < 1 {
				if err := p.Error(fmt.Errorf("GET requires exactly one argument")); err != nil {
					panic(err)
				}
			} else {
				amount, inconsistent := s.get(args[0])

				if err := p.Write("RET", []string{strconv.FormatFloat(amount, 'f', -1, 64), strconv.FormatBool(inconsistent)}); err != nil {
					panic(err)
				}
			}
		case "INC":
			if len(args) < 2 {
				if err := p.Error(fmt.Errorf("INC requires exactly two arguments")); err != nil {
					panic(err)
				}
			} else if amount, err := strconv.ParseFloat(args[1], 64); err != nil {
				if err := p.Error(err); err != nil {
					panic(err)
				}
			} else {
				s.inc(args[0], amount)

				if err := p.Write("OK", []string{}); err != nil {
					panic(err)
				}
			}
		case "RESET":
			if len(args) < 1 {
				if err := p.Error(fmt.Errorf("RESET requires exactly one argument")); err != nil {
					panic(err)
				}
			} else {
				s.reset(args[0])

				if err := p.Write("OK", []string{}); err != nil {
					panic(err)
				}
			}
		case "JOIN":
			if _, err := s.members.Join(args); err != nil {
				if err := p.Error(err); err != nil {
					panic(err)
				}
			} else if err := p.Write("OK", []string{}); err != nil {
				panic(err)
			}
		default:
			if err := p.Error(fmt.Errorf("unknown command %s", cmd)); err != nil {
				panic(err)
			}
		}
	}
}

func (s *Server) start(listen, dbName string) {
	var err error
	s.db, err = sql.Open("sqlite3", dbName)
	if err != nil {
		panic(err)
	}
	defer s.db.Close()

	socket, err := net.Listen("tcp", listen)
	if err != nil {
		panic(err)
	}

	defer func() {
		//flush()

		if err := socket.Close(); err != nil {
			log.Printf("[ERR] %v\n", err)
		}

		// TODO: We need to close all open connections.

		log.Println("[DEBUG] stopped")

		s.wg.Done()
	}()

	//create()
	//load()

	for {
		select {
		case <-s.stop:
			return
		default:
		}

		ts := socket.(*net.TCPListener)
		ts.SetDeadline(time.Now().Add(time.Second))

		conn, err := socket.Accept()
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				continue
			} else {
				log.Printf("[ERR] %v\n", err)
				continue
			}
		}

		go s.handle(conn)
	}
}

func server(arguments []string) {
	flags := flag.NewFlagSet("server", flag.ExitOnError)
	id := flags.Int("id", 0, "Sets the id for this instance")
	bind := flags.String("bind", "127.0.0.1:9373", "Sets the bind address for cluster communication")
	client := flags.String("client", "127.0.0.1:9374", "Sets the address to bind for client access")
	dbName := flags.String("db", "dcounter.sqlite", "persist to this file")
	flags.Parse(arguments)

	if *id == 0 {
		fmt.Print("Invalid id\n\nUsage:\n")
		flags.PrintDefaults()
		return
	}

	s := Server{
		m: Meta{
			id:      *id,
			leaving: false,
		},
		stop:     make(chan struct{}, 0),
		counters: make(map[string]*Counter, 0),
		replicas: make(map[int]map[string]float64, 0),
	}

	config := memberlist.DefaultWANConfig()
	config.Delegate = s
	config.Events = s
	config.Name = "node-" + strconv.FormatInt(int64(*id), 10)
	config.BindAddr, config.BindPort = splitHostPort(*bind)
	config.AdvertiseAddr = config.BindAddr
	config.AdvertisePort = config.BindPort

	config.SuspicionMult = 2
	config.ProbeInterval = 2 * time.Second
	config.ProbeTimeout = 5 * time.Second
	config.GossipNodes = 6
	config.GossipInterval = 500 * time.Millisecond

	var err error
	s.members, err = memberlist.Create(config)
	if err != nil {
		panic(err)
	}

	s.wg.Add(1)

	go s.start(*client, *dbName)

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGQUIT)

	<-c

	s.m.leaving = true

	if err := s.members.Leave(time.Second * 5); err != nil {
		log.Printf("[ERR] %v\n", err)
	}

	if err := s.members.Shutdown(); err != nil {
		panic(err)
	}

	s.stop <- struct{}{}
	s.wg.Wait()
}
