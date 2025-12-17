package backends

import (
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/sateffen/pluggo/config"
)

func TestWoLForwarderBackend_GetName(t *testing.T) {
	backend, err := newWoLForwarderBackend(config.WoLForwarderBackendConfig{
		Name:             "test-wol-forwarder",
		TargetAddr:       "127.0.0.1:80",
		WoLMACAddr:       "00:11:22:33:44:55",
		WoLBroadcastAddr: "255.255.255.255:9",
	})
	if err != nil {
		t.Fatalf("newWoLForwarderBackend() failed: %v", err)
	}

	if got := backend.GetName(); got != "test-wol-forwarder" {
		t.Errorf("GetName() = %q, want %q", got, "test-wol-forwarder")
	}
}

func TestWoLForwarderBackend_TryDial_TargetAlreadyAwake(t *testing.T) {
	targetBackendEnd, targetClientEnd := net.Pipe()
	defer targetBackendEnd.Close()
	defer targetClientEnd.Close()

	dialAttempts := 0
	mockDialer := &mockDialer{
		mockDialTimeout: func(_, _ string, _ time.Duration) (net.Conn, error) {
			dialAttempts++
			// First dial succeeds immediately
			return targetBackendEnd, nil
		},
	}

	mockSleeper := &mockSleeper{trackCalls: true}
	mockWoL := &mockWoLSender{}

	backend, err := newWoLForwarderBackend(config.WoLForwarderBackendConfig{
		Name:             "test-wol",
		TargetAddr:       "127.0.0.2:80",
		WoLMACAddr:       "00:11:22:33:44:55",
		WoLBroadcastAddr: "255.255.255.255:9",
	})
	if err != nil {
		t.Fatalf("newWoLForwarderBackend() failed: %v", err)
	}
	backend.dialer = mockDialer
	backend.sleeper = mockSleeper
	backend.wolSender = mockWoL

	// Call tryDial
	conn, err := backend.tryDial()

	// Should succeed immediately
	if err != nil {
		t.Fatalf("tryDial() failed: %v", err)
	}
	if conn != targetBackendEnd {
		t.Error("tryDial() returned wrong connection")
	}

	// Should have called dial once
	if dialAttempts != 1 {
		t.Error("expected dial to be called")
	}

	// Should NOT have sent WoL packet
	if mockWoL.sendCount != 0 {
		t.Errorf("WoL send count = %d, want 0", mockWoL.sendCount)
	}

	// Should NOT have slept
	if len(mockSleeper.sleepCalls) != 0 {
		t.Errorf("sleep called %d times, want 0", len(mockSleeper.sleepCalls))
	}
}

func TestWoLForwarderBackend_TryDial_WakesTargetAndSucceeds(t *testing.T) {
	targetBackendEnd, targetClientEnd := net.Pipe()
	defer targetBackendEnd.Close()
	defer targetClientEnd.Close()

	dialAttempts := 0
	mockDialer := &mockDialer{
		mockDialTimeout: func(_, _ string, _ time.Duration) (net.Conn, error) {
			dialAttempts++
			// First dial fails, second dial (after WoL + sleep) succeeds
			if dialAttempts == 1 {
				return nil, errors.New("connection refused")
			}
			return targetBackendEnd, nil
		},
	}

	mockSleeper := &mockSleeper{trackCalls: true}
	mockWoL := &mockWoLSender{}

	backend, err := newWoLForwarderBackend(config.WoLForwarderBackendConfig{
		Name:             "test-wol",
		TargetAddr:       "127.0.0.3:80",
		WoLMACAddr:       "00:11:22:33:44:55",
		WoLBroadcastAddr: "255.255.255.255:9",
	})
	if err != nil {
		t.Fatalf("newWoLForwarderBackend() failed: %v", err)
	}
	backend.dialer = mockDialer
	backend.sleeper = mockSleeper
	backend.wolSender = mockWoL

	conn, err := backend.tryDial()

	// Should succeed
	if err != nil {
		t.Fatalf("tryDial() failed: %v", err)
	}
	if conn != targetBackendEnd {
		t.Error("tryDial() returned wrong connection")
	}

	// Should have tried dial twice (initial + one retry)
	if dialAttempts != 2 {
		t.Errorf("dial attempts = %d, want 2", dialAttempts)
	}

	// Should have sent WoL packet once
	if mockWoL.sendCount != 1 {
		t.Errorf("WoL send count = %d, want 1", mockWoL.sendCount)
	}

	// Should have slept twice: 5s initial + 500ms retry
	if len(mockSleeper.sleepCalls) != 2 {
		t.Fatalf("sleep called %d times, want 2", len(mockSleeper.sleepCalls))
	}
	if mockSleeper.sleepCalls[0] != 5*time.Second {
		t.Errorf("first sleep = %v, want %v", mockSleeper.sleepCalls[0], 5*time.Second)
	}
	if mockSleeper.sleepCalls[1] != 500*time.Millisecond {
		t.Errorf("second sleep = %v, want %v", mockSleeper.sleepCalls[1], 500*time.Millisecond)
	}
}

func TestWoLForwarderBackend_TryDial_WoLSendFails(t *testing.T) {
	mockDialer := &mockDialer{
		mockDialTimeout: func(_, _ string, _ time.Duration) (net.Conn, error) {
			// Initial dial fails
			return nil, errors.New("connection refused")
		},
	}

	mockSleeper := &mockSleeper{trackCalls: true}
	mockWoL := &mockWoLSender{
		mockSendWoLPacket: func() error {
			return errors.New("failed to send WoL packet")
		},
	}

	backend, err := newWoLForwarderBackend(config.WoLForwarderBackendConfig{
		Name:             "test-wol",
		TargetAddr:       "127.0.0.4:80",
		WoLMACAddr:       "00:11:22:33:44:55",
		WoLBroadcastAddr: "255.255.255.255:9",
	})
	if err != nil {
		t.Fatalf("newWoLForwarderBackend() failed: %v", err)
	}
	backend.dialer = mockDialer
	backend.sleeper = mockSleeper
	backend.wolSender = mockWoL

	conn, err := backend.tryDial()

	// Should fail
	if err == nil {
		t.Fatal("expected tryDial() to fail when WoL send fails")
	}
	if conn != nil {
		t.Error("expected nil connection when WoL send fails")
	}

	// Should have sent WoL packet once
	if mockWoL.sendCount != 1 {
		t.Errorf("WoL send count = %d, want 1", mockWoL.sendCount)
	}

	// Should NOT have slept (fails before retry loop)
	if len(mockSleeper.sleepCalls) != 0 {
		t.Errorf("sleep called %d times, want 0", len(mockSleeper.sleepCalls))
	}
}

func TestWoLForwarderBackend_TryDial_TimeoutAfterRetries(t *testing.T) {
	dialAttempts := 0
	mockDialer := &mockDialer{
		mockDialTimeout: func(_, _ string, _ time.Duration) (net.Conn, error) {
			dialAttempts++
			// Always fail
			return nil, errors.New("connection refused")
		},
	}

	mockSleeper := &mockSleeper{trackCalls: true}
	mockWoL := &mockWoLSender{}

	backend, err := newWoLForwarderBackend(config.WoLForwarderBackendConfig{
		Name:             "test-wol",
		TargetAddr:       "127.0.0.5:80",
		WoLMACAddr:       "00:11:22:33:44:55",
		WoLBroadcastAddr: "255.255.255.255:9",
	})
	if err != nil {
		t.Fatalf("newWoLForwarderBackend() failed: %v", err)
	}
	backend.dialer = mockDialer
	backend.sleeper = mockSleeper
	backend.wolSender = mockWoL

	conn, err := backend.tryDial()

	// Should fail with timeout
	if err == nil {
		t.Fatal("expected tryDial() to fail after retries")
	}
	if conn != nil {
		t.Error("expected nil connection after timeout")
	}

	// Should have tried initial dial + 50 retries = 51 total
	if dialAttempts != 51 {
		t.Errorf("dial attempts = %d, want 51 (1 initial + 50 retries)", dialAttempts)
	}

	// Should have sent WoL packet once
	if mockWoL.sendCount != 1 {
		t.Errorf("WoL send count = %d, want 1", mockWoL.sendCount)
	}

	// Should have slept 51 times: 1 initial (5s) + 50 retries (500ms each)
	if len(mockSleeper.sleepCalls) != 51 {
		t.Fatalf("sleep called %d times, want 51", len(mockSleeper.sleepCalls))
	}

	// Verify first sleep is 5s
	if mockSleeper.sleepCalls[0] != 5*time.Second {
		t.Errorf("first sleep = %v, want %v", mockSleeper.sleepCalls[0], 5*time.Second)
	}

	// Verify remaining sleeps are 500ms each
	for i := 1; i < len(mockSleeper.sleepCalls); i++ {
		if mockSleeper.sleepCalls[i] != 500*time.Millisecond {
			t.Errorf("sleep call %d = %v, want %v", i, mockSleeper.sleepCalls[i], 500*time.Millisecond)
			break
		}
	}
}

func TestWoLForwarderBackend_Handle_SuccessfulConnection(t *testing.T) {
	targetBackendEnd, targetClientEnd := net.Pipe()
	defer targetBackendEnd.Close()
	defer targetClientEnd.Close()

	mockDialer := &mockDialer{
		mockDialTimeout: func(_, _ string, _ time.Duration) (net.Conn, error) {
			return targetBackendEnd, nil
		},
	}

	mockSleeper := &mockSleeper{}
	mockWoL := &mockWoLSender{}

	backend, err := newWoLForwarderBackend(config.WoLForwarderBackendConfig{
		Name:             "test-wol",
		TargetAddr:       "127.0.0.6:80",
		WoLMACAddr:       "00:11:22:33:44:55",
		WoLBroadcastAddr: "255.255.255.255:9",
	})
	if err != nil {
		t.Fatalf("newWoLForwarderBackend() failed: %v", err)
	}
	backend.dialer = mockDialer
	backend.sleeper = mockSleeper
	backend.wolSender = mockWoL

	incomingBackendConn, incomingTestConn := net.Pipe()
	defer incomingBackendConn.Close()
	defer incomingTestConn.Close()

	backend.Handle(incomingBackendConn)

	// Verify bidirectional data flow
	testData := []byte("test message")
	go func() {
		incomingTestConn.Write(testData)
	}()

	readBuffer := make([]byte, len(testData))
	n, err := io.ReadFull(targetClientEnd, readBuffer)
	if err != nil {
		t.Fatalf("failed to read from target: %v", err)
	}

	if string(readBuffer[:n]) != string(testData) {
		t.Errorf("target received %q, want %q", string(readBuffer[:n]), string(testData))
	}
}

func TestWoLForwarderBackend_Handle_DialFailure(t *testing.T) {
	mockDialer := &mockDialer{
		mockDialTimeout: func(_, _ string, _ time.Duration) (net.Conn, error) {
			return nil, errors.New("connection refused")
		},
	}

	mockSleeper := &mockSleeper{}
	mockWoL := &mockWoLSender{
		mockSendWoLPacket: func() error {
			return errors.New("WoL send failed")
		},
	}

	backend, err := newWoLForwarderBackend(config.WoLForwarderBackendConfig{
		Name:             "test-wol",
		TargetAddr:       "127.0.0.7:80",
		WoLMACAddr:       "00:11:22:33:44:55",
		WoLBroadcastAddr: "255.255.255.255:9",
	})
	if err != nil {
		t.Fatalf("newWoLForwarderBackend() failed: %v", err)
	}
	backend.dialer = mockDialer
	backend.sleeper = mockSleeper
	backend.wolSender = mockWoL

	incomingBackendConn, incomingTestConn := net.Pipe()
	defer incomingTestConn.Close()

	backend.Handle(incomingBackendConn)

	// Connection should be closed when dial fails, so we should read (0, EOF)
	// If the connection is not closed, this will hang till the test-timeout kills it
	readBuffer := make([]byte, 1)
	n, err := incomingTestConn.Read(readBuffer)

	if !errors.Is(err, io.EOF) {
		t.Errorf("expected io.EOF after dial failure, got: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 bytes read, got %d", n)
	}
}
