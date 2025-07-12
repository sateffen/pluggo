package backends

import (
	"container/list"
	"net"

	"github.com/sateffen/pluggo/config"
	"github.com/sateffen/pluggo/utils"
)

type echoBackend struct {
	name              string
	activeConnections *list.List
}

func NewEchoBackend(conf config.EchoBackendConfig) (*echoBackend, error) {
	return &echoBackend{
		name:              conf.Name,
		activeConnections: list.New(),
	}, nil
}

func (self *echoBackend) GetName() string {
	return self.name
}

func (self *echoBackend) Handle(connection net.Conn) {
	pipeHelper := utils.NewPipeHelper(connection, connection)

	listElement := self.activeConnections.PushBack(pipeHelper)

	pipeHelper.OnClose(func() {
		self.activeConnections.Remove(listElement)
	})
}
