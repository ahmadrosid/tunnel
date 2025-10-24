# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary with version info
ARG VERSION=dev
ARG COMMIT=unknown
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags "-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT}" \
    -o tunnel-server ./cmd/server

# Runtime stage
FROM alpine:latest

# Add image labels
LABEL org.opencontainers.image.source="https://github.com/ahmadrosid/tunnel"
LABEL org.opencontainers.image.description="SSH tunneling service like Serveo"
LABEL org.opencontainers.image.licenses="MIT"

# Install ca-certificates for HTTPS/TLS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 tunnel && \
    adduser -D -u 1000 -G tunnel tunnel

WORKDIR /home/tunnel

# Copy binary from builder
COPY --from=builder /app/tunnel-server .

# Create directories for certs and keys
RUN mkdir -p /home/tunnel/certs && \
    chown -R tunnel:tunnel /home/tunnel

# Switch to non-root user
USER tunnel

# Expose SSH port
EXPOSE 2222

# Expose HTTP and HTTPS ports
EXPOSE 80 443

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD netstat -an | grep 2222 > /dev/null || exit 1

# Run the server
CMD ["./tunnel-server"]
