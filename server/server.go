package server

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/hashicorp/memberlist"
)

const (
	resetMessage = iota
	changeMessage
)

type Counters map[string]map[string]*Counter

type states map[string]*state

type Server struct {
	client string

	Config *memberlist.Config

	memberlist *memberlist.Memberlist

	m meta

	stop    chan struct{}
	stopped sync.WaitGroup

	l sync.RWMutex

	consistent bool

	replicas Counters

	changes memberlist.TransmitLimitedQueue

	members states

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

	s.l.Lock()
	defer s.l.Unlock()

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

func (s *Server) readChange(reader *bufio.Reader) error {
	var memberName string
	if n, err := reader.ReadString('\n'); err != nil {
		return err
	} else {
		// Strip off the '\n'.
		memberName = n[:len(n)-1]
	}

	// If we don't know this member we can ignore it.
	if _, ok := s.members[memberName]; !ok {
		return fmt.Errorf("change for unknown member %s", memberName)
	}

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

	if _, ok := s.replicas[memberName]; !ok {
		s.replicas[memberName] = make(map[string]*Counter, 0)
	}

	if oc, ok := s.replicas[memberName][name]; !ok || c.Revision > oc.Revision {
		s.replicas[memberName][name] = &c
	} else {
		if c.Up > oc.Up {
			oc.Up = c.Up
		}
		if c.Down > oc.Down {
			oc.Down = c.Down
		}
	}

	return nil
}

func (s *Server) readReset(reader *bufio.Reader) error {
	var name string
	if s, err := reader.ReadString('\n'); err != nil {
		return err
	} else {
		// Strip off the '\n'.
		name = s[:len(s)-1]
	}

	var c Counter
	if err := binary.Read(reader, binary.LittleEndian, &c); err != nil {
		return err
	}

	if oc, ok := s.replicas[s.Config.Name][name]; ok && c.Revision == oc.Revision {
		oc.Up -= c.Up
		oc.Down -= c.Down
		oc.Revision += 1

		if err := s.broadcastChange(name, oc); err != nil {
			return err
		}
	}

	return nil
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

	s.members[s.Config.Name] = &state{
		Active: 1,
		When:   time.Now().UTC().Unix(),
	}

	if s.Config.LogOutput == nil {
		s.Config.LogOutput = os.Stderr
	}

	s.logger = log.New(s.Config.LogOutput, "", log.LstdFlags|log.Lshortfile)

	var err error
	s.memberlist, err = memberlist.Create(s.Config)
	if err != nil {
		return err
	}

	s.changes = memberlist.TransmitLimitedQueue{
		NumNodes:       s.memberlist.NumMembers,
		RetransmitMult: 4,
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
	s.changes = memberlist.TransmitLimitedQueue{
		NumNodes:       s.memberlist.NumMembers,
		RetransmitMult: 4,
	}
	s.members = map[string]*state{
		s.Config.Name: &state{
			Active: 1, // Start in an inconsistent state until the first push/pull.
			When:   time.Now().UTC().Unix(),
		},
	}
	s.reconnects = make(map[string]struct{}, 0)

	s.l.Unlock()

	_, err := s.memberlist.Join(hosts)
	return err
}

func (s *Server) Load(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	decoder := json.NewDecoder(f)

	s.l.Lock()
	defer s.l.Unlock()

	err = decoder.Decode(&s.replicas)

	if _, ok := s.replicas[s.Config.Name]; !ok {
		s.replicas[s.Config.Name] = make(map[string]*Counter, 0)
	}

	return err
}

func (s *Server) Save(filename string) (err error) {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		if nerr := f.Close(); nerr != nil && err != nil {
			err = nerr
		}
	}()

	encoder := json.NewEncoder(f)

	s.l.Lock()
	defer s.l.Unlock()

	return encoder.Encode(s.replicas)
}

func New(name, bind, advertise, client string) (*Server, error) {
	s := Server{
		client:     client,
		m:          meta{},
		stop:       make(chan struct{}, 0),
		consistent: true,
		replicas:   make(map[string]map[string]*Counter, 0),
		members:    make(map[string]*state, 0),
		reconnects: make(map[string]struct{}, 0),
	}

	s.Config = memberlist.DefaultWANConfig()
	s.Config.Delegate = &s
	s.Config.Events = &s
	s.Config.Name = name

	s.Config.SuspicionMult = 2
	s.Config.PushPullInterval = 60 * time.Second
	s.Config.ProbeInterval = 2 * time.Second
	s.Config.ProbeTimeout = 4 * time.Second
	s.Config.GossipNodes = 4
	s.Config.GossipInterval = 500 * time.Millisecond

	ip, port, err := splitHostPort(bind, 9373)
	if err != nil {
		return nil, err
	}

	s.Config.BindAddr = ip
	s.Config.BindPort = port

	ip, port, err = splitHostPort(advertise, port)
	if err != nil {
		return nil, err
	}

	s.Config.AdvertiseAddr = ip
	s.Config.AdvertisePort = port

	return &s, nil
}
