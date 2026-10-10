package network

import (
	"net"
	"strconv"
)

func IsValidHostPort(addr string) bool {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil || host == "" {
		return false
	}

	port, err := strconv.Atoi(portStr)
	return err == nil && port > 0 && port <= 65535
}
