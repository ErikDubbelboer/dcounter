// +build !race
// testproxy contains races when calling New and Timeout or Stop too quickly.

package server

import (
	"testing"
	"time"

	"github.com/atomx/dcounter/testproxy"
)

// This test is a bit flaky on TravisCI.
func TestNetworkFail(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	tcpa := testproxy.NewTCP(t, "localhost:3001", "localhost:2001")
	udpa := testproxy.NewUDP(t, "localhost:3001", "localhost:2001")
	tcpb := testproxy.NewTCP(t, "localhost:3002", "localhost:2002")
	udpb := testproxy.NewUDP(t, "localhost:3002", "localhost:2002")

	a := NewTestServerOn(t, "a", "localhost:2001", "localhost:3001")
	time.Sleep(time.Second)
	b := NewTestServerOn(t, "b", "localhost:2002", "localhost:3002")
	time.Sleep(time.Second)

	a.JoinOn("localhost:3002")
	time.Sleep(time.Second)

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
	time.Sleep(d + time.Second)
	t.Log("done, waiting for reconnect")
	time.Sleep(time.Second * 16)
	t.Log("should be reconnected now")

	a.Inc("test", 1)
	b.Inc("test", 2)
	time.Sleep(time.Second * 2)
	a.Get("test", 6, true)
	b.Get("test", 6, true)
}
