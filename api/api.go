package dcounter

import (
	"fmt"
	"net"
	"strconv"

	"github.com/atomx/dcounter/proto"
)

// API is the interface returned by New and Dial.
// This is an interface so you can easily mock this interface
// in your tests.
type API interface {
	Ping() error
	Get(id string) (float64, bool, error)
	Inc(id string, amount float64) error
	Set(id string, value float64) error
	List() (map[string]float64, error)
	Join(hosts []string) error
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
			return err
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
			return 0, false, err
		} else if cmd != "RET" {
			return 0, false, fmt.Errorf("RET expected")
		} else if len(args) != 2 {
			return 0, false, fmt.Errorf("2 arguments expected")
		} else if amount, err := strconv.ParseFloat(args[0], 64); err != nil {
			return 0, false, err
		} else if consistent, err := strconv.ParseBool(args[1]); err != nil {
			return 0, false, err
		} else {
			return amount, consistent, nil
		}
	}

	return 0, true, err
}

func (a *api) Inc(id string, amount float64) error {
	var err error

	for i := 0; i < 2; i++ {
		if a.c == nil {
			if err := a.connect(); err != nil {
				return err
			}
		}

		if err = a.p.Write("INC", []string{id, strconv.FormatFloat(amount, 'f', -1, 64)}); err != nil {
			a.c = nil
			continue
		} else if cmd, _, err := a.p.Read(); err != nil {
			return err
		} else if cmd != "OK" {
			return fmt.Errorf("OK expected")
		} else {
			return nil
		}
	}

	return err
}

func (a *api) Set(id string, value float64) error {
	var err error

	for i := 0; i < 2; i++ {
		if a.c == nil {
			if err := a.connect(); err != nil {
				return err
			}
		}

		if err = a.p.Write("SET", []string{id, strconv.FormatFloat(value, 'f', -1, 64)}); err != nil {
			a.c = nil
			continue
		} else if cmd, _, err := a.p.Read(); err != nil {
			return err
		} else if cmd != "OK" {
			return fmt.Errorf("OK expected")
		} else {
			return nil
		}
	}

	return err
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
			return nil, err
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
			return err
		} else if cmd != "OK" {
			return fmt.Errorf("OK expected")
		} else {
			return nil
		}
	}

	return err
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
