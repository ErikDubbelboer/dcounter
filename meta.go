package main

import (
	"bytes"
	"encoding/gob"
)

type Meta struct {
	id      int
	leaving bool
}

func init() {
	gob.Register(&Meta{})
}

func (m *Meta) Encode(limit int) []byte {
	// 32 bytes should be enough to encode the meta.
	buffer := make(Buffer, 0, min(32, limit))
	encoder := gob.NewEncoder(&buffer)

	if err := encoder.Encode(m); err != nil {
		panic(err)
	}

	return buffer
}

func (m *Meta) Decode(b []byte) error {
	reader := bytes.NewReader(b)
	decoder := gob.NewDecoder(reader)

	if err := decoder.Decode(m); err != nil {
		return err
	}
	return nil
}
