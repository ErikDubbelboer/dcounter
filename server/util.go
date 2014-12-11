package server

import (
	"net"
	"strconv"
)

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
