package helper

import (
	"errors"
	"io"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
)

type PipeHelper struct {
	isClosed      atomic.Bool
	closeOnce     sync.Once
	sourceConn    net.Conn
	targetConn    net.Conn
	closeCallback func()
}

// NewPipeHelper creates a new instance of PipeHelper.
// This will start go-routines that start piping data sourceConn -> targetConn and targetConn -> sourceConn.
// The created PipeHelper will take ownership of given connections.
func NewPipeHelper(sourceConn net.Conn, targetConn net.Conn) *PipeHelper {
	pipeHelper := &PipeHelper{
		// isClosed is false by default
		sourceConn:    sourceConn,
		targetConn:    targetConn,
		closeCallback: nil,
	}

	go func() {
		_, err := io.Copy(sourceConn, targetConn)

		if err != nil && !pipeHelper.isClosed.Load() {
			slog.Debug("connection closed during copy", slog.Any("error", err))
		}

		pipeHelper.Close()
	}()

	go func() {
		_, err := io.Copy(targetConn, sourceConn)

		if err != nil && !pipeHelper.isClosed.Load() {
			slog.Debug("connection closed during copy", slog.Any("error", err))
		}

		pipeHelper.Close()
	}()

	return pipeHelper
}

// OnClose registers a callback that gets called, when the connections managed by this PipeHelper instance
// get closed. Only one callback can be registered, and a registered callback can't get unregistered.
func (ph *PipeHelper) OnClose(closeCallback func()) error {
	if ph.closeCallback != nil {
		return errors.New("pipehelper close callback already registered")
	}
	if ph.isClosed.Load() {
		return errors.New("pipehelper is already closed, can't register a close callback anymore")
	}

	ph.closeCallback = closeCallback

	return nil
}

// Close closes this PipeHelper by closing all owned connections and calling the registered OnClose callback
// if present.
func (ph *PipeHelper) Close() {
	ph.closeOnce.Do(func() {
		ph.isClosed.Store(true)

		if err := ph.sourceConn.Close(); err != nil {
			slog.Warn("pipehelper couldn't properly close source connection", slog.Any("error", err))
		}
		if err := ph.targetConn.Close(); err != nil {
			slog.Warn("pipehelper couldn't properly close target connection", slog.Any("error", err))
		}

		if ph.closeCallback != nil {
			ph.closeCallback()
		}
	})
}
