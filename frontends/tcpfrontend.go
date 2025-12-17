package frontends

import (
	"fmt"
	"log/slog"
	"net"
	"sync"

	"github.com/sateffen/pluggo/backends"
	"github.com/sateffen/pluggo/config"
)

type tcpFrontend struct {
	name            string
	targetBackend   backends.Backend
	listenerMutex   sync.RWMutex
	listenAddr      *net.TCPAddr
	listener        tcpListener
	listenerFactory tcpListenerFactory
}

// newTCPFrontend creates a new instance of an tcpFrontend, preparing it with all default dependencies.
func newTCPFrontend(conf config.TCPFrontendConfig, backendList *backends.BackendList) (*tcpFrontend, error) {
	parsedListenAddr, err := net.ResolveTCPAddr("tcp", conf.ListenAddr)
	if err != nil {
		return nil, fmt.Errorf("could not parse listenAddr '%s' of frontend '%s': %w", conf.ListenAddr, conf.Name, err)
	}

	targetBackend, ok := backendList.Get(conf.Target)
	if !ok {
		return nil, fmt.Errorf("target backend '%s' for frontend '%s' does not exist", conf.Target, conf.Name)
	}

	return &tcpFrontend{
		name:            conf.Name,
		targetBackend:   targetBackend,
		listenerMutex:   sync.RWMutex{},
		listenAddr:      parsedListenAddr,
		listener:        nil,
		listenerFactory: defaultTCPListenerFactory{},
	}, nil
}

// GetName returns the name of the current wolForwarderBackend instance.
func (fe *tcpFrontend) GetName() string {
	return fe.name
}

// Listen creates a TCP listener, starts listening and accepting connections.
// Listen blocks the current thread by starting an endless loop accepting new connections.
// Listen is resilient in that it does not stop accepting connections just because an error happens.
func (fe *tcpFrontend) Listen() error {
	fe.listenerMutex.Lock()
	listener, err := fe.listenerFactory.ListenTCP("tcp", fe.listenAddr)
	fe.listener = listener
	fe.listenerMutex.Unlock()

	if err != nil {
		return fmt.Errorf("can't listen on '%s' for frontend '%s': %w", fe.listenAddr, fe.name, err)
	}

	slog.Info("tcpfrontend started listening", slog.String("name", fe.name), slog.String("listenAddr", fe.listenAddr.String()))

	for {
		//nolint:govet // shadowing "err" is fine
		connection, err := listener.Accept()

		if err != nil {
			// Exit the loop if the listener doesn't exist anymore
			fe.listenerMutex.RLock()
			if fe.listener == nil {
				fe.listenerMutex.RUnlock()
				break
			}
			fe.listenerMutex.RUnlock()

			slog.Error("could not accept connection", slog.Any("error", err))
			continue
		}

		fe.targetBackend.Handle(connection)
	}

	return nil
}

// Close closes the listening instance if existing.
func (fe *tcpFrontend) Close() error {
	fe.listenerMutex.Lock()
	defer func() {
		fe.listener = nil
		fe.listenerMutex.Unlock()
	}()

	if fe.listener == nil {
		return nil
	}

	if err := fe.listener.Close(); err != nil {
		return fmt.Errorf("tcpfrontend could not close listener: %w", err)
	}

	return nil
}
