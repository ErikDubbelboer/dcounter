package main

import (
	"database/sql"
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

	"github.com/spotmx/dcounter/proto"
	"github.com/hashicorp/memberlist"

	_ "github.com/mattn/go-sqlite3"
)


type Server struct {
	m *memberlist.Memberlist

	db *sql.DB

	l sync.RWMutex

	stop chan struct{}

	counters map[string]float64
}


func (s *Server) NodeMeta(limit int) []byte {
	return nil
}

func (s *Server) NotifyMsg([]byte) {
}

func (s *Server) GetBroadcasts(overhead, limit int) [][]byte {
	return nil
}

func (s *Server) LocalState(join bool) []byte {
	return nil
}

func (s *Server) MergeRemoteState(buf []byte, join bool) {
}

func (s *Server) get(name string) float64 {
	return 0
}

func (s *Server) inc(name string, diff float64) {
}

func (s *Server) handle(conn net.Conn) {
	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
		}

		if err := conn.Close(); err != nil {
			log.Println(err)
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
			} else if err := p.Write("RET", []string{strconv.FormatFloat(s.get(args[0]), 'f', -1, 64)}); err != nil {
				panic(err)
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
		case "JOIN":
			if _, err := s.m.Join(args); err != nil {
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

/*func create() {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS counters (
		  id    TEXT PRIMARY KEY,
			amount FLOAT
		)
	`); err != nil {
		panic(err)
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS changes (
		  id         TEXT,
			replica_id INTEGER,
			diff       FLOAT,
			PRIMARY KEY (id, replica_id)
		)
	`); err != nil {
		panic(err)
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS replicas (
		  id   INTEGER PRIMARY KEY,
			host TEXT
		)
	`); err != nil {
		panic(err)
	}
}

func load() {
	if rows, err := db.Query(`
		SELECT id, amount
		FROM counters
	`); err != nil {
		panic(err)
	} else {
		for rows.Next() {
			var id string
			var amount float64

			if err := rows.Scan(&id, &amount); err != nil {
				panic(err)
			}

			counters[id] = amount
		}
	}

	if rows, err := db.Query(`
		SELECT id, replica_id, diff
		FROM changes
	`); err != nil {
		panic(err)
	} else {
		for rows.Next() {
			var id string
			var replica uint32
			var diff float64

			if err := rows.Scan(&id, &replica, &diff); err != nil {
				panic(err)
			}

			if _, ok := changes[replica]; !ok {
				changes[replica] = make(map[string]float64, 0)
			}

			changes[replica][id] = diff
		}
	}

	if rows, err := db.Query(`
		SELECT id, host
		FROM replicas
	`); err != nil {
		panic(err)
	} else {
		for rows.Next() {
			var id uint32
			var host string

			if err := rows.Scan(&id, &host); err != nil {
				panic(err)
			}

			replicas[host] = id
		}
	}
}*/

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
	defer socket.Close()

	//create()
	//load()

	defer func() {
		//flush()

		// TODO: We need to close all open connections.

		log.Println("stopped")

		close(s.stop)
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
				log.Println(err)
				continue
			}
		}

		go s.handle(conn)
	}
}

func splitHostPort(hostport string) (string, int) {
	addr, portStr, err := net.SplitHostPort(hostport)
	if err != nil {
		panic(err)
	}

	port, err := strconv.ParseInt(portStr, 10, 32)
	if err != nil {
		panic(err)
	}

	return addr, int(port)
}

func main() {
	bind := flag.String("bind", "0.0.0.0:9373", "Sets the bind address for cluster communication")
	client := flag.String("client", "0.0.0.0:9374", "Sets the address to bind for client access.")
	dbName := flag.String("db", "db.sqlite", "persist to this file")
	flag.Parse()

	s := Server{
	  stop: make(chan struct{}, 0),
		counters: make(map[string]float64, 0),
	}

	config := memberlist.DefaultWANConfig()
	config.AdvertiseAddr = "127.0.0.1"

	config.BindAddr, config.BindPort = splitHostPort(*bind)

	var err error
	s.m, err = memberlist.Create(config)
	if err != nil {
		panic(err)
	}

	go s.start(*client, *dbName)

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGQUIT)

	<-c

	if err := s.m.Leave(time.Second); err != nil {
		panic(err)
	}

	if err := s.m.Shutdown(); err != nil {
		panic(err)
	}

	s.stop <- struct{}{}
	<-s.stop
}
