package backends

import (
	"fmt"
	"net"

	"github.com/sateffen/pluggo/config"
)

type Backend interface {
	GetName() string
	Handle(connection net.Conn)
}

var backendList = make(map[string]Backend)

func InitBackends(conf config.BackendConfigs) error {
	for _, echoConf := range conf.Echo {
		newEchoBackend, err := NewEchoBackend(echoConf)
		if err != nil {
			return fmt.Errorf("could not create backend \"%s\": %q", echoConf.Name, err)
		}

		backendList[echoConf.Name] = newEchoBackend
	}

	for _, tcpForwarderConf := range conf.TCPForwarder {
		newTCPForwarderBackend, err := NewTCPForwarderBackend(tcpForwarderConf)
		if err != nil {
			return fmt.Errorf("could not create backend \"%s\": %q", tcpForwarderConf.Name, err)
		}

		backendList[tcpForwarderConf.Name] = newTCPForwarderBackend
	}

	for _, wolForwarderConf := range conf.WoLForwarder {
		newWoLForwarderBackend, err := NewWoLForwarderBackend(wolForwarderConf)
		if err != nil {
			return fmt.Errorf("could not create backend \"%s\": %q", wolForwarderConf.Name, err)
		}

		backendList[wolForwarderConf.Name] = newWoLForwarderBackend
	}

	return nil
}

func GetBackend(name string) (Backend, bool) {
	backend, ok := backendList[name]

	return backend, ok
}
