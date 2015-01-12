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

	counters  map[string]*Counter
	replicas  map[string]map[string]*Counter
	revisions map[string]uint32

	changes Changes

	members map[string]*State

	reconnects map[string]struct{}

	logger *log.Logger
}

var (
	emptyCounter = &Counter{}
)

// We are consitent if all the members in members that are active
// have a node in our memberlist.
func (s *Server) updateConsistent() {
	nodes := s.memberlist.Members()
	members := make(map[string]struct{}, len(nodes))

	for _, node := range nodes {
		members[node.Name] = struct{}{}
	}

	s.l.RLock()
	defer s.l.RUnlock()

	for name, state := range s.members {
		if state.Active == 1 {
			if _, ok := members[name]; !ok {
				s.logger.Printf("[WARNING] inconsistent because %s is missing", name)
				s.consistent = false
				return
			}
		}
	}

	s.consistent = true
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

	var memberName string
	if s, err := reader.ReadString('\n'); err != nil {
		return err
	} else {
		// Strip off the '\n'.
		memberName = s[:len(s)-1]
	}

	for i := uint32(0); i < l; i++ {
		var name string
		if n, err := reader.ReadString('\n'); err != nil {
			return err
		} else {
			// Strip off the '\n'.
			name = n[:len(n)-1]
		}

		var r uint32
		if err := binary.Read(reader, binary.LittleEndian, &r); err != nil {
			return err
		}

		var c Counter
		if err := binary.Read(reader, binary.LittleEndian, &c); err != nil {
			return err
		}

		// If we don't know this id we can ignore it.
		if _, ok := s.members[memberName]; !ok {
			continue
		}

		if _, ok := s.replicas[memberName]; !ok {
			s.replicas[memberName] = make(map[string]*Counter, 0)
		}

		if or := s.revisions[name]; r > or {
			s.revisions[name] = r

			for _, counters := range s.replicas {
				delete(counters, name)
			}

			s.replicas[memberName][name] = &c
		} else if or == r {
			if oc, ok := s.replicas[memberName][name]; !ok {
				s.replicas[memberName][name] = &c
			} else {
				if c.Up > oc.Up {
					oc.Up = c.Up
				}
				if c.Down > oc.Down {
					oc.Down = c.Down
				}
			}
		}
	}

	return nil
}

func (s *Server) NotifyMsg(msg []byte) {
	go func() {
		reader := bufio.NewReader(bytes.NewReader(msg))

		s.l.Lock()
		defer s.l.Unlock()

		var memberName string
		if n, err := reader.ReadString('\n'); err != nil {
			if err != io.EOF {
				s.logger.Printf("[ERR] %v", err)
			}

			return
		} else {
			// Strip off the '\n'.
			memberName = n[:len(n)-1]
		}

		for {
			var name string
			if n, err := reader.ReadString('\n'); err != nil {
				if err != io.EOF {
					s.logger.Printf("[ERR] %v", err)
				}

				break
			} else {
				// Strip off the '\n'.
				name = n[:len(n)-1]
			}

			var r uint32
			if err := binary.Read(reader, binary.LittleEndian, &r); err != nil {
				if err != io.EOF {
					s.logger.Printf("[ERR] %v", err)
				}

				break
			}

			var c Counter
			if err := binary.Read(reader, binary.LittleEndian, &c); err != nil {
				if err != io.EOF {
					s.logger.Printf("[ERR] %v", err)
				}

				break
			}

			// If we don't know this id we can ignore it.
			if _, ok := s.members[memberName]; !ok {
				continue
			}

			if _, ok := s.replicas[memberName]; !ok {
				s.replicas[memberName] = make(map[string]*Counter, 0)
			}

			if or := s.revisions[name]; r > or {
				s.revisions[name] = r

				for _, counters := range s.replicas {
					delete(counters, name)
				}

				s.replicas[memberName][name] = &c
			} else if or == r {
				if oc, ok := s.replicas[memberName][name]; !ok {
					s.replicas[memberName][name] = &c
				} else {
					if c.Up > oc.Up {
						oc.Up = c.Up
					}
					if c.Down > oc.Down {
						oc.Down = c.Down
					}
				}
			}
		}
	}()
}

func (s *Server) writeCounters(writer *bufio.Writer, memberName string) error {
	l := uint32(len(s.replicas[memberName]))

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

	if _, err := writer.WriteString(memberName + "\n"); err != nil {
		return err
	}

	for name, c := range s.replicas[memberName] {
		if _, err := writer.WriteString(name + "\n"); err != nil {
			return err
		}

		if err := binary.Write(writer, binary.LittleEndian, s.revisions[name]); err != nil {
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
	s.l.RLock()
	defer s.l.RUnlock()

	if len(s.changes) == 0 {
		return nil
	}

	buffer := make(Buffer, 0, limit-overhead)
	writer := bufio.NewWriter(&buffer)

	if _, err := writer.WriteString(s.Config.Name + "\n"); err != nil {
		s.logger.Printf("[ERR] %v", err)
		return nil
	}

	if err := writer.Flush(); err != nil {
		s.logger.Printf("[ERR] %v", err)
		return nil
	}

	for {
		name := s.changes.Peek()

		if name == "" {
			break
		}

		if _, err := writer.WriteString(name + "\n"); err != nil {
			s.logger.Printf("[ERR] %v", err)
			break
		}

		if err := binary.Write(writer, binary.LittleEndian, s.revisions[name]); err != nil {
			s.logger.Printf("[ERR] %v", err)
			break
		}

		c, ok := s.counters[name]
		if !ok {
			c = emptyCounter
		}

		if err := binary.Write(writer, binary.LittleEndian, c); err != nil {
			s.logger.Printf("[ERR] %v", err)
			break
		}

		if err := writer.Flush(); err != nil {
			if err != io.EOF {
				s.logger.Printf("[ERR] %v", err)
			}

			break
		}

		s.changes.Pop()
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

	for memberName, st := range s.members {
		if _, err := writer.WriteString(memberName + "\n"); err != nil {
			s.logger.Printf("[ERR] %v", err)
			return nil
		}

		if err := binary.Write(writer, binary.LittleEndian, st); err != nil {
			s.logger.Printf("[ERR] %v", err)
			return nil
		}
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

	s.l.Lock()
	defer s.l.Unlock()

	for i := uint32(0); i < l; i++ {
		var memberName string
		if name, err := reader.ReadString('\n'); err != nil {
			if err == io.EOF {
				break
			} else {
				s.logger.Printf("[ERR] %v", err)
				return
			}
		} else {
			// Strip off the '\n'.
			memberName = name[:len(name)-1]
		}

		var state State
		if err := binary.Read(reader, binary.LittleEndian, &state); err != nil {
			s.logger.Printf("[ERR] %v", err)
			return
		}

		if x, ok := s.members[memberName]; !ok || state.When > x.When {
			s.members[memberName] = &state
		}
	}

	s.members[s.Config.Name] = &State{
		Active: 1,
		When:   time.Now().UTC().Unix(),
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
	s.l.Lock()
	defer s.l.Unlock()

	s.members[node.Name] = &State{
		Active: 1,
		When:   time.Now().UTC().Unix(),
	}

	addr := node.Addr.String() + ":" + strconv.FormatUint(uint64(node.Port), 10)
	delete(s.reconnects, addr)

	s.logger.Printf("[INFO] %s joined", node.Name)
}

func (s *Server) NotifyLeave(node *memberlist.Node) {
	m := &Meta{}
	if err := m.Decode(node.Meta); err != nil {
		s.logger.Printf("[ERR] %v", err)
		return
	}

	addr := node.Addr.String() + ":" + strconv.FormatUint(uint64(node.Port), 10)

	s.l.Lock()
	defer s.l.Unlock()

	if m.Leaving == 0 {
		s.reconnects[addr] = struct{}{}

		s.logger.Printf("[WARNING] %s missing", node.Name)
	} else {
		s.members[node.Name] = &State{
			Active: 0,
			When:   time.Now().UTC().Unix(),
		}

		delete(s.reconnects, addr)

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

	for _, counters := range s.replicas {
		if c, ok := counters[name]; ok {
			value += c.Up - c.Down
		}
	}

	return value, s.consistent
}

func (s *Server) inc(name string, diff float64) {
	s.l.Lock()

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

	s.changes.Add(name)

	s.l.Unlock()
}

func (s *Server) reset(name string) {
	s.l.Lock()

	s.revisions[name] += 1

	for _, counters := range s.replicas {
		if c, ok := counters[name]; ok {
			c.Up = 0
			c.Down = 0
		}
	}

	s.changes.Add(name)

	s.l.Unlock()
}

func (s *Server) list() map[string]float64 {
	l := make(map[string]float64, 0)

	s.l.RLock()
	for _, counters := range s.replicas {
		for name, c := range counters {
			l[name] += c.Up - c.Down
		}
	}
	s.l.RUnlock()

	return l
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
			} else if args[0] == "" {
				if err := p.Error(fmt.Errorf(`Invalid counter name ""`)); err != nil {
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
		case "LIST":
			counters := s.list()
			args := make([]string, 0, len(counters)*2)

			for name, value := range counters {
				args = append(args, name)
				args = append(args, strconv.FormatFloat(value, 'f', -1, 64))
			}

			if err := p.Write("RET", args); err != nil {
				panic(err)
			}
		case "JOIN":
			if err := s.Join(args); err != nil {
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

func (s *Server) reconnect() {
	defer s.stopped.Done()

	for {
		select {
		case <-s.stop:
			return
		default:
		}

		hosts := make([]string, 0)

		s.l.Lock()
		for host := range s.reconnects {
			hosts = append(hosts, host)
		}
		s.l.Unlock()

		if len(hosts) == 0 {
			time.Sleep(time.Second)
			continue
		}

		s.logger.Printf("reconnecting with %v", hosts)

		s.memberlist.Join(hosts)

		// If we miss 1 host  we try every 12 seconds.
		// If we miss 4 hosts we try every 4.5 seconds.
		time.Sleep(2*time.Second + (time.Second*10)/time.Duration(len(hosts)))
		// This timeout is needed for fail_test.go:TestNetwork
		//time.Sleep(time.Second)
	}
}

func (s *Server) Start() error {
	s.replicas[s.Config.Name] = make(map[string]*Counter, 0)
	s.counters = s.replicas[s.Config.Name]

	s.members[s.Config.Name] = &State{
		Active: 1,
		When:   time.Now().UTC().Unix(),
	}

	if s.Config.LogOutput == nil {
		s.Config.LogOutput = os.Stderr
	}

	s.logger = log.New(s.Config.LogOutput, "", log.LstdFlags)

	var err error
	s.memberlist, err = memberlist.Create(s.Config)
	if err != nil {
		return err
	}

	s.stopped.Add(2)

	socket, err := net.Listen("tcp", s.client)
	if err != nil {
		return err
	}

	go s.listen(socket)
	go s.reconnect()

	return nil
}

func (s *Server) Stop() error {
	close(s.stop)

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
	close(s.stop)

	if err := s.memberlist.Shutdown(); err != nil {
		return err
	}

	return nil
}

func (s *Server) Join(hosts []string) error {
	s.l.Lock()

	s.consistent = false

	s.replicas = make(map[string]map[string]*Counter, 0)
	s.replicas[s.Config.Name] = make(map[string]*Counter, 0)
	s.counters = s.replicas[s.Config.Name]
	s.revisions = make(map[string]uint32, 0)
	s.changes = make(Changes, 0)
	s.members = map[string]*State{
		s.Config.Name: &State{
			Active: 1, // Start in an inconsistent state until the first push/pull.
			When:   time.Now().UTC().Unix(),
		},
	}
	s.reconnects = make(map[string]struct{}, 0)

	s.l.Unlock()

	_, err := s.memberlist.Join(hosts)
	return err
}

func New(name, bind, client string) *Server {
	s := Server{
		client:     client,
		m:          Meta{},
		stop:       make(chan struct{}, 0),
		consistent: true,
		replicas:   make(map[string]map[string]*Counter, 0),
		revisions:  make(map[string]uint32, 0),
		changes:    make(Changes, 0),
		members:    make(map[string]*State, 0),
		reconnects: make(map[string]struct{}, 0),
	}

	s.Config = memberlist.DefaultWANConfig()
	s.Config.Delegate = &s
	s.Config.Events = &s
	s.Config.Name = name
	s.Config.BindAddr, s.Config.BindPort = splitHostPort(bind)
	s.Config.AdvertiseAddr = s.Config.BindAddr
	s.Config.AdvertisePort = s.Config.BindPort

	s.Config.SuspicionMult = 2
	s.Config.PushPullInterval = 60 * time.Second
	s.Config.ProbeInterval = 2 * time.Second
	s.Config.ProbeTimeout = 4 * time.Second
	s.Config.GossipNodes = 4
	s.Config.GossipInterval = 500 * time.Millisecond

	return &s
}
