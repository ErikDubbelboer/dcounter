package server

import (
	"bytes"
	"encoding/binary"
)

type meta struct {
	Leaving byte
}

func (m *meta) Encode(limit int) []byte {
	// 1024 bytes should be enough to encode the meta.
	buffer := make(buffer, 0, min(1024, limit))

	if err := binary.Write(&buffer, binary.LittleEndian, m); err != nil {
		panic(err)
	}

	return buffer
}

func (m *meta) Decode(b []byte) error {
	reader := bytes.NewReader(b)

	if err := binary.Read(reader, binary.LittleEndian, m); err != nil {
		return err
	}

	return nil
}
