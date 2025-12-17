package backends

import (
	"container/list"
	"net"
	"sync"

	"github.com/sateffen/pluggo/backends/helper"
	"github.com/sateffen/pluggo/config"
)

type echoBackend struct {
	name              string
	activeConnections *list.List
	connectionsMutex  sync.Mutex
}

// newEchoBackend creates a new instance of echoBackend, preparing it with all necessary dependencies.
func newEchoBackend(conf config.EchoBackendConfig) *echoBackend {
	return &echoBackend{
		name:              conf.Name,
		activeConnections: list.New(),
	}
}

// GetName returns the name of the current echoBackend instance.
func (be *echoBackend) GetName() string {
	return be.name
}

// Handle handles given connection by writing all bytes read from it back to the connection itself.
// Handle takes ownership of given connection.
func (be *echoBackend) Handle(connection net.Conn) {
	pipeHelper := helper.NewPipeHelper(connection, connection)

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
