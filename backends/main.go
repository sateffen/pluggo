package backends

import (
	"fmt"
	"log/slog"
	"net"

	"github.com/sateffen/pluggo/config"
)

type Backend interface {
	GetName() string
	Handle(connection net.Conn)
	Close() error
}

type BackendList struct {
	list map[string]Backend
}

// NewBackendList creates a new BackendList, filling it with backend instances based on provided BackendConfigs.
func NewBackendList(conf config.BackendConfigs) (*BackendList, error) {
	bl := BackendList{
		list: make(map[string]Backend),
	}

	for _, echoConf := range conf.Echo {
		bl.list[echoConf.Name] = newEchoBackend(echoConf)
	}

	for _, tcpForwarderConf := range conf.TCPForwarder {
		bl.list[tcpForwarderConf.Name] = newTCPForwarderBackend(tcpForwarderConf)
	}

	for _, wolForwarderConf := range conf.WoLForwarder {
		wolForwarderBackend, err := newWoLForwarderBackend(wolForwarderConf)
		if err != nil {
			return nil, fmt.Errorf("could not create backend '%s': %w", wolForwarderConf.Name, err)
		}

		bl.list[wolForwarderConf.Name] = wolForwarderBackend
	}

	return &bl, nil
}

// Get returns the backend with given name if present. The second return value indicates whether
// the value is present, like in a casual map.
func (bl *BackendList) Get(name string) (Backend, bool) {
	backend, ok := bl.list[name]

	return backend, ok
}

// CloseAll closes all backends and therefore all active connections.
func (bl *BackendList) CloseAll() {
	for _, backend := range bl.list {
		if err := backend.Close(); err != nil {
			slog.Warn("couldn't close frontend properly", slog.String("name", backend.GetName()), slog.Any("error", err))
		}
	}
}
