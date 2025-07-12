package backends

import (
	"container/list"
	"log/slog"
	"net"

	"github.com/sateffen/pluggo/config"
	"github.com/sateffen/pluggo/utils"
)

type tcpForwarderBackend struct {
	name              string
	activeConnections *list.List
	targetAddr        string
}

func NewTCPForwarderBackend(conf config.TCPForwarderBackendConfig) (*tcpForwarderBackend, error) {
	return &tcpForwarderBackend{
		name:              conf.Name,
		activeConnections: list.New(),
		targetAddr:        conf.TargetAddr,
	}, nil
}

func (self *tcpForwarderBackend) GetName() string {
	return self.name
}

func (self *tcpForwarderBackend) Handle(connection net.Conn) {
	connectionToTarget, err := net.Dial("tcp", self.targetAddr)
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
