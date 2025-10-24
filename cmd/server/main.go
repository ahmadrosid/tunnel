package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ahmadrosid/tunnel/internal/cert"
	"github.com/ahmadrosid/tunnel/internal/config"
	"github.com/ahmadrosid/tunnel/internal/proxy"
	"github.com/ahmadrosid/tunnel/internal/tunnel"
	"github.com/ahmadrosid/tunnel/internal/websocket"
)

func main() {
	log.Println("Starting tunnel server...")

	// Load configuration
	cfg := config.Load()
	log.Printf("Configuration loaded: WebSocket Port=%d, Domain=%s, HTTP Port=%d, HTTPS Port=%d",
		cfg.WebSocketPort, cfg.Domain, cfg.HTTPPort, cfg.HTTPSPort)

	// Create tunnel registry
	registry := tunnel.NewRegistry()

	// Create certificate manager for TLS
	certManager := cert.NewManager(cfg)

	// Check if WebSocket and HTTPS are on the same port
	if cfg.WebSocketPort == cfg.HTTPSPort && cfg.EnableHTTPS {
		log.Printf("WebSocket and HTTPS sharing port %d - using combined server", cfg.HTTPSPort)

		// Create combined server that handles both WebSocket and proxy on same port
		combinedServer := websocket.NewCombinedServer(cfg, registry, certManager)

		// Handle graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		// Start combined server
		go func() {
			if err := combinedServer.Start(); err != nil {
				log.Fatalf("Combined server error: %v", err)
			}
		}()

		// Wait for shutdown signal
		<-sigChan
		log.Println("\nShutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := combinedServer.Shutdown(ctx); err != nil {
			log.Printf("Error during shutdown: %v", err)
		}
	} else {
		// Run separate servers on different ports
		wsServer := websocket.NewServer(cfg, registry, certManager)
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

		// Start WebSocket server in a goroutine
		go func() {
			if err := wsServer.Start(); err != nil {
				log.Fatalf("WebSocket server error: %v", err)
			}
		}()

		// Wait for shutdown signal
		<-sigChan
		log.Println("\nShutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := proxyServer.Shutdown(ctx); err != nil {
			log.Printf("Error during proxy shutdown: %v", err)
		}

		if err := wsServer.Shutdown(); err != nil {
			log.Printf("Error during WebSocket shutdown: %v", err)
		}
	}

	log.Println("Server stopped")
	os.Exit(0)
}
