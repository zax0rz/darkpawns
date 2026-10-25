//go:build !linux

package main

import (
	"errors"
	"fmt"
	"net"
	"time"
)

func bootLease(time.Duration) (func(), error) {
	return nil, errors.New("listener ownership verification requires Linux /proc")
}

func waitOwnedListener(*process, int, time.Duration) error {
	return errors.New("listener ownership verification requires Linux /proc")
}

func ownedStart(_ time.Duration, _ []net.Listener, _ []int, _ func() (*process, error)) func() (*process, error) {
	return func() (*process, error) {
		return nil, errors.New("listener ownership verification requires Linux /proc")
	}
}

func reserveOraclePort(port int) (net.Listener, error) {
	return net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
}
