# Docker Deployment Guide

Quick guide to deploy the tunnel server with Docker.

## Prerequisites

- Docker and Docker Compose installed
- Domain pointing to your server (A record for `*.easypod.cloud` and `easypod.cloud`)
- Ports 80 and 443 open

## Quick Deploy

### 1. Create `.env` file

```bash
cat > .env << EOF
DOMAIN=easypod.cloud
LETSENCRYPT_EMAIL=admin@easypod.cloud
ENABLE_HTTPS=true
EOF
```

### 2. Deploy with Docker Compose

```bash
docker-compose -f docker-compose.prod.yml up -d
```

### 3. Check logs

```bash
docker-compose -f docker-compose.prod.yml logs -f
```

You should see:
```
tunnel-server-prod | WebSocket server (WSS) listening on port 443
tunnel-server-prod | HTTP proxy listening on port 80
tunnel-server-prod | HTTPS proxy listening on port 443
```

### 4. Test the tunnel

On your local machine:
```bash
cd client
cp .env.example .env
# Make sure .env has: TUNNEL_SERVER=wss://easypod.cloud/tunnel
node client.js myapp 3000
```

## Configuration

### Environment Variables

All set in `.env` file:

| Variable | Default | Description |
|----------|---------|-------------|
| `DOMAIN` | (required) | Your domain name |
| `LETSENCRYPT_EMAIL` | (optional) | Email for Let's Encrypt |
| `ENABLE_HTTPS` | true | Enable HTTPS/WSS |
| `REQUEST_TIMEOUT` | 30s | Request timeout |

### Port Configuration

The server runs:
- **Port 80**: HTTP proxy (redirects to HTTPS)
- **Port 443**: HTTPS proxy + WebSocket (WSS)

WebSocket clients connect to: `wss://easypod.cloud/tunnel`

## Management

### View logs
```bash
docker-compose -f docker-compose.prod.yml logs -f
```

### Restart server
```bash
docker-compose -f docker-compose.prod.yml restart
```

### Stop server
```bash
docker-compose -f docker-compose.prod.yml down
```

### Update to latest version
```bash
docker-compose -f docker-compose.prod.yml pull
docker-compose -f docker-compose.prod.yml up -d
```

## Troubleshooting

### Port 443 already in use
```bash
# Check what's using port 443
sudo lsof -i :443
# or
sudo netstat -tlnp | grep :443

# Stop the conflicting service (e.g., nginx)
sudo systemctl stop nginx
```

### Certificate issues
```bash
# Check certificate cache
ls -la certs/

# Remove old certificates
docker-compose -f docker-compose.prod.yml down
docker volume rm tunnel_tunnel-certs
docker-compose -f docker-compose.prod.yml up -d
```

### Connection timeout from client
```bash
# Verify server is running
docker ps

# Check if port 443 is accessible
curl -I https://easypod.cloud

# Test WebSocket endpoint
curl -I https://easypod.cloud/health
```

## DNS Configuration

Make sure you have these DNS records:

```
Type  Name              Value
A     easypod.cloud     YOUR_SERVER_IP
A     *.easypod.cloud   YOUR_SERVER_IP
```

## Production Recommendations

1. **Use Docker volumes** for persistent storage (already configured)
2. **Enable log rotation** (already configured - 10MB max, 3 files)
3. **Monitor logs** regularly for issues
4. **Backup certificates** from the `tunnel-certs` volume
5. **Set up monitoring** (health check is configured)

## Security

- Let's Encrypt automatically handles SSL certificates
- WebSocket connections are encrypted (WSS)
- All HTTP traffic redirected to HTTPS
- No SSH keys needed anymore

## Support

For issues, check:
1. Docker logs: `docker-compose logs -f`
2. Server health: `curl https://easypod.cloud/health`
3. DNS resolution: `nslookup easypod.cloud`
4. Port accessibility: `nc -zv YOUR_SERVER_IP 443`
