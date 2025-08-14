package backends

import (
	"container/list"
	"net"
	"sync"

	"github.com/sateffen/pluggo/config"
	"github.com/sateffen/pluggo/utils"
)

type echoBackend struct {
	name              string
	activeConnections *list.List
	connectionsMutex  sync.Mutex
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

	backend.connectionsMutex.Lock()
	listElement := backend.activeConnections.PushBack(pipeHelper)
	backend.connectionsMutex.Unlock()

	pipeHelper.OnClose(func() {
		backend.connectionsMutex.Lock()
		backend.activeConnections.Remove(listElement)
		backend.connectionsMutex.Unlock()
	})
}
