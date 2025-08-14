package backends

import (
	"container/list"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/sateffen/pluggo/config"
	"github.com/sateffen/pluggo/utils"
)

type tcpForwarderBackend struct {
	name              string
	activeConnections *list.List
	connectionsMutex  sync.Mutex
	targetAddr        string
}

func NewTCPForwarderBackend(conf config.TCPForwarderBackendConfig) (*tcpForwarderBackend, error) {
	return &tcpForwarderBackend{
		name:              conf.Name,
		activeConnections: list.New(),
		targetAddr:        conf.TargetAddr,
	}, nil
}

func (backend *tcpForwarderBackend) GetName() string {
	return backend.name
}

func (backend *tcpForwarderBackend) Handle(connection net.Conn) {
	connectionToTarget, err := net.DialTimeout("tcp", backend.targetAddr, 10*time.Second)
	if err != nil {
		slog.Info(
			"backend could not connect to target",
			slog.String("targetAddr", backend.targetAddr),
			slog.String("name", backend.name),
			slog.Any("error", err),
		)
		return
	}

	pipeHelper := utils.NewPipeHelper(connection, connectionToTarget)

	backend.connectionsMutex.Lock()
	listElement := backend.activeConnections.PushBack(pipeHelper)
	backend.connectionsMutex.Unlock()

	pipeHelper.OnClose(func() {
		backend.connectionsMutex.Lock()
		backend.activeConnections.Remove(listElement)
		backend.connectionsMutex.Unlock()
	})
}
