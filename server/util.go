package server

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"strconv"
	"time"
)

func init() {
	go http.ListenAndServe(":12345", nil)
}

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

func splitHostPort(hostport string) (string, int) {
	addr, portStr, err := net.SplitHostPort(hostport)
	if err != nil {
		panic(err)
	}

	port, err := strconv.ParseInt(portStr, 10, 32)
	if err != nil {
		panic(err)
	}

	return addr, int(port)
}

// RandomName returns a random node name.
// This is the hex representation of a UUID starting with the current time.
func RandomName() string {
	var id [16]byte

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

	return fmt.Sprintf("%x-%x-%x-%x-%x", id[0:4], id[4:6], id[6:8], id[8:10], id[10:])
}
