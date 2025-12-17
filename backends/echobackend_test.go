package backends

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/sateffen/pluggo/config"
)

func TestEchoBackend_GetName(t *testing.T) {
	backend := newEchoBackend(config.EchoBackendConfig{
		Name: "test-echo",
	})

	if got := backend.GetName(); got != "test-echo" {
		t.Errorf("GetName() = %q, want %q", got, "test-echo")
	}
}

func TestEchoBackend_Handle_EchoesData(t *testing.T) {
	// Create the backend
	backend := newEchoBackend(config.EchoBackendConfig{
		Name: "test-echo",
	})

	// Create a pipe to simulate a connection
	// backendConn is what the backend receives (like from Accept())
	// testConn is what we control in the test to send/receive data
	backendConn, testConn := net.Pipe()
	defer backendConn.Close()
	defer testConn.Close()

	// Call Handle so the echoBackend does its thing
	backend.Handle(backendConn)

	// Write data to the connection
	testData := []byte("hello echo")
	go func() {
		_, err := testConn.Write(testData)
		if err != nil {
			t.Errorf("failed to write test data: %v", err)
		}
	}()

	// Read the echoed data back
	readBuffer := make([]byte, len(testData))
	n, err := io.ReadFull(testConn, readBuffer)
	if err != nil {
		t.Fatalf("failed to read echoed data: %v", err)
	}

	if n != len(testData) {
		t.Errorf("read %d bytes, want %d", n, len(testData))
	}

	if string(readBuffer) != string(testData) {
		t.Errorf("echoed data = %q, want %q", string(readBuffer), string(testData))
	}
}

func TestEchoBackend_Handle_TracksActiveConnections(t *testing.T) {
	backend := newEchoBackend(config.EchoBackendConfig{
		Name: "test-echo",
	})

	// Initially no connections
	if backend.activeConnections.Len() != 0 {
		t.Errorf("initial active connections = %d, want 0", backend.activeConnections.Len())
	}

	// Create and handle first connection
	backendConn1, testConn1 := net.Pipe()
	defer backendConn1.Close()
	defer testConn1.Close()

	backend.Handle(backendConn1)

	// Should have 1 active connection
	if backend.activeConnections.Len() != 1 {
		t.Errorf("after first Handle(), active connections = %d, want 1", backend.activeConnections.Len())
	}

	// Create and handle second connection
	backendConn2, testConn2 := net.Pipe()
	defer backendConn2.Close()
	defer testConn2.Close()

	backend.Handle(backendConn2)

	// Should have 2 active connections
	if backend.activeConnections.Len() != 2 {
		t.Errorf("after second Handle(), active connections = %d, want 2", backend.activeConnections.Len())
	}
}

func TestEchoBackend_Handle_RemovesConnectionOnClose(t *testing.T) {
	backend := newEchoBackend(config.EchoBackendConfig{
		Name: "test-echo",
	})

	backendConn, testConn := net.Pipe()

	backend.Handle(backendConn)

	// Verify connection was added
	if backend.activeConnections.Len() != 1 {
		t.Fatalf("after Handle(), active connections = %d, want 1", backend.activeConnections.Len())
	}

	// Close the connection
	testConn.Close()
	backendConn.Close()

	// Give the PipeHelper's Close callback time to execute
	time.Sleep(50 * time.Millisecond)

	// Connection should be removed from active list
	if backend.activeConnections.Len() != 0 {
		t.Errorf("after Close(), active connections = %d, want 0", backend.activeConnections.Len())
	}
}

func TestEchoBackend_Handle_MultipleEchoRoundTrips(t *testing.T) {
	backend := newEchoBackend(config.EchoBackendConfig{
		Name: "test-echo",
	})

	backendConn, testConn := net.Pipe()
	defer backendConn.Close()
	defer testConn.Close()

	backend.Handle(backendConn)

	// Test multiple round-trips
	testCases := []string{
		"first message",
		"second message",
		"third message with more data",
	}

	for i, testData := range testCases {
		// Write data
		go func(data string) {
			_, err := testConn.Write([]byte(data))
			if err != nil {
				t.Errorf("write %d failed: %v", i, err)
			}
		}(testData)

		// Read echoed data
		readBuffer := make([]byte, len(testData))
		n, err := io.ReadFull(testConn, readBuffer)
		if err != nil {
			t.Fatalf("read %d failed: %v", i, err)
		}

		if string(readBuffer[:n]) != testData {
			t.Errorf("echo %d: got %q, want %q", i, string(readBuffer[:n]), testData)
		}
	}
}

func TestEchoBackend_Handle_ConcurrentConnections(t *testing.T) {
	backend := newEchoBackend(config.EchoBackendConfig{
		Name: "test-echo",
	})

	// Create multiple concurrent connections
	numConns := 5
	backendConns := make([]net.Conn, numConns)
	testConns := make([]net.Conn, numConns)

	for i := range numConns {
		backendConn, testConn := net.Pipe()
		backendConns[i] = backendConn
		testConns[i] = testConn
		backend.Handle(backendConn)
	}

	// Should have all connections tracked
	if backend.activeConnections.Len() != numConns {
		t.Errorf("active connections = %d, want %d", backend.activeConnections.Len(), numConns)
	}

	// Test each connection can echo independently
	for i, testConn := range testConns {
		testData := []byte("message " + string(rune('0'+i)))

		go func(c net.Conn, data []byte) {
			c.Write(data)
		}(testConn, testData)

		readBuffer := make([]byte, len(testData))
		n, err := io.ReadFull(testConn, readBuffer)
		if err != nil {
			t.Errorf("connection %d read failed: %v", i, err)
			continue
		}

		if string(readBuffer[:n]) != string(testData) {
			t.Errorf("connection %d: got %q, want %q", i, string(readBuffer[:n]), string(testData))
		}
	}

	// Cleanup
	for i := range testConns {
		testConns[i].Close()
		backendConns[i].Close()
	}

	// Wait for cleanup
	time.Sleep(50 * time.Millisecond)

	// All connections should be removed
	if backend.activeConnections.Len() != 0 {
		t.Errorf("after cleanup, active connections = %d, want 0", backend.activeConnections.Len())
	}
}
