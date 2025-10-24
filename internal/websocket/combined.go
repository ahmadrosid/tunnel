package websocket

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/ahmadrosid/tunnel/internal/config"
	"github.com/ahmadrosid/tunnel/internal/proxy"
	"github.com/ahmadrosid/tunnel/internal/tunnel"
)

// CombinedServer handles both WebSocket and HTTPS proxy on the same port
type CombinedServer struct {
	config      *config.Config
	registry    *tunnel.Registry
	certManager interface {
		GetTLSConfig() *tls.Config
	}
	server      *http.Server
	httpServer  *http.Server
	wsHandler   *Server
}

// NewCombinedServer creates a combined server for WebSocket and HTTPS proxy
func NewCombinedServer(cfg *config.Config, registry *tunnel.Registry, certManager interface{ GetTLSConfig() *tls.Config }) *CombinedServer {
	cs := &CombinedServer{
		config:      cfg,
		registry:    registry,
		certManager: certManager,
	}

	// Create WebSocket handler (but don't start its server)
	cs.wsHandler = &Server{
		config:      cfg,
		registry:    registry,
		certManager: certManager,
	}

	// Create combined mux
	mux := http.NewServeMux()

	// WebSocket endpoints
	mux.HandleFunc("/tunnel", cs.wsHandler.handleWebSocket)
	mux.HandleFunc("/health", cs.wsHandler.handleHealth)

	// All other requests go to the proxy
	mux.HandleFunc("/", cs.handleProxyOrWebSocket)

	// Get TLS config and disable HTTP/2 (required for connection hijacking)
	tlsConfig := certManager.GetTLSConfig()
	tlsConfig.NextProtos = []string{"http/1.1"} // Force HTTP/1.1, disable HTTP/2

	// HTTPS server on 443
	cs.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPSPort),
		Handler:      mux,
		TLSConfig:    tlsConfig,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	// HTTP server on 80 (for redirects and ACME)
	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/", cs.handleHTTPRedirect)

	cs.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      certManager.(interface{ HTTPHandler() func(http.Handler) http.Handler }).HTTPHandler()(httpMux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	return cs
}

// Start starts the combined server
func (cs *CombinedServer) Start() error {
	// Start HTTP server (for redirects and ACME)
	go func() {
		log.Printf("HTTP server listening on port %d (redirects to HTTPS)", cs.config.HTTPPort)
		if err := cs.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Start HTTPS server (WebSocket + Proxy)
	log.Printf("Combined server (HTTPS + WSS) listening on port %d", cs.config.HTTPSPort)
	return cs.server.ListenAndServeTLS("", "")
}

// Shutdown gracefully shuts down the combined server
func (cs *CombinedServer) Shutdown(ctx context.Context) error {
	log.Println("Shutting down combined server...")

	var err error
	if shutdownErr := cs.httpServer.Shutdown(ctx); shutdownErr != nil {
		err = shutdownErr
	}

	if shutdownErr := cs.server.Shutdown(ctx); shutdownErr != nil {
		err = shutdownErr
	}

	return err
}

// handleProxyOrWebSocket routes requests to either WebSocket or proxy
func (cs *CombinedServer) handleProxyOrWebSocket(w http.ResponseWriter, r *http.Request) {
	// Check if it's a WebSocket upgrade request
	if r.Header.Get("Upgrade") == "websocket" {
		cs.wsHandler.handleWebSocket(w, r)
		return
	}

	// Otherwise, handle as proxy request
	cs.handleProxy(w, r)
}

// handleProxy handles HTTP proxy requests
func (cs *CombinedServer) handleProxy(w http.ResponseWriter, r *http.Request) {
	// Extract subdomain from Host header
	host := r.Host
	subdomain := cs.extractSubdomain(host)

	if subdomain == "" {
		http.Error(w, "Invalid hostname", http.StatusNotFound)
		return
	}

	// Look up tunnel by subdomain
	tun, exists := cs.registry.Get(subdomain)
	if !exists {
		log.Printf("Subdomain not found: %s", subdomain)
		http.Error(w, fmt.Sprintf("Tunnel not found for subdomain: %s", subdomain), http.StatusNotFound)
		return
	}

	// Hijack the connection for raw TCP forwarding
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		log.Printf("Response writer does not support hijacking")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		log.Printf("Failed to hijack connection: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Forward the request to the tunnel
	go func() {
		defer clientConn.Close()

		// Dial through the tunnel to the local server
		tunnelConn, err := proxy.DialThroughTunnel(tun)
		if err != nil {
			log.Printf("Failed to dial through tunnel for %s: %v", subdomain, err)
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

		// Set timeout on client connection
		if cs.config.RequestTimeout > 0 {
			clientConn.SetDeadline(time.Now().Add(cs.config.RequestTimeout))
		}

		// Bidirectional copy
		proxy.CopyBidirectional(clientConn, tunnelConn)
	}()
}

// handleHTTPRedirect redirects HTTP to HTTPS
func (cs *CombinedServer) handleHTTPRedirect(w http.ResponseWriter, r *http.Request) {
	target := "https://" + r.Host + r.URL.Path
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	http.Redirect(w, r, target, http.StatusMovedPermanently)
}

// extractSubdomain extracts the subdomain from a host header
func (cs *CombinedServer) extractSubdomain(host string) string {
	// Remove port if present
	if colonIndex := strings.Index(host, ":"); colonIndex != -1 {
		host = host[:colonIndex]
	}

	// Check if host ends with our domain
	domain := "." + cs.config.Domain
	if !strings.HasSuffix(host, domain) && host != cs.config.Domain {
		if host == cs.config.Domain {
			return ""
		}
		return ""
	}

	// Extract subdomain
	subdomain := strings.TrimSuffix(host, domain)
	subdomain = strings.TrimSpace(subdomain)

	return subdomain
}
