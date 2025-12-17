package helper

import (
	"io"
	"net"
	"testing"
	"time"
)

func TestPipeHelper_OnClose(t *testing.T) {
	sourceConn := &mockConn{}
	targetConn := &mockConn{}
	pipeHelper := NewPipeHelper(sourceConn, targetConn)

	onCloseGotCalled := make(chan bool, 1)
	err := pipeHelper.OnClose(func() { onCloseGotCalled <- true })
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	pipeHelper.Close()

	select {
	case <-onCloseGotCalled:
		// Callback got executed, so both connection should be closed
		if sourceConn.closeCallCount != 1 || targetConn.closeCallCount != 1 {
			t.Error("connections not closed")
		}
	case <-time.After(100 * time.Millisecond):
		// Else if the callback was not called after 100ms, we assume something is wrong
		t.Error("close callback not called")
	}
}

func TestPipeHelper_OnClose_ErrorOnAddCallbackTwice(t *testing.T) {
	sourceConn := &mockConn{}
	targetConn := &mockConn{}
	pipeHelper := NewPipeHelper(sourceConn, targetConn)

	err := pipeHelper.OnClose(func() {})
	if err != nil {
		t.Errorf("unexpected error on first OnClose register: %v", err)
	}

	err = pipeHelper.OnClose(func() {})
	if err == nil {
		t.Error("expected error when registering OnClose twice")
	}
}

func TestPipeHelper_OnClose_ErrorOnAddCallbackAfterClose(t *testing.T) {
	sourceConn := &mockConn{}
	targetConn := &mockConn{}
	pipeHelper := NewPipeHelper(sourceConn, targetConn)

	pipeHelper.Close()
	err := pipeHelper.OnClose(func() {})
	if err == nil {
		t.Errorf("did not receive error during OnClose on already closed PipeHelper")
	}
}

func TestPipeHelper_Close_ExecuteOnceEvenWhenCalledMultipleTimes(t *testing.T) {
	sourceConn := &mockConn{}
	targetConn := &mockConn{}
	pipeHelper := NewPipeHelper(sourceConn, targetConn)

	pipeHelper.Close()
	pipeHelper.Close()
	pipeHelper.Close()

	// Even if we call Close multiple times, the connections should only get closed once
	if sourceConn.closeCallCount != 1 || targetConn.closeCallCount != 1 {
		t.Error("connections not closed exactly once")
	}
}

func TestPipeHelper_DoesPiping(t *testing.T) {
	incomingClientConn, incomingBackendConn := net.Pipe()
	defer incomingClientConn.Close()
	defer incomingBackendConn.Close()
	targetClientConn, targetBackendConn := net.Pipe()
	defer targetClientConn.Close()
	defer targetBackendConn.Close()

	pipeHelper := NewPipeHelper(incomingBackendConn, targetBackendConn)
	defer pipeHelper.Close()

	// Verify data flows through: incoming -> target
	testData := []byte("hello target")
	go func() {
		incomingClientConn.Write(testData)
	}()

	readBuffer := make([]byte, len(testData))
	n, err := io.ReadFull(targetClientConn, readBuffer)
	if err != nil {
		t.Fatalf("failed to read from target: %v", err)
	}
	if string(readBuffer[:n]) != string(testData) {
		t.Errorf("target received %q, want %q", string(readBuffer[:n]), string(testData))
	}

	// Verify data flows back: target -> incoming
	responseData := []byte("hello client")
	go func() {
		targetClientConn.Write(responseData)
	}()

	readBuffer = make([]byte, len(responseData))
	n, err = io.ReadFull(incomingClientConn, readBuffer)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}
	if string(readBuffer[:n]) != string(responseData) {
		t.Errorf("client received %q, want %q", string(readBuffer[:n]), string(responseData))
	}
}
