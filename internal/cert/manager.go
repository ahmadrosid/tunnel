package cert

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"

	"github.com/ahmadrosid/tunnel/internal/config"
	"golang.org/x/crypto/acme/autocert"
)

// Manager handles TLS certificate management
type Manager struct {
	autocertManager *autocert.Manager
	config          *config.Config
}

// NewManager creates a new certificate manager
func NewManager(cfg *config.Config) *Manager {
	m := &autocert.Manager{
		Prompt: autocert.AcceptTOS,
		Cache:  autocert.DirCache(cfg.CertCacheDir),
		HostPolicy: func(ctx context.Context, host string) error {
			// Reject localhost, IPs, and invalid hostnames
			if host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "" {
				return fmt.Errorf("certificates not supported for %s", host)
			}

			// Allow the base domain and all subdomains
			// This prevents the "server name component count invalid" error
			return nil
		},
	}

	// Set email if provided
	if cfg.LetsEncryptEmail != "" {
		m.Email = cfg.LetsEncryptEmail
	}

	return &Manager{
		autocertManager: m,
		config:          cfg,
	}
}

// GetTLSConfig returns a TLS configuration for HTTPS server
func (m *Manager) GetTLSConfig() *tls.Config {
	return m.autocertManager.TLSConfig()
}

// HTTPHandler returns HTTP handler for ACME HTTP-01 challenge
func (m *Manager) HTTPHandler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return m.autocertManager.HTTPHandler(next)
	}
}

// GetCertificate returns a certificate for the given client hello
func (m *Manager) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	cert, err := m.autocertManager.GetCertificate(hello)
	if err != nil {
		log.Printf("Failed to get certificate for %s: %v", hello.ServerName, err)
		return nil, fmt.Errorf("failed to get certificate: %w", err)
	}
	return cert, nil
}
