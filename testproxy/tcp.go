package testproxy

import (
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

type TCP struct {
	t *testing.T

	stop    bool
	timeout time.Time
	in      *net.TCPListener
}

func (p *TCP) handle(conn net.Conn, to string) {
	out, err := net.Dial("tcp", to)
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()

		for {
			n, err := io.CopyN(conn, out, 1024*1024)
			if err != nil && err != io.EOF {
				return
			} else if n != 0 {
				p.t.Logf("TCP %d bytes %v -> %s", n, conn.LocalAddr(), to)
			}
		}
	}()
	go func() {
		defer wg.Done()

		for {
			n, err := io.CopyN(out, conn, 1024*1024)
			if err != nil && err != io.EOF {
				return
			} else if n != 0 {
				p.t.Logf("TCP %d bytes %s -> %v", n, to, conn.LocalAddr())
			}
		}
	}()

	wg.Wait()

	out.Close()
}

func NewTCP(t *testing.T, from, to string) *TCP {
	p := TCP{
		t: t,
	}

	a, err := net.ResolveTCPAddr("tcp", from)
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			if p.stop {
				return
			}

			if time.Now().Before(p.timeout) {
				time.Sleep(p.timeout.Sub(time.Now()))
			}

			var err error
			p.in, err = net.ListenTCP("tcp", a)
			if err != nil {
				panic(err)
			}

			conns := make([]net.Conn, 0)

			for {
				conn, err := p.in.Accept()
				if err != nil {
					if strings.Contains(err.Error(), "use of closed network connection") {
						break
					} else {
						panic(err)
					}
				}

				conn.(*net.TCPConn).SetLinger(0)

				p.t.Logf("TCP new %s -> %s", from, to)

				conns = append(conns, conn)

				go p.handle(conn, to)
			}

			for _, conn := range conns {
				conn.Close()
			}
		}
	}()

	return &p
}

func (p *TCP) Timeout(timeout time.Duration) {
	p.timeout = time.Now().Add(timeout)

	p.in.Close()
}

func (p *TCP) Stop() {
	p.stop = true

	p.in.Close()
}
