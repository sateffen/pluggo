package frontends

import "net"

type tcpListener interface {
	Accept() (net.Conn, error)
	Close() error
	Addr() net.Addr
}

type tcpListenerFactory interface {
	ListenTCP(network string, laddr *net.TCPAddr) (tcpListener, error)
}

type defaultTCPListenerFactory struct{}

func (defaultTCPListenerFactory) ListenTCP(network string, laddr *net.TCPAddr) (tcpListener, error) {
	return net.ListenTCP(network, laddr)
}
