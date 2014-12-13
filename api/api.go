package api

import (
	"fmt"
	"net"
	"strconv"

	"github.com/atomx/dcounter/proto"
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

func (api *API) Get(id string) (float64, bool, error) {
	var err error

	for i := 0; i < 2; i++ {
		if api.c == nil {
			if err := api.connect(); err != nil {
				return 0, false, err
			}
		}

		if err = api.p.Write("GET", []string{id}); err != nil {
			continue
		} else if cmd, args, err := api.p.Read(); err != nil {
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

func (api *API) Reset(id string) error {
	var err error

	for i := 0; i < 2; i++ {
		if api.c == nil {
			if err := api.connect(); err != nil {
				return err
			}
		}

		if err = api.p.Write("RESET", []string{id}); err != nil {
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

func (api *API) List() (map[string]float64, error) {
	var err error

	for i := 0; i < 2; i++ {
		if api.c == nil {
			if err := api.connect(); err != nil {
				return nil, err
			}
		}

		if err = api.p.Write("LIST", []string{}); err != nil {
			continue
		} else if cmd, args, err := api.p.Read(); err != nil {
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

func (api *API) Join(hosts []string) error {
	var err error

	for i := 0; i < 2; i++ {
		if api.c == nil {
			if err := api.connect(); err != nil {
				return err
			}
		}

		if err = api.p.Write("JOIN", hosts); err != nil {
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
