package server

import (
	"testing"
	"time"
)

func TestCluster(t *testing.T) {
	a := NewTestServer(t, "a")
	b := NewTestServer(t, "b")

	b.Join(a)
	time.Sleep(time.Second)
	a.Debug()
	b.Debug()

	a.Inc("test", 1)
	b.Inc("test", 1)
	time.Sleep(time.Second)
	a.Get("test", 2, true)
	b.Get("test", 2, true)
	b.Reset("test")
	time.Sleep(time.Second)
	a.Get("test", 0, true)
	b.Get("test", 0, true)
	go a.Inc("test", -1)
	go b.Inc("test", 4)
	time.Sleep(time.Second)
	a.Get("test", 3, true)
	b.Get("test", 3, true)

	a.Stop()

	b.Get("test", 3, true)

	c := NewTestServer(t, "c")
	defer c.Kill()

	c.Join(b)
	time.Sleep(time.Second)
	b.Debug()
	c.Debug()

	b.Get("test", 3, true)
	c.Get("test", 3, true)

	c.Inc("test", 5)
	time.Sleep(time.Second)

	b.Get("test", 8, true)
	c.Get("test", 8, true)

	b.Stop()

	c.Get("test", 8, true)
}
