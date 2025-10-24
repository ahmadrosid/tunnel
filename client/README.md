# Tunnel Clients

WebSocket-based tunnel clients for exposing your local web server to the internet.

## Node.js Client

### Installation

```bash
cd client
npm install
```

### Usage

```bash
# With custom subdomain
node client.js myapp 3000

# With random subdomain
node client.js - 3000

# Custom server
TUNNEL_SERVER=ws://your-domain.com:8080/tunnel node client.js myapp 3000
```

### Arguments

- `subdomain` - Your desired subdomain (use `-` for random)
- `local-port` - Port where your local server is running (default: 3000)

### Environment Variables

- `TUNNEL_SERVER` - Tunnel server WebSocket URL (default: `ws://localhost:8080/tunnel`)

### Example

```bash
# Start your local server
python3 -m http.server 3000

# In another terminal, start the tunnel
node client.js myapp 3000
```

You'll see output like:
```
🚀 Starting tunnel client...
📍 Server: ws://localhost:8080/tunnel
🏠 Local: http://localhost:3000
🏷️  Subdomain: myapp

✅ Connected to tunnel server
📤 Sent registration request...

✨ Tunnel created successfully!
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🌐 Public URL: https://myapp.easypod.cloud
📌 Subdomain: myapp
🔗 Forwarding to: localhost:3000
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

💡 Press Ctrl+C to stop the tunnel
```

## Browser Demo

Open `client.html` in your browser for an interactive demo with a nice UI.

This is a demonstration client that shows:
- How to connect to the tunnel server
- How to register a tunnel
- How to receive success/error messages

**Note:** The browser demo doesn't implement full HTTP forwarding since browsers can't proxy local servers directly. Use the Node.js client for actual tunneling.

## Protocol

### Register Tunnel

Send this JSON message after connecting:

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

### Success Response

```json
{
  "type": "success",
  "timestamp": "2025-10-24T12:00:00.000Z",
  "data": {
    "tunnel_id": "uuid-here",
    "subdomain": "myapp",
    "full_domain": "myapp.easypod.cloud",
    "local_addr": "localhost:3000",
    "message": "Tunnel created: https://myapp.easypod.cloud -> localhost:3000"
  }
}
```

### Error Response

```json
{
  "type": "error",
  "timestamp": "2025-10-24T12:00:00.000Z",
  "error": "Subdomain 'myapp' is already in use"
}
```

### Keep-Alive (Ping/Pong)

Send periodically to keep connection alive:

```json
{
  "type": "ping",
  "timestamp": "2025-10-24T12:00:00.000Z"
}
```

Server responds with:

```json
{
  "type": "pong",
  "timestamp": "2025-10-24T12:00:00.000Z"
}
```

### Unregister

Send before closing connection:

```json
{
  "type": "unregister",
  "timestamp": "2025-10-24T12:00:00.000Z"
}
```

## Building Your Own Client

To create a client in another language:

1. **Connect** to `ws://your-domain.com:8080/tunnel`
2. **Send register message** with your desired subdomain and local port
3. **Wait for success response** with your public URL
4. **Forward HTTP traffic** - Listen for binary WebSocket messages (HTTP requests), forward them to your local server, and send responses back
5. **Send pings** every 30 seconds to keep the connection alive
6. **Handle disconnection** - Send unregister message before closing

### Example (Python)

```python
import asyncio
import websockets
import json
from datetime import datetime

async def tunnel_client():
    uri = "ws://localhost:8080/tunnel"

    async with websockets.connect(uri) as websocket:
        # Register tunnel
        await websocket.send(json.dumps({
            "type": "register",
            "timestamp": datetime.utcnow().isoformat() + "Z",
            "data": {
                "subdomain": "myapp",
                "local_addr": "localhost:3000",
                "local_port": 3000
            }
        }))

        # Receive response
        response = json.loads(await websocket.recv())
        if response["type"] == "success":
            print(f"Tunnel: {response['data']['full_domain']}")

        # Handle incoming messages
        async for message in websocket:
            # Process HTTP traffic...
            pass

asyncio.run(tunnel_client())
```

## Troubleshooting

### Connection refused
- Make sure the tunnel server is running
- Check the server URL and port

### Subdomain already in use
- Choose a different subdomain
- Or use `-` for a random subdomain

### Local server not responding
- Verify your local server is running on the specified port
- Test it locally: `curl http://localhost:3000`

## License

MIT
