package server

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strconv"
	"syscall"
	"time"

	"github.com/atomx/dcounter/proto"
	"github.com/atomx/dcounter/server/change"

	api "github.com/atomx/dcounter/api"
)

func (s *Server) broadcastChange(name string, c *Counter) error {
	buffer := bytes.Buffer{}
	writer := bufio.NewWriter(&buffer)

	if err := buffer.WriteByte(changeMessage); err != nil {
		return err
	}

	if _, err := writer.WriteString(s.Config.Name + "\n"); err != nil {
		return err
	}

	if _, err := writer.WriteString(name + "\n"); err != nil {
		return err
	}

	if err := binary.Write(writer, binary.LittleEndian, c); err != nil {
		return err
	}

	if err := writer.Flush(); err != nil {
		return err
	}

	s.changes.QueueBroadcast(change.New(name, buffer.Bytes()))

	return nil
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

func (s *Server) inc(name string, diff float64) (old float64, err error) {
	s.l.Lock()
	defer s.l.Unlock()

	c, ok := s.replicas[s.Config.Name][name]
	if !ok {
		c = &Counter{}
		s.replicas[s.Config.Name][name] = c
	} else {
		old = c.Up - c.Down
	}

	if diff > 0 {
		c.Up += diff
	} else {
		c.Down -= diff // diff is negative so -= adds it.
	}

	return old, s.broadcastChange(name, c)
}

func (s *Server) set(name string, value float64) (old float64, err error) {
	var up, down float64

	if value > 0 {
		up = value
	} else {
		down = -value
	}

	buffer := bytes.Buffer{}
	writer := bufio.NewWriter(&buffer)

	buffer.WriteByte(resetMessage)

	if _, err := writer.WriteString(name + "\n"); err != nil {
		return 0, err
	}

	if err := writer.Flush(); err != nil {
		return 0, err
	}

	bl := buffer.Len()
	nodes := s.memberlist.Members()

	for _, node := range nodes {
		if node.Name == s.Config.Name {
			continue
		}

		if err := func() error {
			s.l.RLock()
			defer s.l.RUnlock()

			counters, ok := s.replicas[node.Name]
			if !ok {
				return nil
			}

			counter, ok := counters[name]
			if !ok {
				return nil
			}

			buffer.Truncate(bl)

			s.logger.Printf("[DEBUG] write %v", counter)

			if err := binary.Write(writer, binary.LittleEndian, counter); err != nil {
				return err
			}

			return nil
		}(); err != nil {
			return 0, err
		}

		if err := writer.Flush(); err != nil {
			return 0, err
		}

		if err := s.memberlist.SendToTCP(node, buffer.Bytes()); err != nil {
			return 0, err
		}
	}

	s.l.Lock()
	defer s.l.Unlock()

	c, ok := s.replicas[s.Config.Name][name]
	if ok {
		old = c.Up - c.Down

		c.Up = up
		c.Down = down
		c.Revision += 1
	} else {
		c = &Counter{
			Up:   up,
			Down: down,
		}
		s.replicas[s.Config.Name][name] = c
	}

	return old, s.broadcastChange(name, c)
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

	// Don't return counters which have a value of 0.
	for name, value := range l {
		if value == 0 {
			delete(l, name)
		}
	}

	return l
}

func (s *Server) save() (string, error) {
	s.l.RLock()
	defer s.l.RUnlock()

	data, err := json.Marshal(s.replicas)
	return string(data), err
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
			} else if args[0] == "" {
				if err := p.Error(fmt.Errorf(`Invalid counter name ""`)); err != nil {
					panic(err)
				}
			} else if amount, err := strconv.ParseFloat(args[1], 64); err != nil {
				if err := p.Error(err); err != nil {
					panic(err)
				}
			} else if amount, err := s.inc(args[0], amount); err != nil {
				if err := p.Error(err); err != nil {
					panic(err)
				}
			} else if err := p.Write("RET", []string{strconv.FormatFloat(amount, 'f', -1, 64)}); err != nil {
				panic(err)
			}
		case "SET":
			if len(args) < 2 {
				if err := p.Error(fmt.Errorf("SET requires exactly two argument")); err != nil {
					panic(err)
				}
			} else if args[0] == "" {
				if err := p.Error(fmt.Errorf(`Invalid counter name ""`)); err != nil {
					panic(err)
				}
			} else if value, err := strconv.ParseFloat(args[1], 64); err != nil {
				if err := p.Error(err); err != nil {
					panic(err)
				}
			} else if amount, err := s.set(args[0], value); err != nil {
				if err := p.Error(err); err != nil {
					panic(err)
				}
			} else if err := p.Write("RET", []string{strconv.FormatFloat(amount, 'f', -1, 64)}); err != nil {
				panic(err)
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
		case "SAVE":
			if data, err := s.save(); err != nil {
				if err := p.Error(err); err != nil {
					panic(err)
				}
			} else if err := p.Write("RET", []string{data}); err != nil {
				panic(err)
			}
		case "MEMBERS":
			members := s.memberlist.Members()

			ret := make([]api.Member, 0, len(members))

			for _, member := range members {
				ret = append(ret, api.Member{
					Name: member.Name,
					Addr: member.Addr,
				})
			}

			if data, err := json.Marshal(ret); err != nil {
				if err := p.Error(err); err != nil {
					panic(err)
				}
			} else if err := p.Write("RET", []string{string(data)}); err != nil {
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
