package server

import (
	"testing"
	"time"
)

func TestReplicate(t *testing.T) {
	t.Parallel()

	a := NewTestServer(t, "a")
	b := NewTestServer(t, "b")

	b.Join(a)
	time.Sleep(time.Second)

	a.Inc("test", 1)
	time.Sleep(time.Second)
	a.Get("test", 1, true)
	b.Get("test", 1, true)

	aData := a.Save()
	if _, ok := aData["a"]["test"]; !ok {
		t.Error("a.a does not have test")
	}
	if _, ok := aData["b"]["test"]; ok {
		t.Error("a.b does have test")
	}

	bData := b.Save()
	if _, ok := bData["a"]["test"]; !ok {
		t.Error("b.a does not have test")
	}
	if _, ok := bData["b"]["test"]; ok {
		t.Error("b.b does have test")
	}
}
