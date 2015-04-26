package dcounter

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"

	"github.com/atomx/dcounter/proto"
)

type Member struct {
	Name string
	Addr net.IP
}

// API is the interface returned by New and Dial.
// This is an interface so you can easily mock this interface
// in your tests.
// API is NOT goroutine safe!
type API interface {
	// Ping pings the server and returns nil if everything is ok.
	Ping() error

	// Get returns the value of the counter,
	// a bool indicating if the cluster is stable.
	Get(id string) (float64, bool, error)

	// Inc increments or decrements (diff is negative) a counter.
	// The new value is returned.
	Inc(id string, diff float64) (float64, error)

	// Set sets the value of a counter.
	// Set is a heavy operation as it propagates using TCP.
	// Set returns once the value is propagated to all servers.
	// If set returns an error the change might have already been
	// propagated to some servers.
	// Set returns the old value.
	Set(id string, value float64) (float64, error)

	// List returns a map with all counters.
	List() (map[string]float64, error)

	// Join discards all data in the server and joines a cluster.
	Join(hosts []string) error

	// Save returns a json string containing the data for this server.
	// The json can be saved to a file and loaded using --load when starting a server.
	Save() (string, error)

	// Members returns a list of members of the current cluster.
	Members() ([]Member, error)

	// Close closes any open connection.
	// After close any function will just reopen the connection.
	Close() error
}

type api struct {
	network string
	address string

	c net.Conn
	p *proto.Proto
}

func (a *api) connect() error {
	var err error

	if a.c, err = net.Dial(a.network, a.address); err != nil {
		return err
	}

	a.p = proto.New(a.c)

	return nil
}

func (a *api) Ping() error {
	var err error

	for i := 0; i < 2; i++ {
		if a.c == nil {
			if err := a.connect(); err != nil {
				return err
			}
		}

		if err = a.p.Write("PING", []string{}); err != nil {
			a.c = nil
			continue
		} else if cmd, _, err := a.p.Read(); err != nil {
			a.c = nil
			continue
		} else if cmd != "PONG" {
			return fmt.Errorf("PONG expected")
		}
	}

	return err
}

func (a *api) Get(id string) (float64, bool, error) {
	var err error

	for i := 0; i < 2; i++ {
		if a.c == nil {
			if err := a.connect(); err != nil {
				return 0, false, err
			}
		}

		if err = a.p.Write("GET", []string{id}); err != nil {
			a.c = nil
			continue
		} else if cmd, args, err := a.p.Read(); err != nil {
			a.c = nil
			continue
		} else if cmd != "RET" {
			return 0, false, fmt.Errorf("RET expected")
		} else if len(args) != 2 {
			return 0, false, fmt.Errorf("2 arguments expected")
		} else if value, err := strconv.ParseFloat(args[0], 64); err != nil {
			return 0, false, err
		} else if consistent, err := strconv.ParseBool(args[1]); err != nil {
			return 0, false, err
		} else {
			return value, consistent, nil
		}
	}

	return 0, true, err
}

func (a *api) Inc(id string, diff float64) (float64, error) {
	var err error

	for i := 0; i < 2; i++ {
		if a.c == nil {
			if err := a.connect(); err != nil {
				return 0, err
			}
		}

		if err = a.p.Write("INC", []string{id, strconv.FormatFloat(diff, 'f', -1, 64)}); err != nil {
			a.c = nil
			continue
		} else if cmd, args, err := a.p.Read(); err != nil {
			a.c = nil
			continue
		} else if cmd != "RET" {
			return 0, fmt.Errorf("RET expected")
		} else if len(args) != 1 {
			return 0, fmt.Errorf("1 arguments expected")
		} else if value, err := strconv.ParseFloat(args[0], 64); err != nil {
			return 0, err
		} else {
			return value, nil
		}
	}

	return 0, err
}

func (a *api) Set(id string, value float64) (float64, error) {
	var err error

	for i := 0; i < 2; i++ {
		if a.c == nil {
			if err := a.connect(); err != nil {
				return 0, err
			}
		}

		if err = a.p.Write("SET", []string{id, strconv.FormatFloat(value, 'f', -1, 64)}); err != nil {
			a.c = nil
			continue
		} else if cmd, args, err := a.p.Read(); err != nil {
			a.c = nil
			continue
		} else if cmd != "RET" {
			return 0, fmt.Errorf("RET expected")
		} else if len(args) != 1 {
			return 0, fmt.Errorf("1 arguments expected")
		} else if old, err := strconv.ParseFloat(args[0], 64); err != nil {
			return 0, err
		} else {
			return old, nil
		}
	}

	return 0, err
}

func (a *api) List() (map[string]float64, error) {
	var err error

	for i := 0; i < 2; i++ {
		if a.c == nil {
			if err := a.connect(); err != nil {
				return nil, err
			}
		}

		if err = a.p.Write("LIST", []string{}); err != nil {
			a.c = nil
			continue
		} else if cmd, args, err := a.p.Read(); err != nil {
			a.c = nil
			continue
		} else if cmd != "RET" {
			return nil, fmt.Errorf("RET expected")
		} else {
			list := make(map[string]float64, len(args)/2)

			for i := 0; i < len(args); i += 2 {
				if value, err := strconv.ParseFloat(args[i+1], 64); err != nil {
					return nil, err
				} else {
					list[args[i]] = value
				}
			}

			return list, nil
		}
	}

	return nil, err
}

func (a *api) Join(hosts []string) error {
	var err error

	for i := 0; i < 2; i++ {
		if a.c == nil {
			if err := a.connect(); err != nil {
				return err
			}
		}

		if err = a.p.Write("JOIN", hosts); err != nil {
			a.c = nil
			continue
		} else if cmd, _, err := a.p.Read(); err != nil {
			a.c = nil
			continue
		} else if cmd != "OK" {
			return fmt.Errorf("OK expected")
		} else {
			return nil
		}
	}

	return err
}

func (a *api) Save() (string, error) {
	var err error

	for i := 0; i < 2; i++ {
		if a.c == nil {
			if err := a.connect(); err != nil {
				return "", err
			}
		}

		if err = a.p.Write("SAVE", []string{}); err != nil {
			a.c = nil
			continue
		} else if cmd, args, err := a.p.Read(); err != nil {
			a.c = nil
			continue
		} else if cmd != "RET" {
			return "", fmt.Errorf("RET expected")
		} else {
			return args[0], nil
		}
	}

	return "", err
}

func (a *api) Members() ([]Member, error) {
	var err error

	for i := 0; i < 2; i++ {
		if a.c == nil {
			if err := a.connect(); err != nil {
				return nil, err
			}
		}

		var members []Member

		if err = a.p.Write("MEMBERS", []string{}); err != nil {
			a.c = nil
			continue
		} else if cmd, args, err := a.p.Read(); err != nil {
			a.c = nil
			continue
		} else if cmd != "RET" {
			return nil, fmt.Errorf("RET expected")
		} else if err := json.Unmarshal([]byte(args[0]), &members); err != nil {
			return nil, err
		} else {
			return members, nil
		}
	}

	return nil, err
}

func (a *api) Close() error {
	err := a.c.Close()

	a.c = nil
	a.p = nil

	return err
}

// New returns a new API client but does not connect.
// The connection will be made when the first command is run.
func New(network, address string) API {
	return &api{
		network: network,
		address: address,
	}
}

// Dial tries to connect and if successful returns a new API client.
func Dial(network, address string) (API, error) {
	a := New(network, address)

	if err := a.Ping(); err != nil {
		return nil, err
	}

	return a, nil
}
