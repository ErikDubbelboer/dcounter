package server

import (
	"net"
	"strconv"
	"strings"
)

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

func splitHostPort(hostport string, defaultPort int) (string, int, error) {
	if !strings.ContainsRune(hostport, ':') {
		hostport += ":" + strconv.FormatInt(int64(defaultPort), 10)
	}

	host, portStr, err := net.SplitHostPort(hostport)
	if err != nil {
		return "", 0, err
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return "", 0, err
	}

	port, err := strconv.ParseInt(portStr, 10, 32)
	if err != nil {
		panic(err)
	}

	return ips[0].String(), int(port), nil
}
