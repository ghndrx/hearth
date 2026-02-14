# Hearth — Custom Domain Setup

**Your instance, your domain.**

Hearth is designed to run on any domain you own. No vendor lock-in, no subdomains of our service.

---

## Quick Setup

### Option 1: Bootstrap with Domain

```bash
curl -sSL https://get.hearth.chat | bash -s -- --domain hearth.gregh.dev
```

Done. SSL is automatic via Let's Encrypt.

### Option 2: Manual Configuration

Set `PUBLIC_URL` in your `.env`:

```bash
PUBLIC_URL=https://hearth.gregh.dev
```

---

## DNS Setup

Point your domain to your server:

```
# A record
hearth.gregh.dev    A    YOUR_SERVER_IP

# Or CNAME if using a proxy
hearth.gregh.dev    CNAME    your-server.example.com
```

---

## SSL Options

### Automatic (Caddy)

The bootstrap script uses Caddy which auto-provisions Let's Encrypt certificates:

```
hearth.gregh.dev {
    reverse_proxy hearth:8080
}
```

### Manual (Nginx)

```nginx
server {
    listen 443 ssl http2;
    server_name hearth.gregh.dev;

    ssl_certificate /etc/letsencrypt/live/hearth.gregh.dev/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/hearth.gregh.dev/privkey.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Manual (Traefik)

```yaml
# docker-compose.yml labels
labels:
  - "traefik.enable=true"
  - "traefik.http.routers.hearth.rule=Host(`hearth.gregh.dev`)"
  - "traefik.http.routers.hearth.tls=true"
  - "traefik.http.routers.hearth.tls.certresolver=letsencrypt"
  - "traefik.http.services.hearth.loadbalancer.server.port=8080"
```

### Cloudflare Tunnel

```bash
cloudflared tunnel create hearth
cloudflared tunnel route dns hearth hearth.gregh.dev
```

```yaml
# ~/.cloudflared/config.yml
tunnel: hearth
credentials-file: /root/.cloudflared/hearth.json

ingress:
  - hostname: hearth.gregh.dev
    service: http://localhost:8080
  - service: http_status:404
```

---

## Configuration Reference

All domain-related settings in `.env`:

```bash
# Your public URL (required)
PUBLIC_URL=https://hearth.gregh.dev

# API URL (defaults to PUBLIC_URL)
API_URL=https://hearth.gregh.dev/api/v1

# WebSocket URL (defaults to wss://PUBLIC_URL/gateway)
WS_URL=wss://hearth.gregh.dev/gateway

# CDN URL for assets (optional, defaults to PUBLIC_URL)
CDN_URL=https://hearth.gregh.dev

# Allowed CORS origins (comma-separated, defaults to PUBLIC_URL)
CORS_ORIGINS=https://hearth.gregh.dev

# Cookie domain (for auth cookies)
COOKIE_DOMAIN=hearth.gregh.dev
```

---

## OAuth Redirect URIs

When using external auth (FusionAuth, Google, etc.), add these redirect URIs:

```
https://hearth.gregh.dev/auth/callback
https://hearth.gregh.dev/api/v1/auth/oauth/{provider}/callback
```

---

## Multi-Domain / White-Label

Want to run one Hearth instance with multiple domains?

```bash
# .env
PUBLIC_URL=https://hearth.gregh.dev
ADDITIONAL_DOMAINS=chat.company.com,team.startup.io

# CORS will accept all listed domains
CORS_ORIGINS=https://hearth.gregh.dev,https://chat.company.com,https://team.startup.io
```

Caddy config for multiple domains:

```
hearth.gregh.dev, chat.company.com, team.startup.io {
    reverse_proxy hearth:8080
}
```

---

## Subdomains

You can use subdomains for different services:

| Subdomain | Purpose |
|-----------|---------|
| `hearth.gregh.dev` | Main app |
| `api.hearth.gregh.dev` | API (optional, can be same) |
| `cdn.hearth.gregh.dev` | Static assets / uploads |
| `voice.hearth.gregh.dev` | Voice/video (WebRTC) |

Example config:

```bash
PUBLIC_URL=https://hearth.gregh.dev
API_URL=https://api.hearth.gregh.dev
CDN_URL=https://cdn.hearth.gregh.dev
VOICE_URL=https://voice.hearth.gregh.dev
```

---

## Kubernetes Ingress

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: hearth
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
    - hosts:
        - hearth.gregh.dev
      secretName: hearth-tls
  rules:
    - host: hearth.gregh.dev
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: hearth
                port:
                  number: 8080
```

---

## Troubleshooting

### WebSocket Connection Failed

Check that your proxy supports WebSocket upgrades:

```nginx
proxy_http_version 1.1;
proxy_set_header Upgrade $http_upgrade;
proxy_set_header Connection "upgrade";
```

### OAuth Redirect Mismatch

Ensure `PUBLIC_URL` exactly matches what you registered with your OAuth provider (including https:// and no trailing slash).

### Mixed Content Errors

If you see mixed content warnings, ensure:
- `PUBLIC_URL` uses `https://`
- All asset URLs use HTTPS
- Your proxy sets `X-Forwarded-Proto: https`

### Cookie Issues

For cross-subdomain auth, set:

```bash
COOKIE_DOMAIN=.gregh.dev  # Note the leading dot
```

---

## Examples

### Home Lab

```bash
# Local network only
PUBLIC_URL=http://192.168.1.100:8080
```

### Tailscale

```bash
# Tailscale MagicDNS
PUBLIC_URL=https://hearth.tail1234.ts.net
```

### Custom Domain

```bash
# Your own domain with SSL
PUBLIC_URL=https://chat.mycompany.com
```

---

*Your instance. Your domain. Your data.*
