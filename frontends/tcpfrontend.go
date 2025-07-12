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

	self := &tcpFrontend{
		name:          conf.Name,
		isClosed:      false,
		listenAddr:    parsedListenAddr,
		listener:      listener,
		targetBackend: targetBackend,
	}

	go func() {
		for {
			connection, err := listener.Accept()

			if err == nil {
				go targetBackend.Handle(connection)
			} else if !self.isClosed {
				slog.Error("could not accept connection", slog.Any("error", err))
			}
		}
	}()

	slog.Info("frontend started listening", slog.String("name", conf.Name), slog.String("listenAddr", conf.ListenAddr))

	return self, nil
}

func (self *tcpFrontend) GetName() string {
	return self.name
}

func (self *tcpFrontend) Close() {
	if self.isClosed {
		return
	}

	self.isClosed = true
	self.listener.Close()
}
