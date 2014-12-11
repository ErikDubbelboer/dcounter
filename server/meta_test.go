package server

import (
	"testing"
)

func TestMeta(t *testing.T) {
	m := Meta{
		Id:      ID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		Leaving: 0,
	}

	buffer := m.Encode(1024)

	var r Meta
	if err := r.Decode(buffer); err != nil {
		t.Error(err)
	}

	if r.Id != m.Id {
		t.Errorf("%v does not match %v", r.Id, m.Id)
	} else if r.Leaving != m.Leaving {
		t.Errorf("%v does not match %v", r.Leaving, m.Leaving)
	}
}

func TestMetaDecode(t *testing.T) {
	buffer := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 1}
	id := ID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}

	var m Meta
	if err := m.Decode(buffer); err != nil {
		t.Error(err)
	}

	if m.Id != id {
		t.Errorf("%v does not match %v", m.Id, id)
	} else if m.Leaving != 1 {
		t.Errorf("%v does not match 1", m.Leaving)
	}
}
