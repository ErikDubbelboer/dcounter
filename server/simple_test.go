package server

import (
	"testing"
)

func TestSimple(t *testing.T) {
	t.SkipNow()

	s := NewTestServer(t, "s")

	s.Get("test", 0, true)
	s.Inc("test", 1)
	s.Inc("test", 1)
	s.Get("test", 2, true)
	s.Reset("test")
	s.Get("test", 0, true)
	s.Inc("test", 1)
	s.Inc("test", -1)
	s.Inc("test", -1)
	s.Get("test", -1, true)
}
