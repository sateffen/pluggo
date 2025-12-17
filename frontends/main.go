package frontends

import (
	"fmt"
	"log/slog"

	"github.com/sateffen/pluggo/backends"
	"github.com/sateffen/pluggo/config"
)

type Frontend interface {
	GetName() string
	Listen() error
	Close() error
}

type FrontendList struct {
	list map[string]Frontend
}

// NewFrontendList creates a new instance of FrontendList, filling it frontend instances based on given configs.
func NewFrontendList(conf config.FrontendConfigs, backendList *backends.BackendList) (*FrontendList, error) {
	fl := FrontendList{
		list: make(map[string]Frontend),
	}

	for _, tcpConf := range conf.TCP {
		tcpFrontend, err := newTCPFrontend(tcpConf, backendList)
		if err != nil {
			fl.CloseAll()
			return nil, fmt.Errorf("could not create frontend \"%s\": %w", tcpConf.Name, err)
		}

		fl.list[tcpConf.Name] = tcpFrontend
	}

	return &fl, nil
}

// Get returns the backend with given name if present. The second return value indicates whether
// the value is present, like in a casual map.
func (fl *FrontendList) Get(name string) (Frontend, bool) {
	frontend, ok := fl.list[name]

	return frontend, ok
}

// ListenAll starts the listener for all Frontends. Each listen starts in its own go-routine.
// If any listen fails, this will close all frontends.
func (fl *FrontendList) ListenAll() {
	for _, frontend := range fl.list {
		go func(fe Frontend) {
			err := fe.Listen()

			if err != nil {
				slog.Warn("tcpfrontend failed to listen", slog.String("name", fe.GetName()), slog.Any("error", err))
				fl.CloseAll()
			}
		}(frontend)
	}
}

// CloseAll closes all listening frontends and therefore stops all listening frontends.
func (fl *FrontendList) CloseAll() {
	for _, frontend := range fl.list {
		if err := frontend.Close(); err != nil {
			slog.Warn("couldn't close frontend properly", slog.String("name", frontend.GetName()), slog.Any("error", err))
		}
	}
}
