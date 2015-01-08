package server

import (
	"strconv"
	"testing"
)

func TestChanges(t *testing.T) {
	c := NewChanges()

	s := 10

	for i := 0; i < s; i++ {
		if l := c.Len(); l != i {
			t.Errorf("expected Len to be %d, got %d", i, l)
		}

		c.Add("test" + strconv.FormatInt(int64(i), 10))
	}

	for i := 0; i < s; i++ {
		if l := c.Len(); l != s-i {
			t.Errorf("expected Len to be %d, got %d", s-i, l)
		}

		x := "test" + strconv.FormatInt(int64(i), 10)

		if n := c.Peek(); n != x {
			t.Errorf("expected Peek to be %s, got %s", x, n)
		}

		c.Pop()
	}

	if l := c.Len(); l != 0 {
		t.Errorf("expected Len to be %d, got %d", 0, l)
	}

	s = 40

	for i := 0; i < s; i++ {
		if l := c.Len(); l != i {
			t.Errorf("expected Len to be %d, got %d", i, l)
		}

		c.Add("test" + strconv.FormatInt(int64(i), 10))
	}

	for i := 0; i < s; i++ {
		if l := c.Len(); l != s-i {
			t.Errorf("expected Len to be %d, got %d", s-i, l)
		}

		x := "test" + strconv.FormatInt(int64(i), 10)

		if n := c.Peek(); n != x {
			t.Errorf("expected Peek to be %s, got %s", x, n)
		}

		c.Pop()
	}

	if l := c.Len(); l != 0 {
		t.Errorf("expected Len to be %d, got %d", 0, l)
	}
}
