.PHONY: build run clean docker-build docker-run docker-stop test client-install client-run

# Build the server binary
build:
	@echo "Building tunnel server..."
	@mkdir -p bin
	@go build -o bin/tunnel-server ./cmd/server
	@echo "Build complete: bin/tunnel-server"

# Run the server
run: build
	@echo "Starting tunnel server..."
	@./bin/tunnel-server

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -f ssh_host_key
	@echo "Clean complete"

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	@docker build -t tunnel-server .
	@echo "Docker image built: tunnel-server"

# Run with docker-compose
docker-run:
	@echo "Starting tunnel server with docker-compose..."
	@docker-compose up -d
	@echo "Server started. View logs with: docker-compose logs -f"

# Stop docker-compose
docker-stop:
	@echo "Stopping tunnel server..."
	@docker-compose down
	@echo "Server stopped"

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@echo "Dependencies installed"

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Code formatted"

# Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run ./... || echo "golangci-lint not installed"

# Install client dependencies
client-install:
	@echo "Installing client dependencies..."
	@cd client && npm install
	@echo "Client dependencies installed"

# Run the Node.js client
client-run:
	@echo "Starting tunnel client..."
	@cd client && node client.js $(SUBDOMAIN) $(PORT)

# Show help
help:
	@echo "Available targets:"
	@echo "  build         - Build the server binary"
	@echo "  run           - Build and run the server"
	@echo "  clean         - Clean build artifacts"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run with docker-compose"
	@echo "  docker-stop   - Stop docker-compose"
	@echo "  test          - Run tests"
	@echo "  deps          - Install dependencies"
	@echo "  fmt           - Format code"
	@echo "  lint          - Run linter"
	@echo "  client-install - Install client dependencies"
	@echo "  client-run    - Run client (use: make client-run SUBDOMAIN=myapp PORT=3000)"
	@echo "  help          - Show this help message"
