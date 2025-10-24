package websocket

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ahmadrosid/tunnel/internal/config"
	"github.com/ahmadrosid/tunnel/internal/tunnel"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512 * 1024 // 512KB
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for now - can be restricted in production
		return true
	},
}

// Server represents the WebSocket server
type Server struct {
	config      *config.Config
	registry    *tunnel.Registry
	server      *http.Server
	certManager interface {
		GetTLSConfig() *tls.Config
		GetTLSConfigForHijacking() *tls.Config
	}
}

// NewServer creates a new WebSocket server
func NewServer(cfg *config.Config, registry *tunnel.Registry, certManager interface {
	GetTLSConfig() *tls.Config
	GetTLSConfigForHijacking() *tls.Config
}) *Server {
	s := &Server{
		config:      cfg,
		registry:    registry,
		certManager: certManager,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/tunnel", s.handleWebSocket)
	mux.HandleFunc("/health", s.handleHealth)

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.WebSocketPort),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	// Add TLS config if HTTPS is enabled
	// Use GetTLSConfigForHijacking to disable HTTP/2 (required for connection hijacking)
	if cfg.EnableHTTPS && certManager != nil {
		s.server.TLSConfig = certManager.GetTLSConfigForHijacking()
	}

	return s
}

// Start starts the WebSocket server
func (s *Server) Start() error {
	// If WebSocket is on HTTPS port and HTTPS is enabled, use TLS
	if s.config.EnableHTTPS && s.config.WebSocketPort == s.config.HTTPSPort && s.certManager != nil {
		log.Printf("WebSocket server (WSS) listening on port %d", s.config.WebSocketPort)
		return s.server.ListenAndServeTLS("", "")
	}

	log.Printf("WebSocket server (WS) listening on port %d", s.config.WebSocketPort)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the WebSocket server
func (s *Server) Shutdown() error {
	log.Println("Shutting down WebSocket server...")
	return s.server.Close()
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK\n")
}

// handleWebSocket handles WebSocket upgrade and connection
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	log.Printf("New WebSocket connection from %s", r.RemoteAddr)

	// Handle the WebSocket connection
	go s.handleConnection(conn)
}

// handleConnection manages a WebSocket connection
func (s *Server) handleConnection(conn *websocket.Conn) {
	defer func() {
		conn.Close()
		log.Printf("WebSocket connection closed: %s", conn.RemoteAddr())
	}()

	// Configure connection
	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// Start ping ticker to keep connection alive
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	// Create connection wrapper
	wsConn := NewConnection(conn)

	// Handle messages from client
	handler := NewHandler(s.config, s.registry, wsConn)

	// Start ping routine
	go func() {
		for range ticker.C {
			if err := wsConn.WritePing(); err != nil {
				log.Printf("Failed to send ping: %v", err)
				return
			}
		}
	}()

	// Process incoming messages
	if err := handler.HandleMessages(); err != nil {
		log.Printf("Handler error: %v", err)
	}
}
