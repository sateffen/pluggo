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

func (backend *echoBackend) GetName() string {
	return backend.name
}

func (backend *echoBackend) Handle(connection net.Conn) {
	pipeHelper := utils.NewPipeHelper(connection, connection)

	listElement := backend.activeConnections.PushBack(pipeHelper)

	pipeHelper.OnClose(func() {
		backend.activeConnections.Remove(listElement)
	})
}
