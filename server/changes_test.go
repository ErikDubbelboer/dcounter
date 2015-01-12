package server

import (
	"strconv"
	"testing"
)

func TestChanges(t *testing.T) {
	c := make(Changes, 0)

	s := 10

	for i := 0; i < s; i++ {
		if l := len(c); l != i {
			t.Errorf("expected len to be %d, got %d", i, l)
		}

		c.Add("test" + strconv.FormatInt(int64(i), 10))
	}

	for i := 0; i < s; i++ {
		if l := len(c); l != s-i {
			t.Errorf("expected len to be %d, got %d", s-i, l)
		}

		x := "test" + strconv.FormatInt(int64(i), 10)

		if n := c.Peek(); n != x {
			t.Errorf("expected Peek to be %s, got %s", x, n)
		}

		c.Pop()
	}

	if l := len(c); l != 0 {
		t.Errorf("expected len to be %d, got %d", 0, l)
	}

	s = 40

	for i := 0; i < s; i++ {
		if l := len(c); l != i {
			t.Errorf("expected len to be %d, got %d", i, l)
		}

		c.Add("test" + strconv.FormatInt(int64(i), 10))
	}

	for i := 0; i < s; i++ {
		if l := len(c); l != s-i {
			t.Errorf("expected len to be %d, got %d", s-i, l)
		}

		x := "test" + strconv.FormatInt(int64(i), 10)

		if n := c.Peek(); n != x {
			t.Errorf("expected Peek to be %s, got %s", x, n)
		}

		c.Pop()
	}

	if l := len(c); l != 0 {
		t.Errorf("expected len to be %d, got %d", 0, l)
	}
}
