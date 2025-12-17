package frontends

import (
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/sateffen/pluggo/backends"
	"github.com/sateffen/pluggo/config"
)

// createTestBackendList creates a backend list with echo backends for testing.
func createTestBackendList(backendNames ...string) *backends.BackendList {
	conf := config.BackendConfigs{
		Echo: make([]config.EchoBackendConfig, len(backendNames)),
	}
	for i, name := range backendNames {
		conf.Echo[i] = config.EchoBackendConfig{Name: name}
	}
	bl, _ := backends.NewBackendList(conf)
	return bl
}

func TestTCPFrontend_GetName(t *testing.T) {
	backendList := createTestBackendList("test-backend")

	frontend, err := newTCPFrontend(config.TCPFrontendConfig{
		Name:       "test-frontend",
		ListenAddr: "127.0.0.1:8080",
		Target:     "test-backend",
	}, backendList)
	if err != nil {
		t.Fatalf("newTCPFrontend() failed: %v", err)
	}

	if got := frontend.GetName(); got != "test-frontend" {
		t.Errorf("GetName() = %q, want %q", got, "test-frontend")
	}
}

func TestTCPFrontend_NewTCPFrontend_InvalidListenAddr(t *testing.T) {
	backendList := createTestBackendList("test-backend")

	_, err := newTCPFrontend(config.TCPFrontendConfig{
		Name:       "test-frontend",
		ListenAddr: "invalid-address",
		Target:     "test-backend",
	}, backendList)

	if err == nil {
		t.Fatal("expected newTCPFrontend() to fail with invalid listen address")
	}
}

func TestTCPFrontend_NewTCPFrontend_NonExistentBackend(t *testing.T) {
	backendList := createTestBackendList("other-backend")

	_, err := newTCPFrontend(config.TCPFrontendConfig{
		Name:       "test-frontend",
		ListenAddr: "127.0.0.1:8080",
		Target:     "non-existent-backend",
	}, backendList)

	if err == nil {
		t.Fatal("expected newTCPFrontend() to fail with non-existent backend")
	}
}

func TestTCPFrontend_Listen_ListenTCPFails(t *testing.T) {
	backendList := createTestBackendList("test-backend")

	frontend, err := newTCPFrontend(config.TCPFrontendConfig{
		Name:       "test-frontend",
		ListenAddr: "127.0.0.1:8080",
		Target:     "test-backend",
	}, backendList)
	if err != nil {
		t.Fatalf("newTCPFrontend() failed: %v", err)
	}

	// Replace factory with mock that fails
	frontend.listenerFactory = &mockTCPListenerFactory{
		mockListenTCP: func(_ string, _ *net.TCPAddr) (tcpListener, error) {
			return nil, errors.New("failed to listen")
		},
	}

	err = frontend.Listen()
	if err == nil {
		t.Fatal("expected Listen() to fail when ListenTCP fails")
	}
}

func TestTCPFrontend_Listen_AcceptsConnectionsAndHandles(t *testing.T) {
	// Create pipe to simulate incoming connection
	incomingBackendConn, incomingTestConn := net.Pipe()
	defer incomingBackendConn.Close()
	defer incomingTestConn.Close()

	backendList := createTestBackendList("test-backend")

	frontend, err := newTCPFrontend(config.TCPFrontendConfig{
		Name:       "test-frontend",
		ListenAddr: "127.0.0.1:8080",
		Target:     "test-backend",
	}, backendList)
	if err != nil {
		t.Fatalf("newTCPFrontend() failed: %v", err)
	}

	// Replace backend with mock for unit testing
	handleCalled := make(chan bool, 1)
	frontend.targetBackend = &mockBackend{
		name: "test-backend",
		mockHandle: func(conn net.Conn) {
			handleCalled <- true
			conn.Close()
		},
	}

	// Mock listener that accepts once then blocks
	acceptCount := 0
	acceptShouldBlock := make(chan bool)
	mockListener := &mockTCPListener{
		mockAccept: func() (net.Conn, error) {
			acceptCount++
			if acceptCount == 1 {
				return incomingBackendConn, nil
			}
			// Block until Close() is called
			<-acceptShouldBlock
			return nil, errors.New("listener closed")
		},
		mockClose: func() error {
			close(acceptShouldBlock)
			return nil
		},
	}

	frontend.listenerFactory = &mockTCPListenerFactory{
		mockListenTCP: func(_ string, _ *net.TCPAddr) (tcpListener, error) {
			return mockListener, nil
		},
	}

	// Run Listen in goroutine
	listenDone := make(chan error, 1)
	go func() {
		listenDone <- frontend.Listen()
		close(listenDone)
	}()

	// Wait for handler to be called
	select {
	case <-handleCalled:
		// Success - handler was called
	case <-time.After(100 * time.Millisecond):
		t.Fatal("backend.Handle() was not called in appropriate time")
	}

	// Close frontend to stop listening
	frontend.Close()

	// Wait for Listen to return
	select {
	case err = <-listenDone:
		if err != nil {
			t.Errorf("Listen() returned error: %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Listen() did not return")
	}
}

func TestTCPFrontend_Listen_AcceptErrorContinuesLoop(t *testing.T) {
	incomingBackendConn, incomingTestConn := net.Pipe()
	defer incomingBackendConn.Close()
	defer incomingTestConn.Close()

	backendList := createTestBackendList("test-backend")

	frontend, err := newTCPFrontend(config.TCPFrontendConfig{
		Name:       "test-frontend",
		ListenAddr: "127.0.0.1:8080",
		Target:     "test-backend",
	}, backendList)
	if err != nil {
		t.Fatalf("newTCPFrontend() failed: %v", err)
	}

	// Replace backend with mock
	handleCalled := make(chan bool, 1)
	frontend.targetBackend = &mockBackend{
		name: "test-backend",
		mockHandle: func(conn net.Conn) {
			handleCalled <- true
			conn.Close()
		},
	}

	// Mock listener that fails once, then succeeds, then blocks
	acceptCount := 0
	acceptShouldBlock := make(chan bool)
	mockListener := &mockTCPListener{
		mockAccept: func() (net.Conn, error) {
			acceptCount++
			switch acceptCount {
			case 1:
				// First accept fails (simulates temporary error)
				return nil, errors.New("temporary accept error")
			case 2:
				// Second accept succeeds
				return incomingBackendConn, nil
			default:
				// Block until Close() is called
				<-acceptShouldBlock
				return nil, errors.New("listener closed")
			}
		},
		mockClose: func() error {
			close(acceptShouldBlock)
			return nil
		},
	}

	frontend.listenerFactory = &mockTCPListenerFactory{
		mockListenTCP: func(_ string, _ *net.TCPAddr) (tcpListener, error) {
			return mockListener, nil
		},
	}

	// Run Listen in goroutine
	listenDone := make(chan error, 1)
	go func() {
		listenDone <- frontend.Listen()
		close(listenDone)
	}()

	// Wait for handler to be called (should happen on second accept)
	select {
	case <-handleCalled:
		// Success - handler was called despite first error
	case <-time.After(100 * time.Millisecond):
		t.Fatal("backend.Handle() was not called after temporary accept error")
	}

	// Close frontend
	frontend.Close()

	// Wait for Listen to return
	select {
	case <-listenDone:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Listen() did not return in appropriate time")
	}
}

func TestTCPFrontend_Close_ExitsListenLoop(t *testing.T) {
	backendList := createTestBackendList("test-backend")

	frontend, err := newTCPFrontend(config.TCPFrontendConfig{
		Name:       "test-frontend",
		ListenAddr: "127.0.0.1:8080",
		Target:     "test-backend",
	}, backendList)
	if err != nil {
		t.Fatalf("newTCPFrontend() failed: %v", err)
	}

	closeCalled := make(chan bool, 1)
	acceptShouldBlock := make(chan bool)
	mockListener := &mockTCPListener{
		mockAccept: func() (net.Conn, error) {
			// Block until Close() is called
			<-acceptShouldBlock
			return nil, errors.New("accept error after close")
		},
		mockClose: func() error {
			closeCalled <- true
			close(acceptShouldBlock)
			return nil
		},
	}

	frontend.listenerFactory = &mockTCPListenerFactory{
		mockListenTCP: func(_ string, _ *net.TCPAddr) (tcpListener, error) {
			return mockListener, nil
		},
	}

	// Start listening
	listenDone := make(chan error, 1)
	go func() {
		listenDone <- frontend.Listen()
		close(listenDone)
	}()

	// Give Listen time to start
	time.Sleep(20 * time.Millisecond)

	// Call Close
	err = frontend.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}

	// Verify listener.Close() was called
	select {
	case <-closeCalled:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("listener.Close() was not called")
	}

	// Verify Listen() exits
	select {
	case <-listenDone:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Listen() did not exit after Close()")
	}
}

func TestTCPFrontend_Close_NoListener(t *testing.T) {
	backendList := createTestBackendList("test-backend")

	frontend, err := newTCPFrontend(config.TCPFrontendConfig{
		Name:       "test-frontend",
		ListenAddr: "127.0.0.1:8080",
		Target:     "test-backend",
	}, backendList)
	if err != nil {
		t.Fatalf("newTCPFrontend() failed: %v", err)
	}

	// Close without starting listener
	err = frontend.Close()
	if err != nil {
		t.Errorf("Close() with no listener should return nil, got: %v", err)
	}
}

func TestTCPFrontend_Close_DoubleClose(t *testing.T) {
	backendList := createTestBackendList("test-backend")

	frontend, err := newTCPFrontend(config.TCPFrontendConfig{
		Name:       "test-frontend",
		ListenAddr: "127.0.0.1:8080",
		Target:     "test-backend",
	}, backendList)
	if err != nil {
		t.Fatalf("newTCPFrontend() failed: %v", err)
	}

	closeCallCount := 0
	acceptShouldBlock := make(chan bool)
	mockListener := &mockTCPListener{
		mockAccept: func() (net.Conn, error) {
			<-acceptShouldBlock
			return nil, errors.New("closed")
		},
		mockClose: func() error {
			closeCallCount++
			close(acceptShouldBlock)
			return nil
		},
	}

	frontend.listenerFactory = &mockTCPListenerFactory{
		mockListenTCP: func(_ string, _ *net.TCPAddr) (tcpListener, error) {
			return mockListener, nil
		},
	}

	// Start and stop listening
	listenDone := make(chan error, 1)
	go func() {
		listenDone <- frontend.Listen()
		close(listenDone)
	}()

	time.Sleep(20 * time.Millisecond)

	// First close
	err = frontend.Close()
	if err != nil {
		t.Errorf("first Close() returned error: %v", err)
	}

	<-listenDone

	// Second close
	err = frontend.Close()
	if err != nil {
		t.Errorf("second Close() should return nil, got: %v", err)
	}

	// Verify listener.Close() was only called once
	if closeCallCount != 1 {
		t.Errorf("listener.Close() called %d times, want 1", closeCallCount)
	}
}

func TestTCPFrontend_Close_CloseError(t *testing.T) {
	backendList := createTestBackendList("test-backend")

	frontend, err := newTCPFrontend(config.TCPFrontendConfig{
		Name:       "test-frontend",
		ListenAddr: "127.0.0.1:8080",
		Target:     "test-backend",
	}, backendList)
	if err != nil {
		t.Fatalf("newTCPFrontend() failed: %v", err)
	}

	acceptShouldBlock := make(chan bool)
	mockListener := &mockTCPListener{
		mockAccept: func() (net.Conn, error) {
			<-acceptShouldBlock
			return nil, errors.New("closed")
		},
		mockClose: func() error {
			close(acceptShouldBlock)
			return errors.New("close error")
		},
	}

	frontend.listenerFactory = &mockTCPListenerFactory{
		mockListenTCP: func(_ string, _ *net.TCPAddr) (tcpListener, error) {
			return mockListener, nil
		},
	}

	// Start listening
	listenDone := make(chan error, 1)
	go func() {
		listenDone <- frontend.Listen()
		close(listenDone)
	}()

	time.Sleep(20 * time.Millisecond)

	// Close should return error
	err = frontend.Close()
	if err == nil {
		t.Fatal("expected Close() to return error when listener.Close() fails")
	}

	<-listenDone
}

func TestTCPFrontend_ConcurrentClose(t *testing.T) {
	backendList := createTestBackendList("test-backend")

	frontend, err := newTCPFrontend(config.TCPFrontendConfig{
		Name:       "test-frontend",
		ListenAddr: "127.0.0.1:8080",
		Target:     "test-backend",
	}, backendList)
	if err != nil {
		t.Fatalf("newTCPFrontend() failed: %v", err)
	}

	closeCallCount := 0
	var closeCountMutex sync.Mutex
	acceptShouldBlock := make(chan bool)
	mockListener := &mockTCPListener{
		mockAccept: func() (net.Conn, error) {
			<-acceptShouldBlock
			return nil, errors.New("closed")
		},
		mockClose: func() error {
			closeCountMutex.Lock()
			defer closeCountMutex.Unlock()

			closeCallCount++
			if closeCallCount == 1 {
				close(acceptShouldBlock)
			}

			return nil
		},
	}

	frontend.listenerFactory = &mockTCPListenerFactory{
		mockListenTCP: func(_ string, _ *net.TCPAddr) (tcpListener, error) {
			return mockListener, nil
		},
	}

	// Start listening
	listenDone := make(chan error, 1)
	go func() {
		listenDone <- frontend.Listen()
		close(listenDone)
	}()

	time.Sleep(20 * time.Millisecond)

	// Call Close concurrently from multiple goroutines
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			frontend.Close()
		}()
	}

	wg.Wait()
	<-listenDone

	// Verify listener.Close() was only called once despite concurrent Close calls
	closeCountMutex.Lock()
	finalCount := closeCallCount
	closeCountMutex.Unlock()

	if finalCount != 1 {
		t.Errorf("listener.Close() called %d times, want 1", finalCount)
	}
}
