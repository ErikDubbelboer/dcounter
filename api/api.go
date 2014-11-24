package api

import (
	"fmt"
	"net"
	"strconv"

	"github.com/spotmx/dcounter/proto"
)

type API struct {
	network string
	address string

	c net.Conn
	p *proto.Proto
}

func (api *API) connect() error {
	var err error

	if api.c, err = net.Dial(api.network, api.address); err != nil {
		return err
	}

	api.p = proto.New(api.c)

	return nil
}

func (api *API) Ping() error {
	var err error

	for i := 0; i < 2; i++ {
		if api.c == nil {
			if err := api.connect(); err != nil {
				return err
			}
		}

		if err = api.p.Write("PING", []string{}); err != nil {
			continue
		} else if cmd, _, err := api.p.Read(); err != nil {
			return err
		} else if cmd != "PONG" {
			return fmt.Errorf("PONG expected")
		}
	}

	return err
}

func (api *API) Get(id string) (float64, error) {
	var err error

	for i := 0; i < 2; i++ {
		if api.c == nil {
			if err := api.connect(); err != nil {
				return 0, err
			}
		}

		if err = api.p.Write("GET", []string{id}); err != nil {
			continue
		} else if cmd, args, err := api.p.Read(); err != nil {
			return 0, err
		} else if cmd != "RET" {
			return 0, fmt.Errorf("RET expected")
		} else if len(args) != 1 {
			return 0, fmt.Errorf("1 argument expected")
		} else if amount, err := strconv.ParseFloat(args[0], 64); err != nil {
			return 0, err
		} else {
			return amount, nil
		}
	}

	return 0, err
}

func (api *API) Inc(id string, amount float64) error {
	var err error

	for i := 0; i < 2; i++ {
		if api.c == nil {
			if err := api.connect(); err != nil {
				return err
			}
		}

		if err = api.p.Write("INC", []string{id, strconv.FormatFloat(amount, 'f', -1, 64), "true"}); err != nil {
			continue
		} else if cmd, _, err := api.p.Read(); err != nil {
			return err
		} else if cmd != "OK" {
			return fmt.Errorf("OK expected")
		} else {
			return nil
		}
	}

	return err
}

func (api *API) Replicate(op string, host string) error {
	var err error

	for i := 0; i < 2; i++ {
		if api.c == nil {
			if err := api.connect(); err != nil {
				return err
			}
		}

		if err = api.p.Write("REPLICATE", []string{op, host}); err != nil {
			continue
		} else if cmd, _, err := api.p.Read(); err != nil {
			return err
		} else if cmd != "OK" {
			return fmt.Errorf("OK expected")
		} else {
			return nil
		}
	}

	return err
}

func (api *API) Sync() error {
	var err error

	for i := 0; i < 2; i++ {
		if api.c == nil {
			if err := api.connect(); err != nil {
				return err
			}
		}

		if err = api.p.Write("SYNC", []string{}); err != nil {
			continue
		} else if cmd, _, err := api.p.Read(); err != nil {
			return err
		} else if cmd != "OK" {
			return fmt.Errorf("OK expected")
		} else {
			return nil
		}
	}

	return err
}

func (api *API) Close() error {
	err := api.c.Close()

	api.c = nil
	api.p = nil

	return err
}

func Dial(network, address string) (*API, error) {
	api := &API{
		network: network,
		address: address,
	}

	if err := api.Ping(); err != nil {
		return nil, err
	}

	return api, nil
}
