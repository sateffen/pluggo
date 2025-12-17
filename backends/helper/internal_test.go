package helper

import (
	"errors"
	"net"
	"time"
)

// mockConn implements the net.Conn interface for testing.
type mockConn struct {
	closeCallCount uint
}

func (m *mockConn) Read(_ []byte) (int, error)         { return 0, errors.New("not implemented") }
func (m *mockConn) Write(_ []byte) (int, error)        { return 0, errors.New("not implemented") }
func (m *mockConn) Close() error                       { m.closeCallCount++; return nil }
func (m *mockConn) LocalAddr() net.Addr                { return nil }
func (m *mockConn) RemoteAddr() net.Addr               { return nil }
func (m *mockConn) SetDeadline(_ time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(_ time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(_ time.Time) error { return nil }

// mockUDPDialer implements the udpDialer interface.
type mockUDPDialer struct {
	mockDialUDP func(network string, laddr, raddr *net.UDPAddr) (*net.UDPConn, error)
}

func (m *mockUDPDialer) DialUDP(network string, laddr, raddr *net.UDPAddr) (*net.UDPConn, error) {
	if m.mockDialUDP != nil {
		return m.mockDialUDP(network, laddr, raddr)
	}

	return nil, errors.New("mock dialer: no mock implementation found")
}
