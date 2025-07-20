package backends

import (
	"container/list"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/sateffen/pluggo/config"
	"github.com/sateffen/pluggo/utils"
)

type wolForwarderBackend struct {
	name              string
	activeConnections *list.List
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
		return
	}

	pipeHelper := utils.NewPipeHelper(connection, connectionToTarget)

	listElement := backend.activeConnections.PushBack(pipeHelper)

	pipeHelper.OnClose(func() {
		backend.activeConnections.Remove(listElement)
	})
}

func (backend *wolForwarderBackend) tryDial() (net.Conn, error) {
	timeoutTime := time.Now().Add(60 * time.Second)

	for timeoutTime.After(time.Now()) {
		if backend.activeConnections.Len() == 0 {
			err := backend.wolHelper.SendWOLPaket()
			if err != nil {
				return nil, fmt.Errorf("could not send wol magic paket: %q", err)
			}
		}

		targetConnection, err := net.Dial("tcp", backend.targetAddr)
		if err == nil {
			return targetConnection, nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	return nil, fmt.Errorf("timeout while waiting for target with addr \"%s\"", backend.targetAddr)
}
