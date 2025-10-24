# Let's Encrypt SSL/TLS Configuration Guide

This guide explains how SSL/TLS certificates work in the tunnel server and how to configure Let's Encrypt properly.

---

## How It Works

The tunnel server uses `golang.org/x/crypto/acme/autocert` to automatically obtain and renew Let's Encrypt certificates.

### Certificate Acquisition Flow

```
1. User visits https://testapp999.easypod.cloud
2. TLS handshake initiated
3. Server requests certificate from Let's Encrypt
4. Let's Encrypt performs HTTP-01 challenge
   └─ Requests: http://testapp999.easypod.cloud/.well-known/acme-challenge/{token}
5. Server responds with challenge proof
6. Let's Encrypt validates and issues certificate
7. Certificate cached to /home/tunnel/certs
8. Future requests use cached certificate
```

---

## Current Setup: Individual Subdomain Certificates

### What You Have Now

✅ **Automatic certificate issuance** for each subdomain
✅ **HTTP-01 challenge** via port 80
✅ **Certificate caching** in persistent volume
✅ **Auto-renewal** before expiration

### Limitations

❌ **No wildcard certificate** (`*.easypod.cloud` not supported with HTTP-01)
⚠️ **First-visit delay** (2-5 seconds while cert is issued)
⚠️ **Rate limits** (50 certificates per domain per week)

---

## Configuration

### Environment Variables

```bash
# Required
DOMAIN=easypod.cloud

# Recommended - for renewal notifications
LETSENCRYPT_EMAIL=your-email@example.com

# Certificate cache directory
CERT_CACHE_DIR=/home/tunnel/certs

# Ports (must be 80/443 for Let's Encrypt)
HTTP_PORT=80
HTTPS_PORT=443
ENABLE_HTTPS=true
```

### Docker Compose Configuration

```yaml
services:
  tunnel-server:
    ports:
      - "80:80"     # Required for HTTP-01 challenge
      - "443:443"   # HTTPS traffic
    environment:
      - DOMAIN=easypod.cloud
      - LETSENCRYPT_EMAIL=admin@easypod.cloud
      - ENABLE_HTTPS=true
    volumes:
      - tunnel-certs:/home/tunnel/certs  # Persist certificates
```

---

## DNS Requirements

### Required DNS Records

For domain `easypod.cloud`:

```
# A Record - Point to your server IP
easypod.cloud          A     1.2.3.4

# Wildcard subdomain - Point to same IP
*.easypod.cloud        A     1.2.3.4
```

### Verification

```bash
# Check base domain
dig easypod.cloud +short
# Should show: 1.2.3.4

# Check subdomain
dig testapp999.easypod.cloud +short
# Should show: 1.2.3.4
```

---

## Understanding the Errors

### 1. `TLS handshake error from [::1]:38947: EOF`

**Cause:** Client disconnected during TLS handshake, usually because:
- Certificate not yet issued
- Client doesn't trust Let's Encrypt root CA
- Network timeout during certificate fetch

**Solution:**
- Wait for first certificate issuance (2-5 seconds)
- Ensure client trusts Let's Encrypt certificates
- Check server logs for certificate issuance messages

### 2. `Subdomain not found: testapp999`

**Cause:** HTTPS request received for subdomain without active tunnel

**When this happens:**
- User visits `https://testapp999.easypod.cloud`
- No tunnel client is connected for `testapp999`
- Server returns 404 error

**Solution:**
- Ensure tunnel client is connected BEFORE accessing the URL
- Check client logs: `node client.js testapp999 3000`

---

## Production Deployment Checklist

### Before Going Live

- [ ] DNS records configured (A record + wildcard)
- [ ] Ports 80 and 443 open in firewall
- [ ] DOMAIN environment variable set correctly
- [ ] LETSENCRYPT_EMAIL configured for notifications
- [ ] Certificate volume mounted (persistent storage)
- [ ] Server accessible from internet (not behind NAT)

### DNS Propagation

```bash
# Wait for DNS to propagate (can take 5-60 minutes)
watch dig easypod.cloud +short

# Test from multiple locations
dig @8.8.8.8 easypod.cloud +short  # Google DNS
dig @1.1.1.1 easypod.cloud +short  # Cloudflare DNS
```

### First Certificate Test

```bash
# Start server
docker-compose -f docker-compose.prod.yml up -d

# Start tunnel client
node client.js testapp999 3000

# Test from browser or curl
curl -v https://testapp999.easypod.cloud

# Check certificate
openssl s_client -connect testapp999.easypod.cloud:443 -servername testapp999.easypod.cloud < /dev/null
```

---

## Rate Limits

Let's Encrypt has the following rate limits:

| Limit | Amount | Period |
|-------|--------|--------|
| Certificates per domain | 50 | 1 week |
| Duplicate certificates | 5 | 1 week |
| Failed validations | 5 | 1 hour |

### Avoiding Rate Limits

1. **Use staging environment for testing:**
   ```go
   // In cert/manager.go (for testing only)
   m := &autocert.Manager{
       Client: &acme.Client{
           DirectoryURL: "https://acme-staging-v02.api.letsencrypt.org/directory",
       },
       // ...
   }
   ```

2. **Cache certificates persistently** (already configured)

3. **Don't delete certificate cache** unless necessary

---

## Troubleshooting

### Certificate Not Issued

**Check logs:**
```bash
docker-compose -f docker-compose.prod.yml logs -f | grep -i cert
```

**Common issues:**
- Port 80 not accessible from internet
- DNS not pointing to server
- Firewall blocking HTTP
- Server behind NAT without port forwarding

**Verify HTTP-01 challenge works:**
```bash
# From external machine
curl http://easypod.cloud/.well-known/acme-challenge/test
# Should reach your server (even if 404)
```

### Certificate Expired

Certificates auto-renew automatically 30 days before expiration. If renewal fails:

```bash
# Check certificate expiration
openssl s_client -connect testapp999.easypod.cloud:443 -servername testapp999.easypod.cloud 2>/dev/null | openssl x509 -noout -dates

# Force renewal by deleting cache
docker-compose -f docker-compose.prod.yml down
docker volume rm tunnel_tunnel-certs
docker-compose -f docker-compose.prod.yml up -d
```

### Mixed Content Warnings

If local service uses HTTP but tunnel uses HTTPS:

**Option 1:** Use HTTPS on local service
```bash
# Local service should use HTTPS
http://localhost:3000  ❌
https://localhost:3000 ✅
```

**Option 2:** Update local service to support both

---

## Advanced: Wildcard Certificates (DNS-01)

If you need wildcard certificates (`*.easypod.cloud`), you must use DNS-01 challenge instead of HTTP-01.

### Requirements

- DNS provider API (Cloudflare, Route53, etc.)
- Different certificate manager (not autocert)
- Manual certificate renewal setup

### Implementation

This requires significant code changes. Consider using:

1. **certbot** with DNS plugin (external to Go app)
2. **acme/lego** Go library with DNS provider
3. **External certificate management** (mount certs into container)

**Not recommended** unless you have 100+ subdomains. Individual certs work fine for most use cases.

---

## Security Best Practices

1. ✅ **Always set LETSENCRYPT_EMAIL** for renewal notifications
2. ✅ **Use persistent volume** for certificate cache
3. ✅ **Monitor certificate expiration** (auto-renewal should work, but monitor)
4. ✅ **Keep logs** to track certificate issuance
5. ✅ **Use HTTP/1.1** for hijacking (already configured)

---

## Monitoring

### Certificate Expiration Monitoring

Add to your monitoring system:

```bash
# Check expiration date
openssl s_client -connect testapp999.easypod.cloud:443 -servername testapp999.easypod.cloud 2>/dev/null \
  | openssl x509 -noout -dates

# Alert if < 7 days remaining
```

### Let's Encrypt Status

Check Let's Encrypt status: https://letsencrypt.status.io/

---

## Summary

Your current setup is **production-ready** for individual subdomain certificates:

✅ Automatic issuance
✅ Auto-renewal
✅ HTTP-01 challenge
✅ Certificate caching

The "TLS handshake" and "Subdomain not found" errors are **normal** and expected:
- TLS errors happen during first cert issuance
- Subdomain errors happen when tunnel not registered

**No further action needed** unless you need wildcard certificates!
