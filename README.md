# Tunnel - WebSocket Tunneling Service

Expose your local web server to the internet through WebSocket connections.

## Features

- **WebSocket-based** - Simple, reliable persistent connections
- **Custom subdomains** - Choose your own or get a random one
- **HTTP/HTTPS support** - Automatic Let's Encrypt certificates
- **Cross-platform** - Works anywhere with WebSocket support

## Quick Start

### Run Server

```bash
# Docker (recommended)
DOMAIN=your-domain.com LETSENCRYPT_EMAIL=your@email.com \
  docker-compose -f docker-compose.prod.yml up -d

# Or from source
make run
```

### DNS Setup

Configure wildcard DNS for your domain:
```
A     your-domain.com      YOUR_SERVER_IP
A     *.your-domain.com    YOUR_SERVER_IP
```

### Connect Client

We provide ready-to-use clients in the [client/](client/) folder:

**Node.js Client:**
```bash
cd client
npm install

# Run with custom subdomain
node client.js myapp 3000

# Run with random subdomain
node client.js - 3000
```

**Browser Demo:**
```bash
open client/client.html
```

See [client/README.md](client/README.md) for full documentation.

**Or create your own client** that connects to `ws://your-domain.com:8080/tunnel`:

**Example (Go):**
```go
package main

import (
    "encoding/json"
    "log"
    "github.com/gorilla/websocket"
)

func main() {
    // Connect to tunnel server
    conn, _, err := websocket.DefaultDialer.Dial("ws://easypod.cloud:8080/tunnel", nil)
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    // Register tunnel with custom subdomain
    register := map[string]interface{}{
        "type": "register",
        "data": map[string]interface{}{
            "subdomain":  "myapp",
            "local_addr": "localhost:3000",
            "local_port": 3000,
        },
    }

    if err := conn.WriteJSON(register); err != nil {
        log.Fatal(err)
    }

    // Read response
    var response map[string]interface{}
    if err := conn.ReadJSON(&response); err != nil {
        log.Fatal(err)
    }

    log.Printf("Tunnel created: %v", response)

    // Handle incoming data...
    for {
        _, data, err := conn.ReadMessage()
        if err != nil {
            log.Fatal(err)
        }
        // Forward data to local server and send response back
    }
}
```

**Message Protocol:**

Register tunnel:
```json
{
  "type": "register",
  "data": {
    "subdomain": "myapp",
    "local_addr": "localhost:3000",
    "local_port": 3000
  }
}
```

Response:
```json
{
  "type": "success",
  "data": {
    "tunnel_id": "uuid",
    "subdomain": "myapp",
    "full_domain": "myapp.easypod.cloud",
    "message": "Tunnel created: https://myapp.easypod.cloud -> localhost:3000"
  }
}
```

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `WS_PORT` | 8080 | WebSocket server port |
| `DOMAIN` | easypod.cloud | Base domain |
| `HTTP_PORT` | 80 | HTTP proxy port |
| `HTTPS_PORT` | 443 | HTTPS proxy port |
| `ENABLE_HTTPS` | true | Enable HTTPS with Let's Encrypt |
| `LETSENCRYPT_EMAIL` | (empty) | Email for Let's Encrypt |
| `REQUEST_TIMEOUT` | 30s | Request timeout |

## Architecture

```
Internet → HTTP/HTTPS Proxy → Subdomain Lookup → WebSocket Tunnel → Local Server
```

## Docker

```bash
# Pull image
docker pull ghcr.io/ahmadrosid/tunnel:latest

# Run
docker run -d \
  -p 8080:8080 -p 80:80 -p 443:443 \
  -e DOMAIN=your-domain.com \
  -e LETSENCRYPT_EMAIL=your@email.com \
  ghcr.io/ahmadrosid/tunnel:latest
```

## License

MIT
