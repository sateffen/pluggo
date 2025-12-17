package backends

import (
	"errors"
	"net"
	"time"
)

// mockDialer implements the dialer interface from internal.go.
type mockDialer struct {
	mockDialTimeout func(network, address string, timeout time.Duration) (net.Conn, error)
}

func (m *mockDialer) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	if m.mockDialTimeout != nil {
		return m.mockDialTimeout(network, address, timeout)
	}
	return nil, errors.New("mock dialer: no mock implementation found")
}

// mockSleeper implements the sleeper interface from internal.go.
type mockSleeper struct {
	mockSleep  func(d time.Duration)
	sleepCalls []time.Duration
	totalSlept time.Duration
	trackCalls bool
}

func (m *mockSleeper) Sleep(d time.Duration) {
	if m.trackCalls {
		m.sleepCalls = append(m.sleepCalls, d)
		m.totalSlept += d
	}
	if m.mockSleep != nil {
		m.mockSleep(d)
	}
}

// mockWoLSender implements the utils.WoLSender interface.
type mockWoLSender struct {
	mockSendWoLPacket func() error
	sendCount         int
}

func (m *mockWoLSender) SendWoLPacket() error {
	m.sendCount++
	if m.mockSendWoLPacket != nil {
		return m.mockSendWoLPacket()
	}
	return nil
}
