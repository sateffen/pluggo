package frontends

import (
	"fmt"
	"log/slog"
	"net"

	"github.com/sateffen/pluggo/backends"
	"github.com/sateffen/pluggo/config"
)

type tcpFrontend struct {
	name          string
	isClosed      bool
	listenAddr    *net.TCPAddr
	listener      *net.TCPListener
	targetBackend backends.Backend
}

func NewTCPFrontend(conf config.TCPFrontendConfig) (*tcpFrontend, error) {
	parsedListenAddr, err := net.ResolveTCPAddr("tcp", conf.ListenAddr)
	if err != nil {
		return nil, fmt.Errorf("could not parse listenAddr \"%s\" of frontend \"%s\": %q", conf.ListenAddr, conf.Name, err)
	}

	targetBackend, ok := backends.GetBackend(conf.Target)
	if !ok {
		return nil, fmt.Errorf("target backend \"%s\" for frontend \"%s\" does not exist", conf.Target, conf.Name)
	}

	listener, err := net.ListenTCP("tcp", parsedListenAddr)
	if err != nil {
		return nil, fmt.Errorf("can't listen on \"%s\" for frontend \"%s\": %q", conf.ListenAddr, conf.Name, err)
	}

	frontend := &tcpFrontend{
		name:          conf.Name,
		isClosed:      false,
		listenAddr:    parsedListenAddr,
		listener:      listener,
		targetBackend: targetBackend,
	}

	go func() {
		for {
			connection, err := listener.Accept()

			if err != nil {
				// Exit the loop if the listener is closed
				if frontend.isClosed {
					break
				}

				slog.Error("could not accept connection", slog.Any("error", err))
				continue
			}

			targetBackend.Handle(connection)
		}
	}()

	slog.Info("frontend started listening", slog.String("name", conf.Name), slog.String("listenAddr", conf.ListenAddr))

	return frontend, nil
}

func (frontend *tcpFrontend) GetName() string {
	return frontend.name
}

func (frontend *tcpFrontend) Close() {
	if frontend.isClosed {
		return
	}

	frontend.isClosed = true
	frontend.listener.Close()
}
