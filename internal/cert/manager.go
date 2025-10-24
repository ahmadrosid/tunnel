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
	// Create registry reference for validation (will be set later)
	manager := &Manager{
		config: cfg,
	}

	m := &autocert.Manager{
		Prompt: autocert.AcceptTOS,
		Cache:  autocert.DirCache(cfg.CertCacheDir),
		HostPolicy: func(ctx context.Context, host string) error {
			// Reject localhost, IPs, and invalid hostnames
			if host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "" {
				return fmt.Errorf("certificates not supported for %s", host)
			}

			// Allow the base domain
			if host == cfg.Domain {
				log.Printf("Certificate requested for base domain: %s", host)
				return nil
			}

			// For subdomains, log the request
			// Note: We allow all subdomains because we can't check tunnel registry here
			// The proxy layer will return 404 if tunnel doesn't exist
			log.Printf("Certificate requested for: %s", host)
			return nil
		},
	}

	// Set email if provided
	if cfg.LetsEncryptEmail != "" {
		m.Email = cfg.LetsEncryptEmail
	}

	manager.autocertManager = m
	return manager
}

// GetTLSConfig returns a TLS configuration for HTTPS server
func (m *Manager) GetTLSConfig() *tls.Config {
	return m.autocertManager.TLSConfig()
}

// GetTLSConfigForHijacking returns a TLS configuration with HTTP/2 disabled
// This is required for connection hijacking to work properly.
// HTTP/2 doesn't support hijacking, so we force HTTP/1.1.
func (m *Manager) GetTLSConfigForHijacking() *tls.Config {
	// Clone the config to avoid mutating the shared instance
	cfg := m.autocertManager.TLSConfig().Clone()
	// Disable HTTP/2 by only allowing HTTP/1.1
	cfg.NextProtos = []string{"http/1.1"}
	return cfg
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
