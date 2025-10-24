package proxy

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/ahmadrosid/tunnel/internal/cert"
	"github.com/ahmadrosid/tunnel/internal/config"
	"github.com/ahmadrosid/tunnel/internal/tunnel"
)

// Server represents the HTTP/HTTPS proxy server
type Server struct {
	config      *config.Config
	registry    *tunnel.Registry
	certManager *cert.Manager
	httpServer  *http.Server
	httpsServer *http.Server
}

// NewServer creates a new proxy server
func NewServer(cfg *config.Config, registry *tunnel.Registry) *Server {
	s := &Server{
		config:      cfg,
		registry:    registry,
		certManager: cert.NewManager(cfg),
	}

	// Create HTTP server
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      s.certManager.HTTPHandler()(http.HandlerFunc(s.handleHTTP)),
		ReadTimeout:  cfg.RequestTimeout,
		WriteTimeout: cfg.RequestTimeout,
	}

	// Create HTTPS server if enabled
	if cfg.EnableHTTPS {
		s.httpsServer = &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.HTTPSPort),
			Handler:      http.HandlerFunc(s.handleHTTP),
			TLSConfig:    s.certManager.GetTLSConfig(),
			ReadTimeout:  cfg.RequestTimeout,
			WriteTimeout: cfg.RequestTimeout,
		}
	}

	return s
}

// Start starts the HTTP and HTTPS proxy servers
func (s *Server) Start() error {
	// Start HTTP server
	go func() {
		log.Printf("HTTP proxy listening on port %d", s.config.HTTPPort)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Start HTTPS server if enabled
	if s.config.EnableHTTPS && s.httpsServer != nil {
		go func() {
			log.Printf("HTTPS proxy listening on port %d", s.config.HTTPSPort)
			if err := s.httpsServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				log.Fatalf("HTTPS server error: %v", err)
			}
		}()
	}

	// Block forever
	select {}
}

// Shutdown gracefully shuts down the proxy servers
func (s *Server) Shutdown(ctx context.Context) error {
	var err error
	if s.httpServer != nil {
		if shutdownErr := s.httpServer.Shutdown(ctx); shutdownErr != nil {
			err = shutdownErr
		}
	}
	if s.httpsServer != nil {
		if shutdownErr := s.httpsServer.Shutdown(ctx); shutdownErr != nil {
			err = shutdownErr
		}
	}
	return err
}

// handleHTTP handles incoming HTTP/HTTPS requests
func (s *Server) handleHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract subdomain from Host header
	host := r.Host
	subdomain := s.extractSubdomain(host)

	if subdomain == "" {
		s.writeError(w, http.StatusNotFound, "Invalid hostname")
		return
	}

	// Look up tunnel by subdomain
	tun, exists := s.registry.Get(subdomain)
	if !exists {
		log.Printf("Subdomain not found: %s", subdomain)
		s.writeError(w, http.StatusNotFound, fmt.Sprintf("Tunnel not found for subdomain: %s", subdomain))
		return
	}

	// Hijack the connection for raw TCP forwarding
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		log.Printf("Response writer does not support hijacking")
		s.writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		log.Printf("Failed to hijack connection: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Forward the request to the tunnel
	go func() {
		defer clientConn.Close()

		// Dial through the SSH tunnel to the local server
		tunnelConn, err := DialThroughTunnel(tun)
		if err != nil {
			log.Printf("Failed to dial through tunnel for %s: %v", subdomain, err)
			// Write 502 Bad Gateway error
			response := "HTTP/1.1 502 Bad Gateway\r\nContent-Type: text/plain\r\nContent-Length: 15\r\n\r\nBad Gateway\r\n"
			clientConn.Write([]byte(response))
			return
		}
		defer tunnelConn.Close()

		// Write the original HTTP request to the tunnel
		if err := r.Write(tunnelConn); err != nil {
			log.Printf("Failed to write request to tunnel: %v", err)
			return
		}

		// Set timeout on client connection only
		// SSH channels don't support SetDeadline
		if s.config.RequestTimeout > 0 {
			clientConn.SetDeadline(time.Now().Add(s.config.RequestTimeout))
		}

		// Bidirectional copy
		errChan := make(chan error, 2)

		// Copy from tunnel to client
		go func() {
			_, err := io.Copy(clientConn, tunnelConn)
			errChan <- err
		}()

		// Copy from client to tunnel
		go func() {
			_, err := io.Copy(tunnelConn, clientConn)
			errChan <- err
		}()

		// Wait for completion
		<-errChan
		<-errChan
	}()
}

// extractSubdomain extracts the subdomain from a host header
func (s *Server) extractSubdomain(host string) string {
	// Remove port if present
	if colonIndex := strings.Index(host, ":"); colonIndex != -1 {
		host = host[:colonIndex]
	}

	// Check if host ends with our domain
	domain := "." + s.config.Domain
	if !strings.HasSuffix(host, domain) && host != s.config.Domain {
		// Try without the dot (exact match)
		if host == s.config.Domain {
			return ""
		}
		// Not our domain
		return ""
	}

	// Extract subdomain
	subdomain := strings.TrimSuffix(host, domain)
	subdomain = strings.TrimSpace(subdomain)

	return subdomain
}

// writeError writes an HTTP error response
func (s *Server) writeError(w http.ResponseWriter, statusCode int, message string) {
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "%d %s\n%s\n", statusCode, http.StatusText(statusCode), message)
}
