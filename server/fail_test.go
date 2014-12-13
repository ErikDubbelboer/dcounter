package server

import (
	"testing"
	"time"

	"github.com/atomx/dcounter/testproxy"
)

func TestNetwork(t *testing.T) {
	t.SkipNow()

	tcpa := testproxy.NewTCP(t, "127.0.0.1:3001", "127.0.0.1:2001")
	udpa := testproxy.NewUDP(t, "127.0.0.1:3001", "127.0.0.1:2001")
	tcpb := testproxy.NewTCP(t, "127.0.0.1:3002", "127.0.0.1:2002")
	udpb := testproxy.NewUDP(t, "127.0.0.1:3002", "127.0.0.1:2002")

	a := NewTestServerOn(t, "a", "127.0.0.1:2001", "127.0.0.1:3001")
	b := NewTestServerOn(t, "b", "127.0.0.1:2002", "127.0.0.1:3002")

	a.JoinOn("127.0.0.1:3002")
	time.Sleep(time.Second)
	a.Debug()
	b.Debug()

	a.Inc("test", 1)
	b.Inc("test", 2)
	time.Sleep(time.Second)
	a.Get("test", 3, true)
	b.Get("test", 3, true)

	t.Log("timeout")
	d := time.Second * 10
	tcpa.Timeout(d)
	udpa.Timeout(d)
	tcpb.Timeout(d)
	udpb.Timeout(d)
	time.Sleep(d)
	t.Log("done, waiting for reconnect")
	time.Sleep(time.Second * 5)
	t.Log("should be reconnected now")
	a.Debug()
	b.Debug()

	a.Inc("test", 1)
	b.Inc("test", 2)
	time.Sleep(time.Second * 1)
	a.Get("test", 6, true)
	b.Get("test", 6, true)

	t.Fail()
}
