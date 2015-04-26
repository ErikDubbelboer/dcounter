package server

import (
	"testing"
)

func TestSimple(t *testing.T) {
	s := NewTestServer(t, "s")

	s.Get("test", 0, true)
	s.Inc("test", 1)
	s.Inc("test", 1)
	s.Get("test", 2, true)
	s.Set("test", 0)
	s.Get("test", 0, true)
	s.Inc("test", 1)
	s.Inc("test", -1)
	s.Inc("test", -1)
	s.Get("test", -1, true)
}

func TestSet(t *testing.T) {
	s := NewTestServer(t, "s")

	s.Set("test", 0)
}

func BenchmarkGet(b *testing.B) {
	s := NewTestServer(emptyBorT(0), "s")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s.Get("test", 0, true)
	}
}

func BenchmarkInc(b *testing.B) {
	s := NewTestServer(emptyBorT(0), "s")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s.Inc("test", 1.2)
	}
}

func BenchmarkSet(b *testing.B) {
	s := NewTestServer(emptyBorT(0), "s")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s.Set("test", 0)
	}
}
