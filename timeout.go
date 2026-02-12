package main

import (
	"context"
	"fmt"
	"net"
	"time"
)

const (
	TimeoutWebRTCICE    = 30 * time.Second
	TimeoutTCPConnect   = 10 * time.Second
	TimeoutTCPOperation = 5 * time.Second
	TimeoutFileIO       = 5 * time.Second
	TimeoutNetwork      = 10 * time.Second
)

func RunWithTimeout[T any](operation string, timeout time.Duration, fn func() (T, error)) (T, error) {
	var zero T
	result := make(chan T, 1)
	errChan := make(chan error, 1)

	go func() {
		res, err := fn()
		if err != nil {
			errChan <- err
			return
		}
		result <- res
	}()

	select {
	case res := <-result:
		return res, nil
	case err := <-errChan:
		return zero, err
	case <-time.After(timeout):
		return zero, fmt.Errorf("%s timeout after %v", operation, timeout)
	}
}

func DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	return RunWithTimeout(
		fmt.Sprintf("dial %s://%s", network, address),
		timeout,
		func() (net.Conn, error) {
			return net.DialTimeout(network, address, timeout)
		},
	)
}

func ListenTimeout(network, address string, timeout time.Duration) (net.Listener, error) {
	return RunWithTimeout(
		fmt.Sprintf("listen %s://%s", network, address),
		timeout,
		func() (net.Listener, error) {
			return net.Listen(network, address)
		},
	)
}

func WithTimeoutContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, timeout)
}
