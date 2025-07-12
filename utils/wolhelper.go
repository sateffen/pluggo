package utils

import (
	"bytes"
	"fmt"
	"log/slog"
	"net"
)

func generateMagicPacket(macAddr net.HardwareAddr) []byte {
	var magicPacket bytes.Buffer

	for i := 0; i < 6; i++ {
		magicPacket.WriteByte(0xFF)
	}

	for i := 0; i < 16; i++ {
		magicPacket.Write(macAddr)
	}

	return magicPacket.Bytes()
}

type WoLHelper struct {
	wolMACAddr       net.HardwareAddr
	wolBroadcastAddr *net.UDPAddr
	magicPacket      []byte
}

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
	}, nil
}

func (self *WoLHelper) SendWOLPaket() error {
	conn, err := net.DialUDP("udp", nil, self.wolBroadcastAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to broadcast addr: %w", err)
	}
	defer conn.Close()

	_, err = conn.Write(self.magicPacket)
	if err != nil {
		return fmt.Errorf("failed to send WOL paket to broadcast addr: %w", err)
	}

	slog.Debug("Sent WoL magic packet", slog.String("wolMACAddr", self.wolMACAddr.String()), slog.String("wolBroadcastAddr", self.wolBroadcastAddr.String()))

	return nil
}
