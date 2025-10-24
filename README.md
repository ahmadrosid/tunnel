# Tunnel - WebSocket Tunneling Service

Expose your local web server to the internet through secure WebSocket connections. 

## Features

- ⚡ **WebSocket-based** - Simple, reliable persistent connections (WSS)
- 🔒 **Automatic HTTPS** - Let's Encrypt certificates managed automatically
- 🎯 **Custom subdomains** - Choose your own or get a random one
- 🌐 **Cross-platform** - Works anywhere with WebSocket support
- 🚀 **No Nginx needed** - Pure Go handles everything on port 443
- 📦 **Easy deployment** - Docker or standalone binary

## Quick Start

### 1. Deploy Server

**Docker (Recommended):**
```bash
# Create .env file
cat > .env << EOF
DOMAIN=your-domain.com
LETSENCRYPT_EMAIL=admin@your-domain.com
ENABLE_HTTPS=true
EOF

# Deploy
docker-compose -f docker-compose.prod.yml up -d

# Check logs
docker-compose -f docker-compose.prod.yml logs -f
```

**From Source:**
```bash
# Build
go build -o bin/tunnel-server ./cmd/server

# Run
WS_PORT=443 DOMAIN=your-domain.com ENABLE_HTTPS=true ./bin/tunnel-server
```

See [DOCKER-DEPLOY.md](DOCKER-DEPLOY.md) for detailed deployment guide.

### 2. Configure DNS

Add these DNS records for your domain:

```
Type  Name              Value
A     your-domain.com   YOUR_SERVER_IP
A     *.your-domain.com YOUR_SERVER_IP
```

### 3. Connect Client

**Quick Start:**
```bash
cd client
npm install
node client.js myapp 3000
```

Your local server at `localhost:3000` is now accessible at:
```
https://myapp.your-domain.com
```

## Client Setup

### Node.js Client (Recommended)

```bash
cd client

# Install dependencies
npm install

# Configure (optional - copy and edit .env)
cp .env.example .env

# Run with custom subdomain
node client.js myapp 3000

# Or with random subdomain
node client.js - 3000
```

See [client/README.md](client/README.md) for full client documentation.

### Browser Demo

Open `client/client.html` in your browser for an interactive demo with UI.

### Build Your Own Client

Connect to `wss://your-domain.com/tunnel` and send a registration message:

**Registration:**
```json
{
  "type": "register",
  "timestamp": "2025-10-24T12:00:00.000Z",
  "data": {
    "subdomain": "myapp",
    "local_addr": "localhost:3000",
    "local_port": 3000
  }
}
```

**Success Response:**
```json
{
  "type": "success",
  "timestamp": "2025-10-24T12:00:00.000Z",
  "data": {
    "tunnel_id": "uuid",
    "subdomain": "myapp",
    "full_domain": "myapp.your-domain.com",
    "local_addr": "localhost:3000",
    "message": "Tunnel created: https://myapp.your-domain.com -> localhost:3000"
  }
}
```

**Keep-Alive:**
Send ping messages every 30 seconds:
```json
{
  "type": "ping",
  "timestamp": "2025-10-24T12:00:00.000Z"
}
```

**Example (Go):**
```go
conn, _, _ := websocket.DefaultDialer.Dial("wss://your-domain.com/tunnel", nil)
conn.WriteJSON(map[string]interface{}{
    "type": "register",
    "data": map[string]interface{}{
        "subdomain":  "myapp",
        "local_addr": "localhost:3000",
        "local_port": 3000,
    },
})
```

See [client/README.md](client/README.md) for Python and other examples.

## Architecture

```
┌─────────────┐
│   Client    │
│ (Your Code) │
└──────┬──────┘
       │ WSS (port 443)
       ▼
┌─────────────────────┐
│   Tunnel Server     │
│  - Port 80: HTTP    │
│  - Port 443: HTTPS  │
│               + WSS │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│  Internet Users     │
│ https://*.your.com  │
└─────────────────────┘
```

**Flow:**
1. Client connects via WSS on port 443
2. Server assigns subdomain (e.g., `myapp.your-domain.com`)
3. Internet users visit the subdomain
4. Traffic routes through WebSocket to client
5. Client forwards to local server

**Key Features:**
- Single port (443) handles both HTTPS proxy and WebSocket
- Automatic SSL/TLS with Let's Encrypt
- No SSH required
- Native Go implementation (no Nginx/Apache needed)

## Configuration

### Server Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DOMAIN` | (required) | Your domain name |
| `WS_PORT` | 443 | WebSocket server port |
| `HTTP_PORT` | 80 | HTTP server port |
| `HTTPS_PORT` | 443 | HTTPS server port |
| `ENABLE_HTTPS` | true | Enable HTTPS/WSS with Let's Encrypt |
| `LETSENCRYPT_EMAIL` | (empty) | Email for Let's Encrypt notifications |
| `REQUEST_TIMEOUT` | 30s | Timeout for proxied requests |
| `CERT_CACHE_DIR` | ./certs | Certificate cache directory |

### Client Environment Variables

Create `client/.env`:
```env
TUNNEL_SERVER=wss://your-domain.com/tunnel
SUBDOMAIN=myapp
LOCAL_PORT=3000
LOCAL_HOST=localhost
```

## Deployment

### Production Deployment

See [DOCKER-DEPLOY.md](DOCKER-DEPLOY.md) for complete production deployment guide.

**Quick Deploy:**
```bash
# On your server
cat > .env << EOF
DOMAIN=your-domain.com
LETSENCRYPT_EMAIL=admin@your-domain.com
ENABLE_HTTPS=true
EOF

docker-compose -f docker-compose.prod.yml up -d
```

### Development

```bash
# Run server locally
WS_PORT=8080 HTTP_PORT=8081 ENABLE_HTTPS=false make run

# Run client (another terminal)
cd client
TUNNEL_SERVER=ws://localhost:8080/tunnel node client.js myapp 3000
```

## Troubleshooting

### Port 443 already in use
```bash
# Check what's using port 443
sudo lsof -i :443

# Stop conflicting service (e.g., nginx)
sudo systemctl stop nginx
```

### Certificate issues
```bash
# Remove old certificates
docker-compose -f docker-compose.prod.yml down
docker volume rm tunnel_tunnel-certs
docker-compose -f docker-compose.prod.yml up -d
```

### Client connection timeout
```bash
# Verify server is accessible
curl -I https://your-domain.com/health

# Check WebSocket endpoint
wscat -c wss://your-domain.com/tunnel
```

### DNS not resolving
```bash
# Test DNS
nslookup your-domain.com
nslookup test.your-domain.com

# Verify wildcard works
dig +short *.your-domain.com
```

## Migration from SSH

This project started as an SSH-based tunnel (like Serveo/ngrok). We've migrated to WebSocket for:
- ✅ Simpler protocol (no SSH key management)
- ✅ Better browser compatibility
- ✅ Easier debugging
- ✅ More flexible authentication options
- ✅ Native HTTPS support

Old SSH approach is removed. If you need SSH tunneling, check git history or use established tools like ngrok.

## Contributing

Contributions welcome! Please:
1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing`)
5. Open Pull Request

## License

MIT License - see LICENSE file for details.

## Support

- 📖 Documentation: [DOCKER-DEPLOY.md](DOCKER-DEPLOY.md), [client/README.md](client/README.md)
- 🐛 Issues: [GitHub Issues](https://github.com/ahmadrosid/tunnel/issues)
- 💬 Discussions: [GitHub Discussions](https://github.com/ahmadrosid/tunnel/discussions)

## Credits

Built with:
- [gorilla/websocket](https://github.com/gorilla/websocket) - WebSocket implementation
- [golang.org/x/crypto/acme/autocert](https://pkg.go.dev/golang.org/x/crypto/acme/autocert) - Let's Encrypt integration

Inspired by [Serveo](https://serveo.net) and [ngrok](https://ngrok.com).
