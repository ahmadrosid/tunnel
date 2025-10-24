package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/ahmadrosid/tunnel/internal/config"
	"github.com/ahmadrosid/tunnel/internal/subdomain"
	"github.com/ahmadrosid/tunnel/internal/tunnel"
	"github.com/google/uuid"
)

// MessageType represents the type of WebSocket message
type MessageType string

const (
	MessageTypeRegister   MessageType = "register"
	MessageTypeUnregister MessageType = "unregister"
	MessageTypeSuccess    MessageType = "success"
	MessageTypeError      MessageType = "error"
	MessageTypeData       MessageType = "data"
	MessageTypePing       MessageType = "ping"
	MessageTypePong       MessageType = "pong"
)

// Message represents a WebSocket message
type Message struct {
	Type      MessageType     `json:"type"`
	Data      json.RawMessage `json:"data,omitempty"`
	Error     string          `json:"error,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
}

// RegisterRequest represents a tunnel registration request
type RegisterRequest struct {
	Subdomain string `json:"subdomain,omitempty"` // Empty for random subdomain
	LocalAddr string `json:"local_addr"`          // e.g., "localhost:3000"
	LocalPort int    `json:"local_port"`          // e.g., 3000
}

// RegisterResponse represents a tunnel registration response
type RegisterResponse struct {
	TunnelID   string `json:"tunnel_id"`
	Subdomain  string `json:"subdomain"`
	FullDomain string `json:"full_domain"`
	LocalAddr  string `json:"local_addr"`
	Message    string `json:"message"`
}

// Handler handles WebSocket messages
type Handler struct {
	config    *config.Config
	registry  *tunnel.Registry
	conn      *Connection
	tunnelID  string
	subdomain string
}

// NewHandler creates a new WebSocket handler
func NewHandler(cfg *config.Config, registry *tunnel.Registry, conn *Connection) *Handler {
	return &Handler{
		config:   cfg,
		registry: registry,
		conn:     conn,
	}
}

// HandleMessages processes incoming WebSocket messages
func (h *Handler) HandleMessages() error {
	for {
		msg, err := h.conn.ReadMessage()
		if err != nil {
			log.Printf("Failed to read message: %v", err)
			// Cleanup tunnel on disconnect
			if h.subdomain != "" {
				h.registry.Unregister(h.subdomain)
				log.Printf("Tunnel unregistered on disconnect: %s", h.subdomain)
			}
			return err
		}

		if err := h.handleMessage(msg); err != nil {
			log.Printf("Error handling message: %v", err)
			h.sendError(err.Error())
		}
	}
}

// handleMessage processes a single message
func (h *Handler) handleMessage(msg *Message) error {
	switch msg.Type {
	case MessageTypeRegister:
		return h.handleRegister(msg)
	case MessageTypeUnregister:
		return h.handleUnregister(msg)
	case MessageTypePing:
		return h.handlePing()
	case MessageTypeData:
		// Data messages are handled in the proxy layer
		return nil
	default:
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

// handleRegister handles tunnel registration
func (h *Handler) handleRegister(msg *Message) error {
	var req RegisterRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		return fmt.Errorf("invalid register request: %w", err)
	}

	// Determine subdomain
	var selectedSubdomain string
	if req.Subdomain != "" {
		// Custom subdomain requested
		normalized := subdomain.Normalize(req.Subdomain)
		if err := subdomain.Validate(normalized); err != nil {
			return fmt.Errorf("invalid subdomain: %w", err)
		}

		if !h.registry.IsSubdomainAvailable(normalized) {
			return fmt.Errorf("subdomain '%s' is already in use", normalized)
		}

		selectedSubdomain = normalized
	} else {
		// Generate random subdomain
		var err error
		selectedSubdomain, err = subdomain.Generate()
		if err != nil {
			return fmt.Errorf("failed to generate subdomain: %w", err)
		}
	}

	// Create tunnel
	tunnelID := uuid.New().String()
	localAddr := req.LocalAddr
	if localAddr == "" {
		localAddr = fmt.Sprintf("localhost:%d", req.LocalPort)
	}

	tun := &tunnel.Tunnel{
		ID:         tunnelID,
		Subdomain:  selectedSubdomain,
		WSConn:     h.conn,
		LocalAddr:  localAddr,
		RemotePort: req.LocalPort,
		CreatedAt:  time.Now(),
	}

	// Register tunnel
	if err := h.registry.Register(tun); err != nil {
		return fmt.Errorf("failed to register tunnel: %w", err)
	}

	h.tunnelID = tunnelID
	h.subdomain = selectedSubdomain

	// Send success response
	fullDomain := fmt.Sprintf("%s.%s", selectedSubdomain, h.config.Domain)
	response := RegisterResponse{
		TunnelID:   tunnelID,
		Subdomain:  selectedSubdomain,
		FullDomain: fullDomain,
		LocalAddr:  localAddr,
		Message:    fmt.Sprintf("Tunnel created: https://%s -> %s", fullDomain, localAddr),
	}

	log.Printf("Tunnel registered: %s -> %s", fullDomain, localAddr)

	return h.sendSuccess(response)
}

// handleUnregister handles tunnel unregistration
func (h *Handler) handleUnregister(msg *Message) error {
	if h.subdomain == "" {
		return fmt.Errorf("no tunnel registered")
	}

	h.registry.Unregister(h.subdomain)
	log.Printf("Tunnel unregistered: %s", h.subdomain)

	h.tunnelID = ""
	h.subdomain = ""

	return h.sendSuccess(map[string]string{
		"message": "Tunnel unregistered successfully",
	})
}

// handlePing handles ping messages
func (h *Handler) handlePing() error {
	return h.send(&Message{
		Type:      MessageTypePong,
		Timestamp: time.Now(),
	})
}

// sendSuccess sends a success message
func (h *Handler) sendSuccess(data interface{}) error {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return h.send(&Message{
		Type:      MessageTypeSuccess,
		Data:      dataBytes,
		Timestamp: time.Now(),
	})
}

// sendError sends an error message
func (h *Handler) sendError(errMsg string) error {
	return h.send(&Message{
		Type:      MessageTypeError,
		Error:     errMsg,
		Timestamp: time.Now(),
	})
}

// send sends a message to the client
func (h *Handler) send(msg *Message) error {
	return h.conn.WriteMessage(msg)
}
