package server

import (
	"net"
	//"net/http"
	//_ "net/http/pprof"
	"strconv"
)

func init() {
	//go http.ListenAndServe(":12345", nil)
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
