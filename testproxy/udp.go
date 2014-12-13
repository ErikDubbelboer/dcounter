package testproxy

import (
	"net"
	"testing"
	"time"
)

type UDP struct {
	t *testing.T

	stop    bool
	timeout time.Time
}

func NewUDP(t *testing.T, from, to string) *UDP {
	p := UDP{
		t: t,
	}

	a, err := net.ResolveUDPAddr("udp", from)
	if err != nil {
		panic(err)
	}

	in, err := net.ListenUDP("udp", a)
	if err != nil {
		panic(err)
	}

	out, err := net.ResolveUDPAddr("udp", to)
	if err != nil {
		panic(err)
	}

	go func() {
		defer in.Close()

		var buffer [65535]byte

		for {
			if p.stop {
				return
			}

			in.SetDeadline(time.Now().Add(time.Millisecond * 500))

			read, _, err := in.ReadFromUDP(buffer[0:])
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				continue
			}
			if err != nil {
				panic(err)
			}

			if time.Now().Before(p.timeout) {
				p.t.Logf("UDP drop %d %v -> %s", read, from, to)
				continue
			} else {
				p.t.Logf("UDP send %d %v -> %s", read, from, to)
			}

			in.SetDeadline(time.Time{})

			if _, err := in.WriteToUDP(buffer[:read], out); err != nil {
				panic(err)
			}
		}
	}()

	return &p
}

func (p *UDP) Timeout(timeout time.Duration) {
	p.timeout = time.Now().Add(timeout)
}

func (p *UDP) Stop(timeout time.Duration) {
	p.stop = true
}
