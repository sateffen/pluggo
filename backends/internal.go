package backends

import (
	"net"
	"time"
)

// this file provides interfaces and their default implementations to make the other structs testable.

type dialer interface {
	DialTimeout(network, address string, timeout time.Duration) (net.Conn, error)
}

type defaultDialer struct{}

func (defaultDialer) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	d := net.Dialer{Timeout: timeout}
	return d.Dial(network, address)
}

type sleeper interface {
	Sleep(d time.Duration)
}

type defaultSleeper struct{}

func (defaultSleeper) Sleep(d time.Duration) {
	time.Sleep(d)
}
