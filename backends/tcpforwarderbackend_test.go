package backends

import (
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/sateffen/pluggo/config"
)

func TestTCPForwarderBackend_GetName(t *testing.T) {
	backend := newTCPForwarderBackend(config.TCPForwarderBackendConfig{
		Name:       "test-tcp-forwarder",
		TargetAddr: "127.0.0.1:3000",
	})

	if got := backend.GetName(); got != "test-tcp-forwarder" {
		t.Errorf("GetName() = %q, want %q", got, "test-tcp-forwarder")
	}
}

func TestTCPForwarderBackend_Handle_SuccessfulDial(t *testing.T) {
	// Create a pipe to simulate the target connection
	targetClientEnd, targetBackendEnd := net.Pipe()
	defer targetClientEnd.Close()
	defer targetBackendEnd.Close()

	// Mock dialer that returns our simulated target connection
	mockDialer := &mockDialer{
		mockDialTimeout: func(network, address string, timeout time.Duration) (net.Conn, error) {
			// Verify correct parameters
			if network != "tcp" {
				t.Errorf("dial network = %q, want %q", network, "tcp")
			}
			if address != "127.0.0.2:3000" {
				t.Errorf("dial address = %q, want %q", address, "127.0.0.2:3000")
			}
			if timeout != 10*time.Second {
				t.Errorf("dial timeout = %v, want %v", timeout, 10*time.Second)
			}
			return targetBackendEnd, nil
		},
	}

	backend := newTCPForwarderBackend(config.TCPForwarderBackendConfig{
		Name:       "test-forwarder",
		TargetAddr: "127.0.0.2:3000",
	})
	backend.dialer = mockDialer

	// Create incoming connection
	incomingBackendConn, incomingTestConn := net.Pipe()
	defer incomingBackendConn.Close()
	defer incomingTestConn.Close()

	// Handle the connection
	backend.Handle(incomingBackendConn)

	// Verify data flows through: incoming -> target
	testData := []byte("hello target")
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

	// Verify data flows back: target -> incoming
	responseData := []byte("hello client")
	go func() {
		targetClientEnd.Write(responseData)
	}()

	readBuffer = make([]byte, len(responseData))
	n, err = io.ReadFull(incomingTestConn, readBuffer)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}

	if string(readBuffer[:n]) != string(responseData) {
		t.Errorf("client received %q, want %q", string(readBuffer[:n]), string(responseData))
	}
}

func TestTCPForwarderBackend_Handle_DialFailure(t *testing.T) {
	// Mock dialer that always fails
	mockDialer := &mockDialer{
		mockDialTimeout: func(_, _ string, _ time.Duration) (net.Conn, error) {
			return nil, errors.New("connection refused")
		},
	}

	backend := newTCPForwarderBackend(config.TCPForwarderBackendConfig{
		Name:       "test-forwarder",
		TargetAddr: "127.0.04:3000",
	})
	backend.dialer = mockDialer

	incomingBackendConn, incomingTestConn := net.Pipe()
	defer incomingTestConn.Close()

	// Handle should close the incoming connection when dial fails
	backend.Handle(incomingBackendConn)

	// Read from connection should get EOF when the connection was closed before
	// This blocks until the connection is actually closed or we read anything
	// (or the test-timeout kills the test)
	readBuffer := make([]byte, 1)
	n, err := incomingTestConn.Read(readBuffer)

	if !errors.Is(err, io.EOF) {
		t.Errorf("expected io.EOF after dial failure, got: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 bytes read after close, got %d", n)
	}
}

func TestTCPForwarderBackend_Handle_TracksActiveConnections(t *testing.T) {
	backend := newTCPForwarderBackend(config.TCPForwarderBackendConfig{
		Name:       "test-forwarder",
		TargetAddr: "example.com:80",
	})

	// Initially no connections
	if backend.activeConnections.Len() != 0 {
		t.Errorf("initial active connections = %d, want 0", backend.activeConnections.Len())
	}

	// Create successful dial mock
	targetConn1, _ := net.Pipe()
	defer targetConn1.Close()

	backend.dialer = &mockDialer{
		mockDialTimeout: func(_, _ string, _ time.Duration) (net.Conn, error) {
			return targetConn1, nil
		},
	}

	// Handle first connection
	incomingConn1, testConn1 := net.Pipe()
	defer incomingConn1.Close()
	defer testConn1.Close()

	backend.Handle(incomingConn1)

	// Should have 1 active connection
	if backend.activeConnections.Len() != 1 {
		t.Errorf("after first Handle(), active connections = %d, want 1", backend.activeConnections.Len())
	}

	// Handle second connection
	targetConn2, _ := net.Pipe()
	defer targetConn2.Close()

	backend.dialer = &mockDialer{
		mockDialTimeout: func(_, _ string, _ time.Duration) (net.Conn, error) {
			return targetConn2, nil
		},
	}

	incomingConn2, testConn2 := net.Pipe()
	defer incomingConn2.Close()
	defer testConn2.Close()

	backend.Handle(incomingConn2)

	// Should have 2 active connections
	if backend.activeConnections.Len() != 2 {
		t.Errorf("after second Handle(), active connections = %d, want 2", backend.activeConnections.Len())
	}
}

func TestTCPForwarderBackend_Handle_RemovesConnectionOnClose(t *testing.T) {
	targetBackendEnd, targetClientEnd := net.Pipe()
	defer targetClientEnd.Close()

	backend := newTCPForwarderBackend(config.TCPForwarderBackendConfig{
		Name:       "test-forwarder",
		TargetAddr: "example.com:80",
	})

	backend.dialer = &mockDialer{
		mockDialTimeout: func(_, _ string, _ time.Duration) (net.Conn, error) {
			return targetBackendEnd, nil
		},
	}

	incomingBackendConn, incomingTestConn := net.Pipe()

	backend.Handle(incomingBackendConn)

	// Verify connection was added
	if backend.activeConnections.Len() != 1 {
		t.Fatalf("after Handle(), active connections = %d, want 1", backend.activeConnections.Len())
	}

	// Close both ends of the connection
	incomingTestConn.Close()
	incomingBackendConn.Close()
	targetClientEnd.Close()
	targetBackendEnd.Close()

	// Give the PipeHelper's Close callback time to execute
	time.Sleep(50 * time.Millisecond)

	// Connection should be removed from active list
	if backend.activeConnections.Len() != 0 {
		t.Errorf("after Close(), active connections = %d, want 0", backend.activeConnections.Len())
	}
}

func TestTCPForwarderBackend_Handle_BidirectionalDataFlow(t *testing.T) {
	targetBackendEnd, targetClientEnd := net.Pipe()
	defer targetBackendEnd.Close()
	defer targetClientEnd.Close()

	backend := newTCPForwarderBackend(config.TCPForwarderBackendConfig{
		Name:       "test-forwarder",
		TargetAddr: "127.0.0.4:3000",
	})

	backend.dialer = &mockDialer{
		mockDialTimeout: func(_, _ string, _ time.Duration) (net.Conn, error) {
			return targetBackendEnd, nil
		},
	}

	incomingBackendConn, incomingTestConn := net.Pipe()
	defer incomingBackendConn.Close()
	defer incomingTestConn.Close()

	backend.Handle(incomingBackendConn)

	// Test multiple exchanges in both directions
	testCases := []struct {
		clientToServer string
		serverToClient string
	}{
		{"GET /", "200 OK"},
		{"GET /api", "404 Not Found"},
		{"POST /data", "201 Created"},
	}

	for i, tc := range testCases {
		// Client sends to server
		go func(data string) {
			incomingTestConn.Write([]byte(data))
		}(tc.clientToServer)

		readBuffer := make([]byte, len(tc.clientToServer))
		n, err := io.ReadFull(targetClientEnd, readBuffer)
		if err != nil {
			t.Fatalf("exchange %d: failed to read request: %v", i, err)
		}
		if string(readBuffer[:n]) != tc.clientToServer {
			t.Errorf("exchange %d: server received %q, want %q", i, string(readBuffer[:n]), tc.clientToServer)
		}

		// Server responds to client
		go func(data string) {
			targetClientEnd.Write([]byte(data))
		}(tc.serverToClient)

		readBuffer = make([]byte, len(tc.serverToClient))
		n, err = io.ReadFull(incomingTestConn, readBuffer)
		if err != nil {
			t.Fatalf("exchange %d: failed to read response: %v", i, err)
		}
		if string(readBuffer[:n]) != tc.serverToClient {
			t.Errorf("exchange %d: client received %q, want %q", i, string(readBuffer[:n]), tc.serverToClient)
		}
	}
}
