package utils

import (
	"fmt"
	"io"
	"log/slog"
	"net"
)

type PipeHelper struct {
	isClosed      bool
	sourceConn    net.Conn
	targetConn    net.Conn
	closeCallback func()
}

func NewPipeHelper(sourceConn net.Conn, targetConn net.Conn) *PipeHelper {
	self := &PipeHelper{
		isClosed:      false,
		sourceConn:    sourceConn,
		targetConn:    targetConn,
		closeCallback: nil,
	}

	go func() {
		_, err := io.Copy(sourceConn, targetConn)

		if err != nil && !self.isClosed {
			slog.Error("error while copying data", slog.Any("error", err))
		}

		self.Close()
	}()

	go func() {
		_, err := io.Copy(targetConn, sourceConn)

		if err != nil && !self.isClosed {
			slog.Error("error while copying data", slog.Any("error", err))
		}

		self.Close()
	}()

	return self
}

func (self *PipeHelper) OnClose(closeCallback func()) error {
	if self.closeCallback != nil {
		return fmt.Errorf("close callback already registered for pipehelper")
	}

	self.closeCallback = closeCallback

	return nil
}

func (self *PipeHelper) Close() {
	if self.isClosed {
		return
	}

	self.isClosed = true

	self.sourceConn.Close()
	self.targetConn.Close()

	if self.closeCallback != nil {
		self.closeCallback()
	}
}
