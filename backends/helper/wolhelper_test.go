package helper

import (
	"bytes"
	"errors"
	"net"
	"strings"
	"testing"
	"time"
)

func TestGenerateMagicPacket(t *testing.T) {
	mac, _ := net.ParseMAC("01:23:45:67:89:ab")
	packet := generateMagicPacket(mac)

	if len(packet) != 102 {
		t.Errorf("expected magic packet length 102, got %d", len(packet))
	}

	expectedMagicPacket := []byte{
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0x01, 0x23, 0x45, 0x67, 0x89, 0xAB,
		0x01, 0x23, 0x45, 0x67, 0x89, 0xAB,
		0x01, 0x23, 0x45, 0x67, 0x89, 0xAB,
		0x01, 0x23, 0x45, 0x67, 0x89, 0xAB,
		0x01, 0x23, 0x45, 0x67, 0x89, 0xAB,
		0x01, 0x23, 0x45, 0x67, 0x89, 0xAB,
		0x01, 0x23, 0x45, 0x67, 0x89, 0xAB,
		0x01, 0x23, 0x45, 0x67, 0x89, 0xAB,
		0x01, 0x23, 0x45, 0x67, 0x89, 0xAB,
		0x01, 0x23, 0x45, 0x67, 0x89, 0xAB,
		0x01, 0x23, 0x45, 0x67, 0x89, 0xAB,
		0x01, 0x23, 0x45, 0x67, 0x89, 0xAB,
		0x01, 0x23, 0x45, 0x67, 0x89, 0xAB,
		0x01, 0x23, 0x45, 0x67, 0x89, 0xAB,
		0x01, 0x23, 0x45, 0x67, 0x89, 0xAB,
		0x01, 0x23, 0x45, 0x67, 0x89, 0xAB,
	}

	if !bytes.Equal(packet, expectedMagicPacket) {
		t.Errorf("magic packet does not match expected value")
	}
}

func TestNewWoLHelper_InvalidMAC(t *testing.T) {
	_, err := NewWoLHelper("invalid-mac", "127.0.0.1:9")
	if err == nil {
		t.Error("expected error for invalid MAC, got nil")
	}
}

func TestNewWoLHelper_InvalidBroadcast(t *testing.T) {
	_, err := NewWoLHelper("01:23:45:67:89:ab", "invalid-addr")
	if err == nil {
		t.Error("expected error for invalid broadcast address, got nil")
	}
}

func TestNewWoLHelper_GenerateCorrectMagicPacket(t *testing.T) {
	wolHelper, err := NewWoLHelper("01:23:45:67:89:ab", "127.0.0.2:9")
	if err != nil {
		t.Fatal("expected wolHelper to get created successfully")
	}

	mac, _ := net.ParseMAC("01:23:45:67:89:ab")
	magicPacket := generateMagicPacket(mac)

	if !bytes.Equal(magicPacket, wolHelper.magicPacket) {
		t.Errorf("wolHelper stored magic packet does not match freshly generated one")
	}
}

func TestWoLHelper_SendWoLPacket_Success(t *testing.T) {
	wolHelper, err := NewWoLHelper("01:23:45:67:89:ab", "127.0.0.3:9")
	if err != nil {
		t.Fatalf("NewWoLHelper() failed: %v", err)
	}

	receivedUDPListenerData := make(chan []byte, 1)
	mockDialer := &mockUDPDialer{
		mockDialUDP: func(network string, laddr, raddr *net.UDPAddr) (*net.UDPConn, error) {
			// Verify parameters
			if network != "udp" {
				t.Errorf("DialUDP network = %q, want %q", network, "udp")
			}
			if laddr != nil {
				t.Errorf("DialUDP laddr = %v, want nil", laddr)
			}
			if raddr.String() != "127.0.0.3:9" {
				t.Errorf("DialUDP raddr = %q, want %q", raddr.String(), "127.0.0.3:9")
			}

			// Because mocking UDP is hard, we create a real UDP listener and use that for testing
			localAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:0")
			//nolint:govet // shadowing "err" is fine
			udpListener, err := net.ListenUDP("udp", localAddr)
			if err != nil {
				return nil, err
			}

			// Read the first packet we get and close the listener
			go func() {
				// We defer here so if anything goes wrong while reading, the listener gets closed anyway
				defer udpListener.Close()
				readBuffer := make([]byte, 1024)
				n, _ := udpListener.Read(readBuffer)
				receivedUDPListenerData <- readBuffer[:n]
			}()

			// Connect to the udpListener and create the actual connection
			mockConn, err := net.DialUDP("udp", nil, udpListener.LocalAddr().(*net.UDPAddr))
			if err != nil {
				udpListener.Close()
				return nil, err
			}

			return mockConn, nil
		},
	}

	wolHelper.dialer = mockDialer

	err = wolHelper.SendWoLPacket()
	if err != nil {
		t.Fatalf("SendWoLPacket() failed: %v", err)
	}

	select {
	case receivedData := <-receivedUDPListenerData:
		if !bytes.Equal(receivedData, wolHelper.magicPacket) {
			t.Errorf("sent packet does not match magic packet")
		}
	case <-time.After(100 * time.Millisecond):
		t.Errorf("did not receive wol-magic-packet in reasonable amount of time")
	}
}

func TestWoLHelper_SendWoLPacket_DialError(t *testing.T) {
	wolHelper, err := NewWoLHelper("01:23:45:67:89:ab", "192.168.1.255:9")
	if err != nil {
		t.Fatalf("NewWoLHelper() failed: %v", err)
	}

	mockDialer := &mockUDPDialer{
		mockDialUDP: func(_ string, _, _ *net.UDPAddr) (*net.UDPConn, error) {
			return nil, errors.New("network unreachable")
		},
	}

	wolHelper.dialer = mockDialer

	err = wolHelper.SendWoLPacket()
	if err == nil {
		t.Fatal("expected SendWoLPacket() to fail when dial fails")
	}
	if !strings.Contains(err.Error(), "failed to connect to broadcast addr") {
		t.Errorf("error message = %q, want to contain %q", err.Error(), "failed to connect to broadcast addr")
	}
}

func TestWoLHelper_SendWoLPacket_WriteError(t *testing.T) {
	wolHelper, err := NewWoLHelper("01:23:45:67:89:ab", "192.168.1.255:9")
	if err != nil {
		t.Fatalf("NewWoLHelper() failed: %v", err)
	}

	// Create a closed connection to simulate write error
	mockDialer := &mockUDPDialer{
		mockDialUDP: func(_ string, _, raddr *net.UDPAddr) (*net.UDPConn, error) {
			// Create and immediately close a real connection to trigger write error
			// (udp is stateless, so creating the connection works fine)
			//nolint:govet // shadowing "err" is fine
			udpConn, err := net.DialUDP("udp", nil, raddr)
			if err != nil {
				return nil, err
			}
			udpConn.Close()
			return udpConn, nil
		},
	}

	wolHelper.dialer = mockDialer

	err = wolHelper.SendWoLPacket()
	if err == nil {
		t.Fatal("expected SendWoLPacket() to fail when write fails")
	}
	if !strings.Contains(err.Error(), "failed to send WOL paket") {
		t.Errorf("error message = %q, want to contain %q", err.Error(), "failed to send WOL paket")
	}
}
