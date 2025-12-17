package backends

import (
	"container/list"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/sateffen/pluggo/backends/helper"
	"github.com/sateffen/pluggo/config"
)

const wolDialTimeout = 2 * time.Second
const wolTimeAfterMagicPacket = 5 * time.Second
const wolMaxDialRetryCount = 50
const wolTimeBetweenRetries = 500 * time.Millisecond

type wolForwarderBackend struct {
	name              string
	activeConnections *list.List
	connectionsMutex  sync.Mutex
	wolSender         helper.WoLSender
	targetAddr        string
	dialer            dialer
	sleeper           sleeper
}

// newWoLForwarderBackend creates a new instance of wolForwarderBackend, preparing it with all necessary dependencies.
func newWoLForwarderBackend(conf config.WoLForwarderBackendConfig) (*wolForwarderBackend, error) {
	wolHelper, err := helper.NewWoLHelper(conf.WoLMACAddr, conf.WoLBroadcastAddr)
	if err != nil {
		return nil, fmt.Errorf("could not create WoL helper: %w", err)
	}

	return &wolForwarderBackend{
		name:              conf.Name,
		activeConnections: list.New(),
		wolSender:         wolHelper,
		targetAddr:        conf.TargetAddr,
		dialer:            defaultDialer{},
		sleeper:           defaultSleeper{},
	}, nil
}

// GetName returns the name of the current wolForwarderBackend instance.
func (be *wolForwarderBackend) GetName() string {
	return be.name
}

// Handle handles given connection by trying to dial the target host. If the target host is reachable,
// a pipe will get generated, else the connection gets closed.
// Handle takes ownership of given connection.
func (be *wolForwarderBackend) Handle(connection net.Conn) {
	connectionToTarget, err := be.tryDial()
	if err != nil {
		slog.Info(
			"backend could not connect to target",
			slog.String("targetAddr", be.targetAddr),
			slog.String("name", be.name),
			slog.Any("error", err),
		)

		if err = connection.Close(); err != nil {
			slog.Warn("could not properly close incoming connection after dialer timeout", slog.Any("error", err))
		}

		return
	}

	pipeHelper := helper.NewPipeHelper(connection, connectionToTarget)

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

// tryDial tries to dial the target host. If successful, the generated connection gets returned. Otherwise
// a wake-on-lan magic-packet is sent and we try to connect to the target host for some time. If a connection
// is establised, we return it, else we return an error.
func (be *wolForwarderBackend) tryDial() (net.Conn, error) {
	// First, try a quick connection to see if target is already awake
	targetConnection, err := be.dialer.DialTimeout("tcp", be.targetAddr, wolDialTimeout)
	if err == nil {
		return targetConnection, nil
	}

	// Target is unreachable - send WoL packet and retry
	slog.Debug("failed to connect to host, sending wol to wake it up", slog.String("targetAddr", be.targetAddr))
	err = be.wolSender.SendWoLPacket()
	if err != nil {
		return nil, fmt.Errorf("could not send wol magic paket: %w", err)
	}

	// Let's give the target system some time to come up, before we try to dial
	be.sleeper.Sleep(wolTimeAfterMagicPacket)

	// Then we retry for something ~2min, else we give up.
	for i := range wolMaxDialRetryCount {
		slog.Debug("trying to connect to host", slog.String("targetAddr", be.targetAddr), slog.Int("retryCount", i))

		be.sleeper.Sleep(wolTimeBetweenRetries)
		targetConnection, err = be.dialer.DialTimeout("tcp", be.targetAddr, wolDialTimeout)
		if err == nil {
			return targetConnection, nil
		}
	}

	return nil, fmt.Errorf("timeout while waiting for target with addr '%s'", be.targetAddr)
}
