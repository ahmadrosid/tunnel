package tunnel

import (
	"fmt"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// Tunnel represents an active SSH tunnel
type Tunnel struct {
	ID         string
	Subdomain  string
	SSHConn    ssh.Conn
	LocalAddr  string // e.g., "localhost:3000"
	RemotePort int    // e.g., 80 or 443
	CreatedAt  time.Time
}

// Registry manages active tunnels
type Registry struct {
	mu      sync.RWMutex
	tunnels map[string]*Tunnel // subdomain -> tunnel
}

// NewRegistry creates a new tunnel registry
func NewRegistry() *Registry {
	return &Registry{
		tunnels: make(map[string]*Tunnel),
	}
}

// Register adds a new tunnel to the registry
func (r *Registry) Register(tunnel *Tunnel) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if subdomain is already taken
	if _, exists := r.tunnels[tunnel.Subdomain]; exists {
		return fmt.Errorf("subdomain '%s' is already in use", tunnel.Subdomain)
	}

	r.tunnels[tunnel.Subdomain] = tunnel
	return nil
}

// Unregister removes a tunnel from the registry
func (r *Registry) Unregister(subdomain string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.tunnels, subdomain)
}

// Get retrieves a tunnel by subdomain
func (r *Registry) Get(subdomain string) (*Tunnel, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tunnel, exists := r.tunnels[subdomain]
	return tunnel, exists
}

// Count returns the number of active tunnels
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.tunnels)
}

// IsSubdomainAvailable checks if a subdomain is available
func (r *Registry) IsSubdomainAvailable(subdomain string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.tunnels[subdomain]
	return !exists
}
