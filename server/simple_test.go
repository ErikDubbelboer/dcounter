package server

import (
	"testing"
)

func TestSimple(t *testing.T) {
	//t.SkipNow()

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

func TestReset(t *testing.T) {
	//t.SkipNow()

	s := NewTestServer(t, "s")

	s.Reset("test")
}

func BenchmarkGet(b *testing.B) {
	//b.SkipNow()

	s := NewTestServer(b, "s")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s.Get("test", 0, true)
	}
}

func BenchmarkInc(b *testing.B) {
	//b.SkipNow()

	s := NewTestServer(b, "s")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s.Inc("test", 1.2)
	}
}

func BenchmarkReset(b *testing.B) {
	//b.SkipNow()

	s := NewTestServer(b, "s")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s.Reset("test")
	}
}
