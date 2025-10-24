# Tunnel - SSH Tunneling Service

A Serveo-like SSH tunneling service that exposes your local web server to the internet through SSH reverse port forwarding.

## Features

- **Anonymous SSH connections** - No authentication required
- **Custom subdomains** - Request your own subdomain or get a random one
- **HTTP/HTTPS support** - Automatic Let's Encrypt certificates
- **Easy to use** - Just SSH, no installation needed on client side
- **Configurable** - Environment variables for all settings
- **Direct TCP forwarding** - Efficient traffic routing through SSH tunnels

## Prerequisites

### DNS Configuration (Production)

For the tunnel service to work properly, you need to configure a wildcard DNS record:

1. Add an A record for your domain pointing to your server's IP:
   ```
   unggahin.com → YOUR_SERVER_IP
   ```

2. Add a wildcard A record for all subdomains:
   ```
   *.unggahin.com → YOUR_SERVER_IP
   ```

**Example DNS Configuration:**
```
Type  Name              Value
A     unggahin.com      203.0.113.10
A     *.unggahin.com    203.0.113.10
```

### Local Testing

For local testing without DNS, you can:

1. **Disable HTTPS** and test HTTP only:
   ```bash
   export ENABLE_HTTPS=false
   ```

2. **Edit `/etc/hosts`** to test specific subdomains:
   ```
   127.0.0.1  myapp.unggahin.com
   127.0.0.1  test.unggahin.com
   ```

## Quick Start

### Option 1: Docker from GHCR (Recommended for Production)

Pull and run the pre-built image from GitHub Container Registry:

```bash
# Quick start with docker-compose
DOMAIN=your-domain.com LETSENCRYPT_EMAIL=your@email.com \
  docker-compose -f docker-compose.prod.yml up -d

# Or run directly with docker
docker run -d \
  -p 2222:2222 \
  -p 80:80 \
  -p 443:443 \
  -e DOMAIN=your-domain.com \
  -e LETSENCRYPT_EMAIL=your@email.com \
  -v tunnel-keys:/home/tunnel \
  -v tunnel-certs:/home/tunnel/certs \
  --name tunnel-server \
  --restart unless-stopped \
  ghcr.io/ahmadrosid/tunnel:latest

# View logs
docker logs -f tunnel-server

# Stop the server
docker stop tunnel-server
```

**Available tags:**
- `latest` - Latest stable release
- `v*.*.*` - Specific version (e.g., v1.0.0)
- `main` - Latest from main branch

### Option 2: Docker (Build Locally)

```bash
# Build and run with docker-compose
docker-compose up -d

# View logs
docker-compose logs -f

# Stop the server
docker-compose down
```

### Option 3: Build from Source

```bash
# Using Makefile (recommended)
make run

# Or manually
go build -o bin/tunnel-server ./cmd/server
./bin/tunnel-server
```

The server will start on port 2222 by default (SSH), port 80 (HTTP), and port 443 (HTTPS).

### Connect as Client

#### Random Subdomain

```bash
ssh -R 80:localhost:3000 unggahin.com -p 2222
```

This will:
- Connect to the tunnel server
- Forward HTTP traffic from a random subdomain (e.g., `a1b2c3d4.unggahin.com`)
- Route it to your local server at `localhost:3000`

#### Custom Subdomain

```bash
ssh -R 80:localhost:3000 myapp@unggahin.com -p 2222
```

This will:
- Request the custom subdomain `myapp.unggahin.com`
- Forward traffic to your local server at `localhost:3000`
- Fail if the subdomain is already taken

## Configuration

Configuration is done via environment variables with sensible defaults:

| Variable | Default | Description |
|----------|---------|-------------|
| `SSH_PORT` | 2222 | SSH server port |
| `DOMAIN` | unggahin.com | Base domain for tunnels |
| `HTTP_PORT` | 80 | HTTP proxy port |
| `HTTPS_PORT` | 443 | HTTPS proxy port |
| `HOST_KEY_PATH` | ./ssh_host_key | Path to SSH host key |
| `CERT_CACHE_DIR` | ./certs | Certificate cache directory |
| `LETSENCRYPT_EMAIL` | (empty) | Email for Let's Encrypt notifications |
| `REQUEST_TIMEOUT` | 30s | Timeout for proxied requests |
| `ENABLE_HTTPS` | true | Enable HTTPS with Let's Encrypt |

Example:

```bash
export SSH_PORT=2222
export DOMAIN=mytunnel.dev
export LETSENCRYPT_EMAIL=admin@mytunnel.dev
export ENABLE_HTTPS=true
./bin/tunnel-server
```

Or copy `.env.example` to `.env` and modify as needed.

## Architecture

```
Internet User → HTTP/HTTPS Proxy → Subdomain Lookup → SSH Tunnel → Client's Local Server
                      ↓
                Let's Encrypt
                (Auto Certs)
```

**Traffic Flow:**
1. Client creates SSH tunnel: `ssh -R 80:localhost:3000 myapp@unggahin.com -p 2222`
2. Server assigns subdomain and registers tunnel
3. Internet user visits: `https://myapp.unggahin.com`
4. Proxy extracts subdomain "myapp", looks up tunnel in registry
5. Proxy opens direct-tcpip connection through SSH to `localhost:3000`
6. Request is forwarded, response comes back through tunnel
7. Proxy returns response to internet user

## Testing

### Basic HTTP Test

```bash
# Terminal 1: Start the server (with HTTPS disabled for local testing)
export ENABLE_HTTPS=false
make run

# Terminal 2: Start a local web server
python3 -m http.server 3000

# Terminal 3: Create a tunnel
ssh -R 80:localhost:3000 myapp@localhost -p 2222

# Terminal 4: Test the tunnel
curl http://myapp.unggahin.com
```

### Testing Different Scenarios

**1. Test 404 (subdomain not found):**
```bash
curl http://nonexistent.unggahin.com
# Expected: 404 Not Found
```

**2. Test 502 (local server down):**
```bash
# Create tunnel, then stop your local server
ssh -R 80:localhost:3000 myapp@localhost -p 2222
# Stop python server
curl http://myapp.unggahin.com
# Expected: 502 Bad Gateway
```

**3. Test HTTPS (production with DNS):**
```bash
export ENABLE_HTTPS=true
export LETSENCRYPT_EMAIL=your-email@example.com
make run

# Let's Encrypt will automatically provision certificates
curl https://myapp.unggahin.com
```

## Troubleshooting

### HTTPS not working
- Ensure ports 80 and 443 are open on your server
- Verify wildcard DNS is configured correctly
- Check Let's Encrypt rate limits (5 certificates per domain per week)
- View certificate cache: `ls -la ./certs/`

### Connection timeout
- Increase `REQUEST_TIMEOUT` environment variable
- Check if local server is responding
- Verify SSH tunnel is still active

### Subdomain conflicts
- Use custom subdomains to avoid conflicts
- Check active tunnels in server logs

### Port permission denied (Linux)
If running on ports 80/443 requires sudo:
```bash
# Option 1: Use higher ports
export HTTP_PORT=8080
export HTTPS_PORT=8443

# Option 2: Grant capability (Linux)
sudo setcap 'cap_net_bind_service=+ep' ./bin/tunnel-server
```

## Docker Images

Pre-built Docker images are available on GitHub Container Registry:

**Registry:** `ghcr.io/ahmadrosid/tunnel`

**Available Tags:**
- `latest` - Latest stable release from main branch
- `v*.*.*` - Specific version tags (e.g., `v1.0.0`, `v1.0.1`)
- `main` - Latest commit from main branch (bleeding edge)
- `<branch>-<sha>` - Specific commit builds

**Pull Image:**
```bash
docker pull ghcr.io/ahmadrosid/tunnel:latest
```

**View Available Tags:**
Visit: https://github.com/ahmadrosid/tunnel/pkgs/container/tunnel

### Multi-Architecture Support

Images are built for multiple architectures:
- `linux/amd64` - Intel/AMD 64-bit (most common)
- `linux/arm64` - ARM 64-bit (Apple Silicon, AWS Graviton, etc.)

Docker will automatically pull the correct image for your platform.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

MIT
