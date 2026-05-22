package proxy

import (
	"net"
)

type netTCPAddr = net.TCPAddr

func ioListener(addr string) (net.Listener, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return ln, nil
}
