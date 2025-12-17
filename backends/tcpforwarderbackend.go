package backends

import (
	"container/list"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/sateffen/pluggo/backends/helper"
	"github.com/sateffen/pluggo/config"
)

const tcpDialTimeout = 10 * time.Second

type tcpForwarderBackend struct {
	name              string
	activeConnections *list.List
	connectionsMutex  sync.Mutex
	targetAddr        string
	dialer            dialer
}

// newTCPForwarderBackend creates a new instance of tcpForwarderBackend, preparing it with all necessary dependencies.
func newTCPForwarderBackend(conf config.TCPForwarderBackendConfig) *tcpForwarderBackend {
	return &tcpForwarderBackend{
		name:              conf.Name,
		activeConnections: list.New(),
		targetAddr:        conf.TargetAddr,
		dialer:            defaultDialer{},
	}
}

// GetName returns the name of the current tcpForwarderBackend instance.
func (be *tcpForwarderBackend) GetName() string {
	return be.name
}

// Handle handles given connection by trying to dial the target host. If the target host is reachable,
// a pipe will get generated, else the connection gets closed.
// Handle takes ownership of given connection.
func (be *tcpForwarderBackend) Handle(connection net.Conn) {
	connectionToTarget, err := be.dialer.DialTimeout("tcp", be.targetAddr, tcpDialTimeout)
	if err != nil {
		slog.Info(
			"backend could not connect to target",
			slog.String("targetAddr", be.targetAddr),
			slog.String("name", be.name),
			slog.Any("error", err),
		)

		if err = connection.Close(); err != nil {
			slog.Warn("could not properly close incoming connection after dialer timeout", slog.Any("error", err))
		}

		return
	}

	pipeHelper := helper.NewPipeHelper(connection, connectionToTarget)

	be.connectionsMutex.Lock()
	listElement := be.activeConnections.PushBack(pipeHelper)
	be.connectionsMutex.Unlock()

	//nolint:gosec // if an error happens here, the matrix is broken
	pipeHelper.OnClose(func() {
		be.connectionsMutex.Lock()
		be.activeConnections.Remove(listElement)
		be.connectionsMutex.Unlock()
	})
}
