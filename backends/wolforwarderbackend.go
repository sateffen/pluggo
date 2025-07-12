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

func (self *wolForwarderBackend) GetName() string {
	return self.name
}

func (self *wolForwarderBackend) Handle(connection net.Conn) {
	connectionToTarget, err := self.tryDial()
	if err != nil {
		slog.Info(
			"backend could not connect to target",
			slog.String("targetAddr", self.targetAddr),
			slog.String("name", self.name),
			slog.Any("error", err),
		)
		return
	}

	pipeHelper := utils.NewPipeHelper(connection, connectionToTarget)

	listElement := self.activeConnections.PushBack(pipeHelper)

	pipeHelper.OnClose(func() {
		self.activeConnections.Remove(listElement)
	})
}

func (self *wolForwarderBackend) tryDial() (net.Conn, error) {
	timeoutTime := time.Now().Add(60 * time.Second)

	for timeoutTime.After(time.Now()) {
		if self.activeConnections.Len() == 0 {
			err := self.wolHelper.SendWOLPaket()
			if err != nil {
				return nil, fmt.Errorf("could not send wol magic paket: %q", err)
			}
		}

		targetConnection, err := net.Dial("tcp", self.targetAddr)
		if err == nil {
			return targetConnection, nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	return nil, fmt.Errorf("timeout while waiting for target with addr \"%s\"", self.targetAddr)
}
