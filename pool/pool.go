package dcounter

import (
	"fmt"
	"time"

	api "github.com/atomx/dcounter/api"
)

type Pool chan *api.API

// Get gets a connection from the pool.
// A zero or negative timeout causes Get to wait as long as it takes to get a connection.
func (p Pool) Get(timeout time.Duration) (*api.API, error) {
	if timeout > 0 {
		select {
		case <-time.After(timeout):
			return nil, fmt.Errorf("timeout while waiting for dcounter pool")
		case a := <-p:
			return a, nil
		}
	} else {
		return <-p, nil
	}
}

func (p Pool) Put(a *api.API) {
	p <- a
}

func New(network, addr string, size int) Pool {
	p := make(chan *api.API, size)

	for i := 0; i < size; i++ {
		p <- api.New(network, addr)
	}

	return p
}
