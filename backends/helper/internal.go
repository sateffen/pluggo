package helper

import "net"

type udpDialer interface {
	DialUDP(network string, laddr, raddr *net.UDPAddr) (*net.UDPConn, error)
}

type defaultUDPDialer struct{}

func (defaultUDPDialer) DialUDP(network string, laddr, raddr *net.UDPAddr) (*net.UDPConn, error) {
	return net.DialUDP(network, laddr, raddr)
}
