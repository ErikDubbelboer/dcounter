package server

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"strconv"
	"time"

	"github.com/atomx/memberlist"
)

func (s *Server) NodeMeta(limit int) []byte {
	return s.m.Encode(limit)
}

func (s *Server) NotifyMsg(msg []byte) {
	reader := bufio.NewReader(bytes.NewReader(msg))

	t, err := reader.ReadByte()
	if err != nil {
		s.logger.Printf("[ERR] %v", err)
		return
	}

	s.l.Lock()
	defer s.l.Unlock()

	if t == resetMessage {
		if err := s.readReset(reader); err != nil {
			s.logger.Printf("[ERR] %v", err)
		}
	} else if t == changeMessage {
		if err := s.readChange(reader); err != nil {
			s.logger.Printf("[ERR] %v", err)
		}
	} else {
		s.logger.Printf("[ERR] unknown message type (%d)", t)
	}
}

func (s *Server) GetBroadcasts(overhead, limit int) [][]byte {
	s.l.RLock()
	defer s.l.RUnlock()

	s.logger.Printf("[DEBUG] %d messages queued", s.changes.NumQueued())

	return s.changes.GetBroadcasts(overhead, limit)
}

// LocalState is called when memberlist wants to do a full TCP
// state transfer to one random node.
// Since it's only one random node we shouldn't reset s.changes here.
func (s *Server) LocalState(join bool) []byte {
	buffer := bytes.Buffer{}
	enc := gob.NewEncoder(&buffer)

	s.l.RLock()
	defer s.l.RUnlock()

	if err := enc.Encode(s.members); err != nil {
		s.logger.Printf("[ERR] %v", err)
		return nil
	}

	if err := enc.Encode(s.replicas); err != nil {
		s.logger.Printf("[ERR] %v", err)
		return nil
	}

	s.logger.Printf("[DEBUG] %d byte local state", buffer.Len())

	return buffer.Bytes()
}

func (s *Server) MergeRemoteState(buf []byte, join bool) {
	dec := gob.NewDecoder(bytes.NewReader(buf))

	var members states
	if err := dec.Decode(&members); err != nil {
		s.logger.Printf("[ERR] %v", err)
		return
	}

	var replicas Counters
	if err := dec.Decode(&replicas); err != nil {
		s.logger.Printf("[ERR] %v", err)
		return
	}

	s.l.Lock()
	defer s.l.Unlock()

	for memberName, state := range members {
		if x, ok := s.members[memberName]; !ok || state.When > x.When {
			s.members[memberName] = state
		}
	}

	s.members[s.Config.Name] = &state{
		Active: 1,
		When:   time.Now().UTC().Unix(),
	}

	for memberName, counters := range replicas {
		if _, ok := s.members[memberName]; !ok {
			continue
		}

		if _, ok := s.replicas[memberName]; !ok {
			s.replicas[memberName] = make(map[string]*Counter, 0)
		}

		for name, c := range counters {
			if oc, ok := s.replicas[memberName][name]; !ok || c.Revision > oc.Revision {
				s.replicas[memberName][name] = c
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

	go s.updateConsistent()
}

func (s *Server) NotifyJoin(node *memberlist.Node) {
	s.l.Lock()
	defer s.l.Unlock()

	s.members[node.Name] = &state{
		Active: 1,
		When:   time.Now().UTC().Unix(),
	}

	addr := node.Addr.String() + ":" + strconv.FormatUint(uint64(node.Port), 10)
	delete(s.reconnects, addr)

	s.logger.Printf("[INFO] %s joined", node.Name)
}

func (s *Server) NotifyLeave(node *memberlist.Node) {
	m := &meta{}
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
		s.members[node.Name] = &state{
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
