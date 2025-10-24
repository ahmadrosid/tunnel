package proxy

import (
	"fmt"
	"io"

	"github.com/ahmadrosid/tunnel/internal/tunnel"
)

// DialThroughTunnel creates a connection through a WebSocket tunnel
func DialThroughTunnel(tun *tunnel.Tunnel) (tunnel.Connection, error) {
	// Check if WebSocket connection is still alive
	if tun.WSConn == nil {
		return nil, fmt.Errorf("WebSocket connection is nil")
	}

	// The WebSocket connection is already established and ready to use
	// No need to open a new channel like in SSH
	return tun.WSConn, nil
}

// CopyBidirectional copies data bidirectionally between two connections
func CopyBidirectional(conn1, conn2 io.ReadWriteCloser) error {
	errChan := make(chan error, 2)

	// Copy from conn1 to conn2
	go func() {
		_, err := io.Copy(conn2, conn1)
		errChan <- err
	}()

	// Copy from conn2 to conn1
	go func() {
		_, err := io.Copy(conn1, conn2)
		errChan <- err
	}()

	// Wait for either direction to complete
	err := <-errChan

	// Close both connections to stop the other goroutine
	conn1.Close()
	conn2.Close()

	// Wait for second goroutine
	if err2 := <-errChan; err2 != nil && err == nil {
		err = err2
	}

	return err
}
