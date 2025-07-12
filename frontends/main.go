package frontends

import (
	"fmt"

	"github.com/sateffen/pluggo/config"
)

type Frontend interface {
	GetName() string
	Close()
}

var frontendList = make(map[string]Frontend)

func InitFrontends(conf config.FrontendConfigs) error {
	for _, tcpConf := range conf.Tcp {
		newTCPFrontend, err := NewTCPFrontend(tcpConf)
		if err != nil {
			return fmt.Errorf("could not create frontend \"%s\": %q", tcpConf.Name, err)
		}

		frontendList[tcpConf.Name] = newTCPFrontend
	}

	return nil
}

func GetFrontend(name string) (Frontend, bool) {
	backend, ok := frontendList[name]

	return backend, ok
}

func CloseFrontends() {
	for _, frontend := range frontendList {
		frontend.Close()
	}

	frontendList = make(map[string]Frontend)
}
