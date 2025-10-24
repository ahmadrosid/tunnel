package proxy

import (
	"io"
	"sync"

	"github.com/ahmadrosid/tunnel/internal/tunnel"
)

// VirtualConnection wraps a tunnel connection for a single HTTP request
// It prevents closing the underlying WebSocket connection when the HTTP request completes.
// This allows multiple HTTP requests to be handled over the same persistent WebSocket.
type VirtualConnection struct {
	underlying tunnel.Connection
	closed     bool
	mu         sync.Mutex
}

// NewVirtualConnection creates a new virtual connection wrapper
func NewVirtualConnection(conn tunnel.Connection) *VirtualConnection {
	return &VirtualConnection{
		underlying: conn,
		closed:     false,
	}
}

// Read implements io.Reader
func (v *VirtualConnection) Read(p []byte) (n int, err error) {
	v.mu.Lock()
	if v.closed {
		v.mu.Unlock()
		return 0, io.EOF
	}
	v.mu.Unlock()

	return v.underlying.Read(p)
}

// Write implements io.Writer
func (v *VirtualConnection) Write(p []byte) (n int, err error) {
	v.mu.Lock()
	if v.closed {
		v.mu.Unlock()
		return 0, io.ErrClosedPipe
	}
	v.mu.Unlock()

	return v.underlying.Write(p)
}

// Close marks this virtual connection as closed, but does NOT close the underlying WebSocket
// This allows the WebSocket to stay alive for future HTTP requests
func (v *VirtualConnection) Close() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.closed {
		return nil
	}

	v.closed = true
	// Intentionally do NOT close v.underlying
	// The WebSocket connection must stay alive for future requests
	return nil
}
