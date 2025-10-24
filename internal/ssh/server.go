package ssh

import (
	"fmt"
	"log"
	"net"

	"github.com/ahmadrosid/tunnel/internal/config"
	"github.com/ahmadrosid/tunnel/internal/tunnel"
	"golang.org/x/crypto/ssh"
)

// Server represents the SSH server
type Server struct {
	config    *config.Config
	registry  *tunnel.Registry
	sshConfig *ssh.ServerConfig
}

func NewServer(cfg *config.Config, registry *tunnel.Registry) (*Server, error) {
	hostKey, err := LoadOrGenerateHostKey(cfg.HostKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load host key: %w", err)
	}

	sshConfig := &ssh.ServerConfig{
		NoClientAuth: true, // Allow anonymous connections
	}
	sshConfig.AddHostKey(hostKey)

	return &Server{
		config:    cfg,
		registry:  registry,
		sshConfig: sshConfig,
	}, nil
}

// Start starts the SSH server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.SSHPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	log.Printf("SSH server listening on port %d", s.config.SSHPort)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		// Handle connection in a goroutine
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(netConn net.Conn) {
	sshConn, chans, reqs, err := ssh.NewServerConn(netConn, s.sshConfig)
	if err != nil {
		log.Printf("Failed to handshake: %v", err)
		return
	}
	defer sshConn.Close()

	log.Printf("New SSH connection from %s (user: %s)", sshConn.RemoteAddr(), sshConn.User())

	go s.handleRequests(reqs, sshConn)
	go s.handleChannels(chans, sshConn)

	sshConn.Wait()
	log.Printf("Connection closed: %s", sshConn.RemoteAddr())
}
