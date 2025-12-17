package helper

import (
	"bytes"
	"fmt"
	"log/slog"
	"net"
)

// generateMagicPacket generated a magic wake-on-lan packet for given mac-address and returns it as buffer.
func generateMagicPacket(macAddr net.HardwareAddr) []byte {
	var magicPacket bytes.Buffer

	for range 6 {
		//nolint:mnd // 0xFF is an obvious value for this usecase
		magicPacket.WriteByte(0xFF)
	}

	for range 16 {
		magicPacket.Write(macAddr)
	}

	return magicPacket.Bytes()
}

type WoLSender interface {
	SendWoLPacket() error
}

type WoLHelper struct {
	wolMACAddr       net.HardwareAddr
	wolBroadcastAddr *net.UDPAddr
	magicPacket      []byte
	dialer           udpDialer
}

// NewWoLHelper creates a new instance of WoLHelper, preparing it with all necessary dependencies. Additionally
// generates all necessary structures for simple sending of a wake-on-lan packet later on, caching the values.
func NewWoLHelper(wolMACAddr string, wolBroadcastAddr string) (*WoLHelper, error) {
	macAddr, err := net.ParseMAC(wolMACAddr)
	if err != nil {
		return nil, err
	}

	broadcastAddr, err := net.ResolveUDPAddr("udp", wolBroadcastAddr)
	if err != nil {
		return nil, err
	}

	return &WoLHelper{
		wolMACAddr:       macAddr,
		wolBroadcastAddr: broadcastAddr,
		magicPacket:      generateMagicPacket(macAddr),
		dialer:           defaultUDPDialer{},
	}, nil
}

// SendWoLPacket sends a wake-on-lan magic-packet to preconfigured address, or returns the
// error if anything goes wrong.
func (wh *WoLHelper) SendWoLPacket() error {
	conn, err := wh.dialer.DialUDP("udp", nil, wh.wolBroadcastAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to broadcast addr: %w", err)
	}
	defer conn.Close()

	_, err = conn.Write(wh.magicPacket)
	if err != nil {
		return fmt.Errorf("failed to send WOL paket to broadcast addr: %w", err)
	}

	slog.Debug(
		"Sent WoL magic packet",
		slog.String("wolMACAddr", wh.wolMACAddr.String()),
		slog.String("wolBroadcastAddr", wh.wolBroadcastAddr.String()),
	)

	return nil
}
