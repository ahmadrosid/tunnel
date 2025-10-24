package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ahmadrosid/tunnel/internal/config"
	"github.com/ahmadrosid/tunnel/internal/proxy"
	sshserver "github.com/ahmadrosid/tunnel/internal/ssh"
	"github.com/ahmadrosid/tunnel/internal/tunnel"
)

func main() {
	log.Println("Starting tunnel server...")

	// Load configuration
	cfg := config.Load()
	log.Printf("Configuration loaded: SSH Port=%d, Domain=%s, HTTP Port=%d, HTTPS Port=%d",
		cfg.SSHPort, cfg.Domain, cfg.HTTPPort, cfg.HTTPSPort)

	// Create tunnel registry
	registry := tunnel.NewRegistry()

	// Create SSH server
	sshServer, err := sshserver.NewServer(cfg, registry)
	if err != nil {
		log.Fatalf("Failed to create SSH server: %v", err)
	}

	// Create HTTP/HTTPS proxy server
	proxyServer := proxy.NewServer(cfg, registry)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start proxy server in a goroutine
	go func() {
		if err := proxyServer.Start(); err != nil {
			log.Fatalf("Proxy server error: %v", err)
		}
	}()

	// Start SSH server in a goroutine
	go func() {
		if err := sshServer.Start(); err != nil {
			log.Fatalf("SSH server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Println("\nShutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := proxyServer.Shutdown(ctx); err != nil {
		log.Printf("Error during proxy shutdown: %v", err)
	}

	log.Println("Server stopped")
	os.Exit(0)
}
