package tunnel

import (
	"fmt"
	"sync"
	"time"
)

// Connection represents a generic connection interface
type Connection interface {
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	Close() error
}

type Tunnel struct {
	ID         string
	Subdomain  string
	WSConn     Connection // WebSocket connection
	LocalAddr  string     // e.g., "localhost:3000"
	RemotePort int        // e.g., 80 or 443
	CreatedAt  time.Time
}

type Registry struct {
	mu      sync.RWMutex
	tunnels map[string]*Tunnel // subdomain -> tunnel
}

func NewRegistry() *Registry {
	return &Registry{
		tunnels: make(map[string]*Tunnel),
	}
}

func (r *Registry) Register(tunnel *Tunnel) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tunnels[tunnel.Subdomain]; exists {
		return fmt.Errorf("subdomain '%s' is already in use", tunnel.Subdomain)
	}

	r.tunnels[tunnel.Subdomain] = tunnel
	return nil
}

func (r *Registry) Unregister(subdomain string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.tunnels, subdomain)
}

func (r *Registry) Get(subdomain string) (*Tunnel, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tunnel, exists := r.tunnels[subdomain]
	return tunnel, exists
}

func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.tunnels)
}

func (r *Registry) IsSubdomainAvailable(subdomain string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.tunnels[subdomain]
	return !exists
}
