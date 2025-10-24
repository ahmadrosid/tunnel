# Deployment Guide

This guide explains how to deploy the tunnel server and publish Docker images to GitHub Container Registry.

## Prerequisites

1. GitHub repository: `github.com/ahmadrosid/tunnel`
2. DNS wildcard record configured (e.g., `*.unggahin.com`)
3. Server with Docker installed
4. Ports 22, 80, 443, and 2222 open

## Publishing to GitHub Container Registry (GHCR)

### 1. Enable GitHub Container Registry

The GitHub Actions workflow is already configured in `.github/workflows/docker-publish.yml`.

### 2. Push Code to GitHub

```bash
# Initialize git (if not already done)
git init
git add .
git commit -m "Initial commit: SSH tunneling service"

# Add remote (replace with your repo URL)
git remote add origin git@github.com:ahmadrosid/tunnel.git

# Push to GitHub
git push -u origin main
```

### 3. Automatic Build

Once pushed, GitHub Actions will automatically:
- Build Docker images for amd64 and arm64
- Push to `ghcr.io/ahmadrosid/tunnel:latest`
- Create tagged versions for releases

### 4. View Published Images

Visit: https://github.com/ahmadrosid/tunnel/pkgs/container/tunnel

## Creating a Release

To create a versioned release:

```bash
# Tag the release
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

This will create:
- `ghcr.io/ahmadrosid/tunnel:v1.0.0`
- `ghcr.io/ahmadrosid/tunnel:v1.0`
- `ghcr.io/ahmadrosid/tunnel:v1`
- `ghcr.io/ahmadrosid/tunnel:latest`

## Production Deployment

### Option 1: Docker Compose (Recommended)

```bash
# On your server, create a directory
mkdir -p ~/tunnel-server
cd ~/tunnel-server

# Download docker-compose.prod.yml
wget https://raw.githubusercontent.com/ahmadrosid/tunnel/main/docker-compose.prod.yml

# Create .env file
cat > .env << EOF
DOMAIN=unggahin.com
LETSENCRYPT_EMAIL=your-email@example.com
ENABLE_HTTPS=true
EOF

# Start the server
docker-compose -f docker-compose.prod.yml up -d

# View logs
docker-compose -f docker-compose.prod.yml logs -f
```

### Option 2: Direct Docker Run

```bash
docker run -d \
  --name tunnel-server \
  --restart unless-stopped \
  -p 2222:2222 \
  -p 80:80 \
  -p 443:443 \
  -e DOMAIN=unggahin.com \
  -e LETSENCRYPT_EMAIL=your-email@example.com \
  -e ENABLE_HTTPS=true \
  -v tunnel-keys:/home/tunnel \
  -v tunnel-certs:/home/tunnel/certs \
  ghcr.io/ahmadrosid/tunnel:latest
```

### Option 3: Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tunnel-server
spec:
  replicas: 1
  selector:
    matchLabels:
      app: tunnel-server
  template:
    metadata:
      labels:
        app: tunnel-server
    spec:
      containers:
      - name: tunnel-server
        image: ghcr.io/ahmadrosid/tunnel:latest
        ports:
        - containerPort: 2222
          name: ssh
        - containerPort: 80
          name: http
        - containerPort: 443
          name: https
        env:
        - name: DOMAIN
          value: "unggahin.com"
        - name: LETSENCRYPT_EMAIL
          value: "your-email@example.com"
        - name: ENABLE_HTTPS
          value: "true"
        volumeMounts:
        - name: tunnel-keys
          mountPath: /home/tunnel
        - name: tunnel-certs
          mountPath: /home/tunnel/certs
      volumes:
      - name: tunnel-keys
        persistentVolumeClaim:
          claimName: tunnel-keys-pvc
      - name: tunnel-certs
        persistentVolumeClaim:
          claimName: tunnel-certs-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: tunnel-server
spec:
  type: LoadBalancer
  ports:
  - port: 2222
    targetPort: 2222
    name: ssh
  - port: 80
    targetPort: 80
    name: http
  - port: 443
    targetPort: 443
    name: https
  selector:
    app: tunnel-server
```

## Updating the Deployment

```bash
# Pull latest image
docker pull ghcr.io/ahmadrosid/tunnel:latest

# Restart the container
docker-compose -f docker-compose.prod.yml up -d --force-recreate

# Or with direct docker
docker stop tunnel-server
docker rm tunnel-server
# Run the docker run command again
```

## Monitoring

### Check Status

```bash
# View logs
docker logs -f tunnel-server

# Check container health
docker ps | grep tunnel-server

# View active tunnels (from logs)
docker logs tunnel-server | grep "Tunnel created"
```

### Metrics

```bash
# Container stats
docker stats tunnel-server

# View all active connections
docker exec tunnel-server netstat -an | grep ESTABLISHED
```

## Backup

### Backup SSH Host Keys

```bash
# SSH host keys are stored in a Docker volume
docker run --rm \
  -v tunnel-keys:/source \
  -v $(pwd):/backup \
  alpine tar czf /backup/tunnel-keys-backup.tar.gz -C /source .
```

### Backup SSL Certificates

```bash
# SSL certificates are stored in a Docker volume
docker run --rm \
  -v tunnel-certs:/source \
  -v $(pwd):/backup \
  alpine tar czf /backup/tunnel-certs-backup.tar.gz -C /source .
```

### Restore

```bash
# Restore SSH keys
docker run --rm \
  -v tunnel-keys:/target \
  -v $(pwd):/backup \
  alpine sh -c "cd /target && tar xzf /backup/tunnel-keys-backup.tar.gz"

# Restore certificates
docker run --rm \
  -v tunnel-certs:/target \
  -v $(pwd):/backup \
  alpine sh -c "cd /target && tar xzf /backup/tunnel-certs-backup.tar.gz"
```

## Troubleshooting

### Container won't start

```bash
# Check logs
docker logs tunnel-server

# Common issues:
# - Port 80/443 already in use
# - Invalid domain configuration
# - Permission issues with volumes
```

### HTTPS not working

```bash
# Check Let's Encrypt logs
docker logs tunnel-server | grep -i "acme\|certificate"

# Verify DNS is correct
dig unggahin.com
dig test.unggahin.com

# Check ports are accessible
curl -I http://unggahin.com
```

### High resource usage

```bash
# Check resource limits
docker stats tunnel-server

# Limit resources in docker-compose.yml:
services:
  tunnel-server:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 512M
```

## Security Best Practices

1. **Firewall**: Only open necessary ports (80, 443, 2222)
2. **Monitoring**: Set up log aggregation and monitoring
3. **Updates**: Regularly update to latest image version
4. **Backups**: Regular backups of SSH keys and certificates
5. **Rate Limiting**: Consider adding rate limiting at firewall level
6. **DDoS Protection**: Use Cloudflare or similar for DDoS protection

## Support

For issues and questions:
- GitHub Issues: https://github.com/ahmadrosid/tunnel/issues
- Documentation: https://github.com/ahmadrosid/tunnel#readme
