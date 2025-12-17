package frontends

import (
	"errors"
	"net"
)

// mockTCPListener implements the tcpListener interface from internal.go.
type mockTCPListener struct {
	mockAccept func() (net.Conn, error)
	mockClose  func() error
	mockAddr   func() net.Addr
}

func (m *mockTCPListener) Accept() (net.Conn, error) {
	if m.mockAccept != nil {
		return m.mockAccept()
	}
	return nil, errors.New("mock listener: no mock implementation for Accept")
}

func (m *mockTCPListener) Close() error {
	if m.mockClose != nil {
		return m.mockClose()
	}
	return nil
}

func (m *mockTCPListener) Addr() net.Addr {
	if m.mockAddr != nil {
		return m.mockAddr()
	}
	addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	return addr
}

// mockTCPListenerFactory implements the tcpListenerFactory interface from internal.go.
type mockTCPListenerFactory struct {
	mockListenTCP func(network string, laddr *net.TCPAddr) (tcpListener, error)
}

func (m *mockTCPListenerFactory) ListenTCP(network string, laddr *net.TCPAddr) (tcpListener, error) {
	if m.mockListenTCP != nil {
		return m.mockListenTCP(network, laddr)
	}
	return nil, errors.New("mock listener factory: no mock implementation for ListenTCP")
}

// mockBackend implements the backends.Backend interface for testing.
type mockBackend struct {
	name       string
	mockHandle func(conn net.Conn)
}

func (m *mockBackend) GetName() string {
	return m.name
}

func (m *mockBackend) Handle(conn net.Conn) {
	if m.mockHandle != nil {
		m.mockHandle(conn)
		return
	}
	// Default: just close the connection
	conn.Close()
}

func (m *mockBackend) Close() error {
	return nil
}
