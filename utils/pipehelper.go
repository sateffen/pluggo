package utils

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
)

type PipeHelper struct {
	isClosed      bool
	closeOnce     sync.Once
	sourceConn    net.Conn
	targetConn    net.Conn
	closeCallback func()
}

func NewPipeHelper(sourceConn net.Conn, targetConn net.Conn) *PipeHelper {
	pipeHelper := &PipeHelper{
		isClosed:      false,
		sourceConn:    sourceConn,
		targetConn:    targetConn,
		closeCallback: nil,
	}

	go func() {
		_, err := io.Copy(sourceConn, targetConn)

		if err != nil && !pipeHelper.isClosed {
			slog.Debug("connection closed during copy", slog.Any("error", err))
		}

		pipeHelper.Close()
	}()

	go func() {
		_, err := io.Copy(targetConn, sourceConn)

		if err != nil && !pipeHelper.isClosed {
			slog.Debug("connection closed during copy", slog.Any("error", err))
		}

		pipeHelper.Close()
	}()

	return pipeHelper
}

func (pipeHelper *PipeHelper) OnClose(closeCallback func()) error {
	if pipeHelper.closeCallback != nil {
		return fmt.Errorf("close callback already registered for pipehelper")
	}

	pipeHelper.closeCallback = closeCallback

	return nil
}

func (pipeHelper *PipeHelper) Close() {
	pipeHelper.closeOnce.Do(func() {
		pipeHelper.isClosed = true

		pipeHelper.sourceConn.Close()
		pipeHelper.targetConn.Close()

		if pipeHelper.closeCallback != nil {
			pipeHelper.closeCallback()
		}
	})
}
