package ssh

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ahmadrosid/tunnel/internal/subdomain"
	"github.com/ahmadrosid/tunnel/internal/tunnel"
	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"
)

// handleRequests processes global SSH requests (like remote port forwarding)
func (s *Server) handleRequests(reqs <-chan *ssh.Request, sshConn ssh.Conn) {
	for req := range reqs {
		switch req.Type {
		case "tcpip-forward":
			s.handleTCPIPForward(req, sshConn)
		case "cancel-tcpip-forward":
			s.handleCancelTCPIPForward(req, sshConn)
		default:
			if req.WantReply {
				req.Reply(false, nil)
			}
		}
	}
}

// handleTCPIPForward handles remote port forwarding requests
func (s *Server) handleTCPIPForward(req *ssh.Request, sshConn ssh.Conn) {
	// Parse the request payload
	type forwardRequest struct {
		BindAddr string
		BindPort uint32
	}

	var fwdReq forwardRequest
	if err := ssh.Unmarshal(req.Payload, &fwdReq); err != nil {
		log.Printf("Failed to unmarshal forward request: %v", err)
		if req.WantReply {
			req.Reply(false, nil)
		}
		return
	}

	log.Printf("Forward request: %s:%d", fwdReq.BindAddr, fwdReq.BindPort)

	// Determine subdomain (from username or generate random)
	var selectedSubdomain string
	username := sshConn.User()

	if username != "" && username != "root" {
		// Custom subdomain requested
		normalized := subdomain.Normalize(username)
		if err := subdomain.Validate(normalized); err != nil {
			errMsg := fmt.Sprintf("Invalid subdomain: %v", err)
			log.Printf("%s", errMsg)
			sshConn.SendRequest("error", false, []byte(errMsg))
			if req.WantReply {
				req.Reply(false, nil)
			}
			return
		}

		if !s.registry.IsSubdomainAvailable(normalized) {
			errMsg := fmt.Sprintf("Subdomain '%s' is already in use", normalized)
			log.Printf("%s", errMsg)
			sshConn.SendRequest("error", false, []byte(errMsg))
			if req.WantReply {
				req.Reply(false, nil)
			}
			return
		}

		selectedSubdomain = normalized
	} else {
		// Generate random subdomain
		var err error
		selectedSubdomain, err = subdomain.Generate()
		if err != nil {
			log.Printf("Failed to generate subdomain: %v", err)
			if req.WantReply {
				req.Reply(false, nil)
			}
			return
		}
	}

	// Create tunnel
	tun := &tunnel.Tunnel{
		ID:         uuid.New().String(),
		Subdomain:  selectedSubdomain,
		SSHConn:    sshConn,
		LocalAddr:  fmt.Sprintf("%s:%d", fwdReq.BindAddr, fwdReq.BindPort),
		RemotePort: int(fwdReq.BindPort),
		CreatedAt:  time.Now(),
	}

	// Register tunnel
	if err := s.registry.Register(tun); err != nil {
		log.Printf("Failed to register tunnel: %v", err)
		if req.WantReply {
			req.Reply(false, nil)
		}
		return
	}

	// Clean up on disconnect
	go func() {
		sshConn.Wait()
		s.registry.Unregister(selectedSubdomain)
		log.Printf("Tunnel closed: %s.%s", selectedSubdomain, s.config.Domain)
	}()

	// Send success message to client
	fullDomain := fmt.Sprintf("%s.%s", selectedSubdomain, s.config.Domain)
	welcomeMsg := fmt.Sprintf("\n\nForwarding HTTP traffic from:\nhttps://%s\n-> %s\n\n", fullDomain, tun.LocalAddr)

	log.Printf("Tunnel created: %s -> %s", fullDomain, tun.LocalAddr)

	// Reply with success
	if req.WantReply {
		// Reply with the port that was bound (in this case, we just echo back)
		type forwardResponse struct {
			Port uint32
		}
		req.Reply(true, ssh.Marshal(forwardResponse{Port: fwdReq.BindPort}))
	}

	// Send welcome message to the client
	sshConn.SendRequest("info", false, []byte(welcomeMsg))
}

// handleCancelTCPIPForward handles cancel forward requests
func (s *Server) handleCancelTCPIPForward(req *ssh.Request, sshConn ssh.Conn) {
	// For now, we just acknowledge the request
	// The tunnel will be cleaned up when the connection closes
	if req.WantReply {
		req.Reply(true, nil)
	}
}

// handleChannels processes SSH channels (like shell sessions)
func (s *Server) handleChannels(chans <-chan ssh.NewChannel, sshConn ssh.Conn) {
	for newChannel := range chans {
		go s.handleChannel(newChannel, sshConn)
	}
}

// handleChannel handles individual SSH channels
func (s *Server) handleChannel(newChannel ssh.NewChannel, sshConn ssh.Conn) {
	// We accept session channels to send messages to the client
	if newChannel.ChannelType() == "session" {
		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Printf("Failed to accept channel: %v", err)
			return
		}
		defer channel.Close()

		// Handle session requests (shell, exec, etc.)
		go func() {
			for req := range requests {
				switch req.Type {
				case "shell", "pty-req":
					// Accept these requests but don't actually provide a shell
					if req.WantReply {
						req.Reply(true, nil)
					}
				default:
					if req.WantReply {
						req.Reply(false, nil)
					}
				}
			}
		}()

		// Keep channel open for messages
		channel.Read(make([]byte, 1))
		return
	}

	// Handle forwarded-tcpip channels (actual traffic forwarding)
	if strings.HasPrefix(newChannel.ChannelType(), "forwarded-tcpip") {
		// This will be implemented in Step 2 when we add HTTP proxy
		newChannel.Reject(ssh.UnknownChannelType, "forwarding not yet implemented")
		return
	}

	// Reject other channel types
	newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
}
