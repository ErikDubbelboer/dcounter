package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/atomx/dcounter/proto"
	"github.com/hashicorp/memberlist"
)

type Server struct {
	memberlist *memberlist.Memberlist

	m Meta

	stop    chan struct{}
	stopped sync.WaitGroup

	l sync.RWMutex

	consistent bool

	counters map[string]*Counter
	replicas map[ID]map[string]*Counter

	// True means member, false means left.
	members map[ID]*State

	db *sql.DB
}

// We are consitent if all the members in members that are active
// have a node in our memberlist.
//
// getConsitent assumes s.l is locked in write mode.
func (s *Server) updateConsitent() {
	nodes := s.memberlist.Members()
	members := make(map[ID]struct{}, len(nodes))

	for _, node := range nodes {
		m := &Meta{}
		if err := m.Decode(node.Meta); err != nil {
			log.Printf("[ERR] %v\n", err)

			// With an error we always assume we're inconsistent.
			s.consistent = false
			return
		}

		members[m.Id] = struct{}{}
	}

	s.l.RLock()
	defer s.l.RUnlock()

	for id, x := range s.members {
		if x.Active == 0 {
			continue
		}

		if _, ok := members[id]; !ok {
			fmt.Printf("[WARNING] inconsistent because %s is missing\n", id)
			s.consistent = false
			return
		}
	}

	s.consistent = true
}

func (s *Server) NodeMeta(limit int) []byte {
	return s.m.Encode(limit)
}

func (s *Server) NotifyMsg(msg []byte) {
	reader := bufio.NewReader(bytes.NewReader(msg))

	var id ID
	if err := binary.Read(reader, binary.LittleEndian, &id); err != nil {
		log.Printf("[ERR] %v\n", err)
		return
	}

	s.l.Lock()
	defer s.l.Unlock()

	// If we don't know this id we can ignore it.
	if _, ok := s.members[id]; !ok {
		return
	}

	for {
		var name string
		if n, err := reader.ReadString('\n'); err != nil {
			if err == io.EOF {
				break
			} else {
				log.Printf("[ERR] %v\n", err)
				return
			}
		} else {
			// Strip off the '\n'.
			name = n[:len(n)-1]
		}

		var c Counter
		if err := binary.Read(reader, binary.LittleEndian, &c); err != nil {
			if err == io.EOF {
				break
			} else {
				log.Printf("[ERR] %v\n", err)
				return
			}
		}

		if _, ok := s.replicas[id]; !ok {
			s.replicas[id] = make(map[string]*Counter, 0)
		}

		if o, ok := s.counters[name]; !ok {
			s.counters[name] = &Counter{}

			// TODO: Does using a new counter here keep c on the heap
			// and thereby keeping the number of allocations a lot lower?
			s.replicas[id][name] = &c
		} else if o.Revision < c.Revision {
			o.Up = 0
			o.Down = 0
			o.Revision = c.Revision

			for _, counters := range s.replicas {
				delete(counters, name)
			}

			// TODO: Does using a new counter here keep c on the heap
			// and thereby keeping the number of allocations a lot lower?
			s.replicas[id][name] = &c
		} else if o, ok = s.replicas[id][name]; ok {
			if c.Up > o.Up {
				o.Up = c.Up
			}
			if c.Down > o.Down {
				o.Down = c.Down
			}
		} else {
			// TODO: Does using a new counter here keep c on the heap
			// and thereby keeping the number of allocations a lot lower?
			s.replicas[id][name] = &c
		}
	}
}

func (s *Server) GetBroadcasts(overhead, limit int) [][]byte {
	buffer := make(Buffer, 0, limit-overhead)
	writer := bufio.NewWriter(&buffer)

	if err := binary.Write(writer, binary.LittleEndian, s.m.Id); err != nil {
		log.Printf("[ERR] %v\n", err)
		return nil
	}

	s.l.RLock()
	defer s.l.RUnlock()

	for name, c := range s.counters {
		if _, err := writer.WriteString(name + "\n"); err != nil {
			//if err == io.EOF {
			//	break
			//} else {
			log.Printf("[ERR] %v\n", err)
			return nil
			//}
		}

		if err := binary.Write(writer, binary.LittleEndian, c); err != nil {
			//if err == io.EOF {
			//	break
			//} else {
			log.Printf("[ERR] %v\n", err)
			return nil
			//}
		}
	}

	if err := writer.Flush(); err != nil {
		//if err == io.EOF {
		//	break
		//} else {
		log.Printf("[ERR] %v\n", err)
		return nil
		//}
	}

	return [][]byte{buffer}
}

func (s *Server) LocalState(join bool) []byte {
	buffer := bytes.Buffer{}
	writer := bufio.NewWriter(&buffer)

	s.l.RLock()
	defer s.l.RUnlock()

	for id, s := range s.members {
		if err := binary.Write(writer, binary.LittleEndian, id); err != nil {
			log.Printf("[ERR] %v\n", err)
			return nil
		}

		if err := binary.Write(writer, binary.LittleEndian, s); err != nil {
			log.Printf("[ERR] %v\n", err)
			return nil
		}
	}

	return buffer.Bytes()
}

func (s *Server) MergeRemoteState(buf []byte, join bool) {
	reader := bufio.NewReader(bytes.NewReader(buf))

	var members map[ID]*State
	for {
		var id ID
		if err := binary.Read(reader, binary.LittleEndian, &id); err != nil {
			if err == io.EOF {
				break
			} else {
				log.Printf("[ERR] %v\n", err)
				return
			}
		}

		var s State
		if err := binary.Read(reader, binary.LittleEndian, &s); err != nil {
			if err == io.EOF {
				break
			} else {
				log.Printf("[ERR] %v\n", err)
				return
			}
		}

		members[id] = &s
	}

	s.l.Lock()
	defer s.l.Unlock()

	for id, state := range members {
		if x, ok := s.members[id]; !ok || state.When > x.When {
			s.members[id] = state
		}
	}

	go s.updateConsitent()
}

func (s *Server) NotifyJoin(node *memberlist.Node) {
	m := &Meta{}
	if err := m.Decode(node.Meta); err != nil {
		log.Printf("[ERR] %v\n", err)
		return
	}

	s.l.Lock()
	s.members[m.Id] = &State{
		Active: 1 - m.Leaving,
		When:   time.Now().UTC().Unix(),
	}
	s.l.Unlock()

	log.Printf("[INFO] %s joined\n", m.Id)
}

func (s *Server) NotifyLeave(node *memberlist.Node) {
	m := &Meta{}
	if err := m.Decode(node.Meta); err != nil {
		log.Printf("[ERR] %v\n", err)
		return
	}

	s.l.Lock()
	defer s.l.Unlock()

	if m.Leaving == 0 {
		log.Printf("[WARNING] %d is not reachable\n", m.Id)
	} else {
		s.members[m.Id] = &State{
			Active: 0,
			When:   time.Now().UTC().Unix(),
		}

		log.Printf("[INFO] %d left\n", m.Id)
	}

	go s.updateConsitent()
}

func (s *Server) NotifyUpdate(node *memberlist.Node) {
}

func (s *Server) get(name string) (float64, bool) {
	value := float64(0)

	s.l.RLock()
	defer s.l.RUnlock()

	if v, ok := s.counters[name]; ok {
		value += v.Up - v.Down
	}

	for _, counters := range s.replicas {
		if v, ok := counters[name]; ok {
			value += v.Up - v.Down
		}
	}

	return value, s.consistent
}

func (s *Server) inc(name string, diff float64) {
	s.l.Lock()
	defer s.l.Unlock()

	c, ok := s.counters[name]
	if !ok {
		c = &Counter{}
		s.counters[name] = c
	}

	if diff > 0 {
		c.Up += diff
	} else {
		c.Down += diff
	}
}

func (s *Server) reset(name string) {
	s.l.Lock()
	defer s.l.Unlock()

	if c, ok := s.counters[name]; ok {
		c.Up = 0
		c.Down = 0
		c.Revision += 1
	} else {
		s.counters[name] = &Counter{
			Revision: 1,
		}
	}
}

func (s *Server) handle(conn net.Conn) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("[ERR] %v\n%s\n\n", err, string(debug.Stack()))
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
				amount, consistent := s.get(args[0])

				if err := p.Write("RET", []string{strconv.FormatFloat(amount, 'f', -1, 64), strconv.FormatBool(consistent)}); err != nil {
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
			s.l.Lock()
			{
				s.consistent = false

				s.counters = make(map[string]*Counter, 0)
				s.replicas = make(map[ID]map[string]*Counter, 0)
				s.members = make(map[ID]*State, 0)
			}
			s.l.Unlock()

			if _, err := s.memberlist.Join(args); err != nil {
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

		log.Printf("[DEBUG] stopped listening for clients\n")

		s.stopped.Done()
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
	go http.ListenAndServe("127.0.0.1:12345", nil)

	flags := flag.NewFlagSet("server", flag.ExitOnError)
	bind := flags.String("bind", "127.0.0.1:9373", "Sets the bind address for cluster communication")
	client := flags.String("client", "127.0.0.1:9374", "Sets the address to bind for client access")
	dbName := flags.String("db", "dcounter.sqlite", "persist to this file")
	flags.Parse(arguments)

	s := Server{
		m:          Meta{},
		stop:       make(chan struct{}, 0),
		consistent: true,
		counters:   make(map[string]*Counter, 0),
		replicas:   make(map[ID]map[string]*Counter, 0),
		members:    make(map[ID]*State, 0),
	}

	s.m.Id.Randomize()

	config := memberlist.DefaultWANConfig()
	config.Delegate = &s
	config.Events = &s
	config.Name = s.m.Id.String()
	config.BindAddr, config.BindPort = splitHostPort(*bind)
	config.AdvertiseAddr = config.BindAddr
	config.AdvertisePort = config.BindPort

	config.SuspicionMult = 2
	config.ProbeInterval = 2 * time.Second
	config.ProbeTimeout = 5 * time.Second
	config.GossipNodes = 6
	config.GossipInterval = 500 * time.Millisecond

	var err error
	s.memberlist, err = memberlist.Create(config)
	if err != nil {
		panic(err)
	}

	s.stopped.Add(1)

	go s.start(*client, *dbName)

	log.Printf("[INFO] started %s\n", config.Name)

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGQUIT)

	<-c

	log.Printf("[INFO] stopping %s\n", config.Name)

	s.stop <- struct{}{}

	s.m.Leaving = 1

	s.memberlist.UpdateNode(10 * time.Second)

	time.Sleep(2 * time.Second)

	if err := s.memberlist.Leave(10 * time.Second); err != nil {
		log.Printf("[ERR] %v\n", err)
	}

	if err := s.memberlist.Shutdown(); err != nil {
		panic(err)
	}

	s.stopped.Wait()
}
