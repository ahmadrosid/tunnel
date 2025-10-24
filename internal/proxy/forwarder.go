package proxy

import (
	"fmt"
	"io"
	"log"

	"github.com/ahmadrosid/tunnel/internal/tunnel"
	"golang.org/x/crypto/ssh"
)

// DialThroughTunnel creates a connection through an SSH tunnel
func DialThroughTunnel(tun *tunnel.Tunnel) (ssh.Channel, error) {
	// Check if SSH connection is still alive
	if tun.SSHConn == nil {
		return nil, fmt.Errorf("SSH connection is nil")
	}

	// Create forwarded-tcpip request for reverse port forwarding
	// This tells the SSH client "here's a connection for the port you asked me to forward"
	// The client will then forward it according to its -R configuration
	type forwardedTCPIPMsg struct {
		ConnectedAddr string
		ConnectedPort uint32
		OriginAddr    string
		OriginPort    uint32
	}

	// Use empty string for ConnectedAddr and the RemotePort
	// The SSH client knows where to forward based on its -R configuration
	payload := ssh.Marshal(forwardedTCPIPMsg{
		ConnectedAddr: "",
		ConnectedPort: uint32(tun.RemotePort),
		OriginAddr:    "proxy",
		OriginPort:    uint32(tun.RemotePort),
	})

	// Open a forwarded-tcpip channel (for reverse port forwarding)
	channel, reqs, err := tun.SSHConn.OpenChannel("forwarded-tcpip", payload)
	if err != nil {
		log.Printf("Failed to open channel through tunnel %s: %v", tun.Subdomain, err)
		return nil, fmt.Errorf("failed to connect to local server: %w", err)
	}

	// Discard all requests
	go ssh.DiscardRequests(reqs)

	return channel, nil
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
