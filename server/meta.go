package server

import (
	"bytes"
	"encoding/binary"
)

// TODO: implement GobEncoder and GobDecoder to reduce size?
type Meta struct {
	Id      ID
	Leaving byte
}

func (m *Meta) Encode(limit int) []byte {
	// 1024 bytes should be enough to encode the meta.
	buffer := make(Buffer, 0, min(1024, limit))

	if err := binary.Write(&buffer, binary.LittleEndian, m); err != nil {
		panic(err)
	}

	return buffer
}

func (m *Meta) Decode(b []byte) error {
	reader := bytes.NewReader(b)

	if err := binary.Read(reader, binary.LittleEndian, m); err != nil {
		return err
	}

	return nil
}
