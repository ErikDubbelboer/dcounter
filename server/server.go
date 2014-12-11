package server

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/atomx/dcounter/proto"
	"github.com/hashicorp/memberlist"
)

type Server struct {
	client string

	Config *memberlist.Config

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

	logger *log.Logger
}

// We are consitent if all the members in members that are active
// have a node in our memberlist.
func (s *Server) updateConsistent() {
	nodes := s.memberlist.Members()
	members := make(map[ID]struct{}, len(nodes))

	for _, node := range nodes {
		m := &Meta{}
		if err := m.Decode(node.Meta); err != nil {
			s.logger.Printf("[ERR] %v", err)

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
			s.logger.Printf("[WARNING] inconsistent because %s is missing", id)
			s.consistent = false
			return
		}
	}

	s.consistent = true
}

func (s *Server) logDebug() {
	nodes := s.memberlist.Members()
	members := make(map[ID]*memberlist.Node, len(nodes))

	s.logger.Print("memberlist:")

	for _, node := range nodes {
		m := &Meta{}
		if err := m.Decode(node.Meta); err != nil {
			s.logger.Printf("[ERR] %v", err)

			// With an error we always assume we're inconsistent.
			s.consistent = false
			return
		}

		s.logger.Printf("%s %s", node.Name, m.Id)

		members[m.Id] = node
	}

	s.l.RLock()
	defer s.l.RUnlock()

	s.logger.Print("members:")

	for id, x := range s.members {
		name := "-"

		if node, ok := members[id]; ok {
			name = node.Name
		}

		s.logger.Printf("%s %v %s", id, x.Active, name)
	}
}

func (s *Server) NodeMeta(limit int) []byte {
	return s.m.Encode(limit)
}

func (s *Server) readCounters(reader *bufio.Reader) error {
	var l uint32
	if err := binary.Read(reader, binary.LittleEndian, &l); err != nil {
		return err
	}

	if l == 0 {
		return nil
	}

	var id ID
	if err := binary.Read(reader, binary.LittleEndian, &id); err != nil {
		return err
	}

	for i := uint32(0); i < l; i++ {
		var name string
		if n, err := reader.ReadString('\n'); err != nil {
			return err
		} else {
			// Strip off the '\n'.
			name = n[:len(n)-1]
		}

		var c Counter
		if err := binary.Read(reader, binary.LittleEndian, &c); err != nil {
			return err
		}

		// If we don't know this id we can ignore it.
		if _, ok := s.members[id]; !ok {
			return nil
		}

		// Never merge our own counters.
		// This will happen with a TCP state exchange of all counters.
		if id == s.m.Id {
			continue
		}

		if _, ok := s.replicas[id]; !ok {
			s.replicas[id] = make(map[string]*Counter, 0)
			s.replicas[id][name] = &c
		} else {
			if o, ok := s.counters[name]; ok && o.Revision < c.Revision {
				o.Up = 0
				o.Down = 0
				o.Revision = c.Revision

				for id, counters := range s.replicas {
					if id == s.m.Id {
						continue
					}

					delete(counters, name)
				}
			}

			if o, ok := s.replicas[id][name]; ok && o.Revision < c.Revision {
				o.Up = c.Up
				o.Down = c.Down
				o.Revision = c.Revision
			} else if ok {
				if c.Up > o.Up {
					o.Up = c.Up
				}
				if c.Down > o.Down {
					o.Down = c.Down
				}
			} else {
				s.replicas[id][name] = &c
			}
		}
	}

	return nil
}

func (s *Server) NotifyMsg(msg []byte) {
	reader := bufio.NewReader(bytes.NewReader(msg))

	s.l.Lock()
	defer s.l.Unlock()

	if err := s.readCounters(reader); err != nil && err != io.EOF {
		s.logger.Printf("[ERR] %v", err)
	}
}

func (s *Server) writeCounters(writer *bufio.Writer, id ID) error {
	l := uint32(len(s.replicas[id]))

	if err := binary.Write(writer, binary.LittleEndian, l); err != nil {
		s.logger.Printf("[ERR] %v", err)
		return nil
	}

	if err := writer.Flush(); err != nil {
		return err
	}

	if l == 0 {
		return nil
	}

	if err := binary.Write(writer, binary.LittleEndian, id); err != nil {
		s.logger.Printf("[ERR] %v", err)
		return nil
	}

	for name, c := range s.replicas[id] {
		if _, err := writer.WriteString(name + "\n"); err != nil {
			return err
		}

		if err := binary.Write(writer, binary.LittleEndian, c); err != nil {
			return err
		}

		if err := writer.Flush(); err != nil {
			//if err == io.EOF {
			//	break
			//} else {
			return err
			//}
		}
	}

	return nil
}

func (s *Server) GetBroadcasts(overhead, limit int) [][]byte {
	buffer := make(Buffer, 0, limit-overhead)
	writer := bufio.NewWriter(&buffer)

	s.l.RLock()
	defer s.l.RUnlock()

	if err := s.writeCounters(writer, s.m.Id); err != nil {
		s.logger.Printf("[ERR] %v", err)
		return nil
	}

	return [][]byte{buffer}
}

func (s *Server) LocalState(join bool) []byte {
	buffer := bytes.Buffer{}
	writer := bufio.NewWriter(&buffer)

	s.l.RLock()
	defer s.l.RUnlock()

	if err := binary.Write(writer, binary.LittleEndian, uint32(len(s.members))); err != nil {
		s.logger.Printf("[ERR] %v", err)
		return nil
	}

	for id, st := range s.members {
		if err := binary.Write(writer, binary.LittleEndian, id); err != nil {
			s.logger.Printf("[ERR] %v", err)
			return nil
		}

		if err := binary.Write(writer, binary.LittleEndian, st); err != nil {
			s.logger.Printf("[ERR] %v", err)
			return nil
		}
	}

	if err := writer.Flush(); err != nil {
		s.logger.Printf("[ERR] %v", err)
		return nil
	}

	for id := range s.replicas {
		if err := s.writeCounters(writer, id); err != nil {
			s.logger.Printf("[ERR] %v", err)
			return nil
		}
	}

	return buffer.Bytes()
}

func (s *Server) MergeRemoteState(buf []byte, join bool) {
	reader := bufio.NewReader(bytes.NewReader(buf))

	var l uint32
	if err := binary.Read(reader, binary.LittleEndian, &l); err != nil {
		if err == io.EOF {
			return
		} else {
			s.logger.Printf("[ERR] %v", err)
			return
		}
	}

	members := make(map[ID]*State, l)

	for i := uint32(0); i < l; i++ {
		var id ID
		if err := binary.Read(reader, binary.LittleEndian, &id); err != nil {
			if err == io.EOF {
				break
			} else {
				s.logger.Printf("[ERR] %v", err)
				return
			}
		}

		var st State
		if err := binary.Read(reader, binary.LittleEndian, &st); err != nil {
			s.logger.Printf("[ERR] %v", err)
		}

		members[id] = &st
	}

	s.l.Lock()
	defer s.l.Unlock()

	for id, state := range members {
		if x, ok := s.members[id]; !ok || state.When > x.When {
			s.members[id] = state
		}
	}

	go s.updateConsistent()

	for {
		if err := s.readCounters(reader); err != nil {
			if err != io.EOF {
				s.logger.Printf("[ERR] %v", err)
			}

			break
		}
	}
}

func (s *Server) NotifyJoin(node *memberlist.Node) {
	m := &Meta{}
	if err := m.Decode(node.Meta); err != nil {
		s.logger.Printf("[ERR] %v", err)
		return
	}

	s.l.Lock()
	defer s.l.Unlock()

	s.members[m.Id] = &State{
		Active: 1 - m.Leaving,
		When:   time.Now().UTC().Unix(),
	}

	s.logger.Printf("[INFO] %s joined", node.Name)
}

func (s *Server) NotifyLeave(node *memberlist.Node) {
	m := &Meta{}
	if err := m.Decode(node.Meta); err != nil {
		s.logger.Printf("[ERR] %v", err)
		return
	}

	if m.Leaving == 0 {
		s.logger.Printf("[WARNING] %s is not reachable", node.Name)
	} else {
		s.l.Lock()
		defer s.l.Unlock()

		s.members[m.Id] = &State{
			Active: 0,
			When:   time.Now().UTC().Unix(),
		}

		s.logger.Printf("[INFO] %s left", node.Name)
	}

	go s.updateConsistent()
}

func (s *Server) NotifyUpdate(node *memberlist.Node) {
	go s.updateConsistent()
}

func (s *Server) get(name string) (float64, bool) {
	value := float64(0)

	s.l.RLock()
	defer s.l.RUnlock()

	for id, counters := range s.replicas {
		s.logger.Printf("%s", id)
		if v, ok := counters[name]; ok {
			s.logger.Printf("%#v", v)
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
		c.Down -= diff // diff is negative so -= adds it.
	}
}

func (s *Server) reset(name string) {
	s.l.Lock()
	defer s.l.Unlock()

	for id, counters := range s.replicas {
		if id == s.m.Id {
			if c, ok := s.counters[name]; ok {
				c.Up = 0
				c.Down = 0
				c.Revision += 1
			} else {
				s.counters[name] = &Counter{
					Revision: 1,
				}
			}
		} else {
			delete(counters, name)
		}
	}
}

func (s *Server) handle(conn net.Conn) {
	defer func() {
		if err := recover(); err != nil {
			s.logger.Printf("[ERR] %v", err)
		}

		if err := conn.Close(); err != nil {
			s.logger.Printf("[ERR] %v", err)
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

				s.replicas = make(map[ID]map[string]*Counter, 0)
				s.replicas[s.m.Id] = make(map[string]*Counter, 0)
				s.counters = s.replicas[s.m.Id]
				s.members = map[ID]*State{
					s.m.Id: &State{
						Active: 1,
						When:   time.Now().UTC().Unix(),
					},
				}
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

func (s *Server) listen(socket net.Listener) {
	defer func() {
		if err := socket.Close(); err != nil {
			s.logger.Printf("[ERR] %v", err)
		}

		// TODO: We need to close all open connections.

		s.logger.Printf("[DEBUG] stopped listening for clients")

		s.stopped.Done()
	}()

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
				s.logger.Printf("[ERR] %v", err)
				continue
			}
		}

		go s.handle(conn)
	}
}

func (s *Server) Start() error {
	if s.Config.LogOutput == nil {
		s.Config.LogOutput = os.Stderr
	}

	s.logger = log.New(s.Config.LogOutput, "", log.LstdFlags)

	var err error
	s.memberlist, err = memberlist.Create(s.Config)
	if err != nil {
		return err
	}

	s.stopped.Add(1)

	socket, err := net.Listen("tcp", s.client)
	if err != nil {
		return err
	}

	go s.listen(socket)

	return nil
}

func (s *Server) Stop() error {
	s.stop <- struct{}{}

	s.m.Leaving = 1

	s.memberlist.UpdateNode(10 * time.Second)

	time.Sleep(2 * time.Second)

	if err := s.memberlist.Leave(10 * time.Second); err != nil {
		return err
	}

	if err := s.memberlist.Shutdown(); err != nil {
		return err
	}

	s.stopped.Wait()

	return nil
}

func (s *Server) Kill() error {
	s.stop <- struct{}{}

	if err := s.memberlist.Shutdown(); err != nil {
		return err
	}

	return nil
}

func New(bind, client string) *Server {
	s := Server{
		client:     client,
		m:          Meta{},
		stop:       make(chan struct{}, 0),
		consistent: true,
		replicas:   make(map[ID]map[string]*Counter, 0),
		members:    make(map[ID]*State, 0),
	}

	s.m.Id.Randomize()

	s.members[s.m.Id] = &State{
		Active: 1,
		When:   time.Now().UTC().Unix(),
	}

	s.replicas[s.m.Id] = make(map[string]*Counter, 0)
	s.counters = s.replicas[s.m.Id]

	s.Config = memberlist.DefaultWANConfig()
	s.Config.Delegate = &s
	s.Config.Events = &s
	s.Config.Name = bind
	s.Config.BindAddr, s.Config.BindPort = splitHostPort(bind)
	s.Config.AdvertiseAddr = s.Config.BindAddr
	s.Config.AdvertisePort = s.Config.BindPort

	s.Config.SuspicionMult = 2
	s.Config.ProbeInterval = 2 * time.Second
	s.Config.ProbeTimeout = 5 * time.Second
	s.Config.GossipNodes = 6
	s.Config.GossipInterval = 500 * time.Millisecond

	return &s
}
