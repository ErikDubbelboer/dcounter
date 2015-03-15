package server

import (
	"sync"
	"testing"
	"time"
)

func TestCluster(t *testing.T) {
	t.Parallel()

	a := NewTestServer(t, "a")
	b := NewTestServer(t, "b")

	b.Join(a)
	time.Sleep(time.Second)
	//a.Debug()
	//b.Debug()

	a.Inc("test", 1)
	b.Inc("test", 2)
	time.Sleep(time.Second)
	a.Get("test", 3, true)
	b.Get("test", 3, true)
	b.Set("test", 1)
	time.Sleep(time.Second)
	a.Get("test", 1, true)
	b.Get("test", 1, true)

	// the time.Sleep below should be enough prevent a race condition but the
	// race detector doesn't have enough knowledge for this so use a waitgroup
	// to prevent it from detecting a race condition here.
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		a.Inc("test", -2)
		wg.Done()
	}()
	go func() {
		b.Inc("test", 4)
		wg.Done()
	}()
	wg.Wait()

	time.Sleep(time.Second)
	a.Get("test", 3, true)
	b.Get("test", 3, true)

	a.Stop()

	b.Get("test", 3, true)

	c := NewTestServer(t, "c")
	defer c.Kill()

	c.Join(b)
	time.Sleep(time.Second)
	//b.Debug()
	//c.Debug()

	b.Get("test", 3, true)
	c.Get("test", 3, true)

	c.Inc("test", 5)
	time.Sleep(time.Second)

	b.Get("test", 8, true)
	c.Get("test", 8, true)

	b.Stop()

	c.Get("test", 8, true)
}

func TestNoJoin(t *testing.T) {
	t.Parallel()

	a := NewTestServer(t, "a")

	a.a.Join([]string{"127.0.0.1:19727"})
	time.Sleep(time.Second)

	a.Get("test", 0, false)
}

func BenchmarkCluserInc(b *testing.B) {
	a := NewTestServer(emptyBorT(0), "a")
	c := NewTestServer(emptyBorT(0), "c")

	b.ResetTimer()

	a.Join(c)

	for i := 0; i < b.N; i++ {
		a.Inc("test", 1.2)
		c.Inc("test", -1.2)
	}
}

func BenchmarkCluserSet(b *testing.B) {
	a := NewTestServer(emptyBorT(0), "a")
	c := NewTestServer(emptyBorT(0), "c")

	b.ResetTimer()

	a.Join(c)

	for i := 0; i < b.N; i++ {
		a.Set("test", 0)
		c.Inc("test", 1)
	}
}
