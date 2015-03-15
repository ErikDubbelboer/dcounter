package change

import (
	"github.com/atomx/memberlist"
)

type Change struct {
	name string
	data []byte
}

func New(name string, data []byte) Change {
	return Change{
		name: name,
		data: data,
	}
}

func (c Change) Invalidates(b memberlist.Broadcast) bool {
	o := b.(Change)

	return c.name == o.name
}

func (c Change) Message() []byte {
	return c.data
}

func (c Change) Finished() {
}
