package server

import (
	"testing"
)

func TestBuffer(t *testing.T) {
	buffer := make(buffer, 0, 2)

	if n, err := buffer.Write([]byte{1}); err != nil {
		t.Error(err)
	} else if n != 1 {
		t.Errorf("only wrote %d, expected 1", n)
	} else if len(buffer) != 1 {
		t.Errorf("len = %d, expected 1", len(buffer))
	}

	if _, err := buffer.Write([]byte{1, 2, 3}); err == nil {
		t.Error("expected error")
	}
}
