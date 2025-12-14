package backends

import (
	"container/list"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/sateffen/pluggo/config"
	"github.com/sateffen/pluggo/utils"
)

type wolForwarderBackend struct {
	name              string
	activeConnections *list.List
	connectionsMutex  sync.Mutex
	wolHelper         *utils.WoLHelper
	targetAddr        string
}

func NewWoLForwarderBackend(conf config.WoLForwarderBackendConfig) (*wolForwarderBackend, error) {
	wolHelper, err := utils.NewWoLHelper(conf.WoLMACAddr, conf.WoLBroadcastAddr)
	if err != nil {
		return nil, fmt.Errorf("could not create WoL helper: %q", err)
	}

	return &wolForwarderBackend{
		name:              conf.Name,
		activeConnections: list.New(),
		wolHelper:         wolHelper,
		targetAddr:        conf.TargetAddr,
	}, nil
}

func (backend *wolForwarderBackend) GetName() string {
	return backend.name
}

func (backend *wolForwarderBackend) Handle(connection net.Conn) {
	connectionToTarget, err := backend.tryDial()
	if err != nil {
		slog.Info(
			"backend could not connect to target",
			slog.String("targetAddr", backend.targetAddr),
			slog.String("name", backend.name),
			slog.Any("error", err),
		)
		connection.Close()
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

func (backend *wolForwarderBackend) tryDial() (net.Conn, error) {
	// First, try a quick connection to see if target is already awake
	targetConnection, err := net.DialTimeout("tcp", backend.targetAddr, 2*time.Second)
	if err == nil {
		return targetConnection, nil
	}

	// Target is unreachable - send WoL packet and retry
	slog.Debug("failed to connect to host, sending wol to wake it up", slog.String("targetAddr", backend.targetAddr))
	err = backend.wolHelper.SendWOLPaket()
	if err != nil {
		return nil, fmt.Errorf("could not send wol magic paket: %q", err)
	}

	// Then we retry for something ~2min, else we give up.
	for i := range 50 {
		slog.Debug("trying to connect to host", slog.String("targetAddr", backend.targetAddr), slog.Int("retryCount", i))

		time.Sleep(500 * time.Millisecond)
		targetConnection, err = net.DialTimeout("tcp", backend.targetAddr, 2*time.Second)
		if err == nil {
			return targetConnection, nil
		}
	}

	return nil, fmt.Errorf("timeout while waiting for target with addr \"%s\"", backend.targetAddr)
}
