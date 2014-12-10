package main

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"time"
)

type ID [16]byte

func (id *ID) Randomize() {
	_, err := rand.Read(id[4:])
	if err != nil {
		panic(err)
	}
	// first 4 bytes are the timestamp the ID was generated.
	// We use big endian so we can print the whole number using %x
	// and take the first 8 characters for a timestamp.
	// Big endian also makes the first bytes stay the same which could
	// help if we want to do some compression in the future.
	binary.BigEndian.PutUint32(id[:4], uint32(time.Now().Unix()))
}

func (id ID) String() string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", id[0:4], id[4:6], id[6:8], id[8:10], id[10:])
}
