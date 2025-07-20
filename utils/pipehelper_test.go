package utils

import (
	"errors"
	"net"
	"testing"
	"time"
)

type mockConn struct {
	closed bool
}

func (m *mockConn) Read(b []byte) (n int, err error)   { return 0, errors.New("not implemented") }
func (m *mockConn) Write(b []byte) (n int, err error)  { return 0, errors.New("not implemented") }
func (m *mockConn) Close() error                       { m.closed = true; return nil }
func (m *mockConn) LocalAddr() net.Addr                { return nil }
func (m *mockConn) RemoteAddr() net.Addr               { return nil }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

func TestPipeHelperOnClose(t *testing.T) {
	sourceConn := &mockConn{}
	targetConn := &mockConn{}
	pipeHelper := NewPipeHelper(sourceConn, targetConn)

	onCloseGotCalled := false
	err := pipeHelper.OnClose(func() { onCloseGotCalled = true })
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	pipeHelper.Close()

	if !onCloseGotCalled {
		t.Error("close callback not called")
	}

	if !sourceConn.closed || !targetConn.closed {
		t.Error("connections not closed")
	}
}

func TestPipeHelperOnCloseTwice(t *testing.T) {
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
