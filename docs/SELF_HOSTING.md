# Hearth — Self-Hosting Guide

**Version:** 1.0  
**Last Updated:** 2026-02-11

---

## Requirements

### Minimum
- **CPU:** 1 core
- **RAM:** 512 MB
- **Storage:** 1 GB + media storage
- **OS:** Linux (amd64 or arm64), Windows, macOS

### Recommended
- **CPU:** 2+ cores
- **RAM:** 2 GB+
- **Storage:** SSD, 10 GB + media
- **Network:** 100 Mbps+

### For Voice/Video
- **RAM:** 4 GB+ (WebRTC SFU is memory-intensive)
- **Network:** 1 Gbps recommended for many concurrent users
- **Ports:** UDP 3478 (TURN), UDP 49152-65535 (RTP)

---

## Quick Start (Docker)

### 1. Create data directory
```bash
mkdir -p ~/hearth/data
cd ~/hearth
```

### 2. Generate secret key
```bash
openssl rand -base64 32 > .secret_key
```

### 3. Create docker-compose.yml
```yaml
version: '3.8'

services:
  hearth:
    image: ghcr.io/ghndrx/hearth:latest
    container_name: hearth
    ports:
      - "8080:8080"      # HTTP
      - "3478:3478/udp"  # TURN
    volumes:
      - ./data:/data
    environment:
      - SECRET_KEY_FILE=/run/secrets/secret_key
      - DATABASE_URL=sqlite:///data/hearth.db
      - STORAGE_PATH=/data/uploads
      - PUBLIC_URL=https://chat.yourdomain.com
    secrets:
      - secret_key
    restart: unless-stopped

secrets:
  secret_key:
    file: .secret_key
```

### 4. Start Hearth
```bash
docker-compose up -d
```

### 5. Access
Open `http://localhost:8080` and create your first account.

---

## Production Deployment

### With PostgreSQL
```yaml
version: '3.8'

services:
  hearth:
    image: ghcr.io/ghndrx/hearth:latest
    environment:
      - DATABASE_URL=postgres://hearth:${DB_PASSWORD}@db:5432/hearth
      - REDIS_URL=redis://redis:6379
      - STORAGE_URL=s3://minio:9000/hearth
      - STORAGE_ACCESS_KEY=${MINIO_ACCESS_KEY}
      - STORAGE_SECRET_KEY=${MINIO_SECRET_KEY}
      - PUBLIC_URL=https://chat.yourdomain.com
      - SECRET_KEY=${SECRET_KEY}
    depends_on:
      - db
      - redis
      - minio
    restart: unless-stopped

  db:
    image: postgres:16-alpine
    environment:
      - POSTGRES_USER=hearth
      - POSTGRES_PASSWORD=${DB_PASSWORD}
      - POSTGRES_DB=hearth
    volumes:
      - postgres-data:/var/lib/postgresql/data
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    volumes:
      - redis-data:/data
    restart: unless-stopped

  minio:
    image: minio/minio
    command: server /data --console-address ":9001"
    environment:
      - MINIO_ROOT_USER=${MINIO_ACCESS_KEY}
      - MINIO_ROOT_PASSWORD=${MINIO_SECRET_KEY}
    volumes:
      - minio-data:/data
    restart: unless-stopped

volumes:
  postgres-data:
  redis-data:
  minio-data:
```

### Reverse Proxy (Caddy)
```
# Caddyfile
chat.yourdomain.com {
    reverse_proxy hearth:8080
}
```

### Reverse Proxy (Nginx)
```nginx
server {
    listen 443 ssl http2;
    server_name chat.yourdomain.com;

    ssl_certificate /etc/ssl/certs/chat.pem;
    ssl_certificate_key /etc/ssl/private/chat.key;

    location / {
        proxy_pass http://hearth:8080;
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

---

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SECRET_KEY` | (required) | 32-byte secret for JWT signing |
| `DATABASE_URL` | sqlite:///data/hearth.db | Database connection string |
| `REDIS_URL` | (none) | Redis connection for caching/pubsub |
| `STORAGE_PATH` | /data/uploads | Local file storage path |
| `STORAGE_URL` | (none) | S3-compatible storage URL |
| `PUBLIC_URL` | http://localhost:8080 | Public URL for links/embeds |
| `PORT` | 8080 | HTTP server port |
| `LOG_LEVEL` | info | debug, info, warn, error |
| `LOG_FORMAT` | json | json, text |
| `SMTP_HOST` | (none) | SMTP server for emails |
| `SMTP_PORT` | 587 | SMTP port |
| `SMTP_USER` | (none) | SMTP username |
| `SMTP_PASS` | (none) | SMTP password |
| `SMTP_FROM` | noreply@example.com | From address |
| `REGISTRATION_ENABLED` | true | Allow new registrations |
| `INVITE_ONLY` | false | Require invite to register |
| `MAX_SERVERS_PER_USER` | 100 | Server creation limit |
| `MAX_UPLOAD_SIZE_MB` | 8 | File upload limit |
| `TURN_ENABLED` | true | Enable TURN server |
| `TURN_SECRET` | (auto) | TURN auth secret |

### Config File (Alternative)
```yaml
# config.yaml
server:
  port: 8080
  public_url: https://chat.yourdomain.com

database:
  url: postgres://hearth:pass@localhost:5432/hearth
  max_connections: 25

redis:
  url: redis://localhost:6379

storage:
  type: s3
  endpoint: https://s3.amazonaws.com
  bucket: hearth-uploads
  region: us-east-1

auth:
  registration_enabled: true
  invite_only: false
  mfa_required_for_admins: true

limits:
  max_servers_per_user: 100
  max_upload_size_mb: 25
  message_rate_limit: 5/5s
```

---

## Database

### SQLite (Default)
Good for small instances (<100 concurrent users).
```
DATABASE_URL=sqlite:///data/hearth.db
```

### PostgreSQL (Recommended)
Better for larger deployments.
```
DATABASE_URL=postgres://user:pass@host:5432/hearth?sslmode=require
```

### Migrations
Migrations run automatically on startup. To run manually:
```bash
docker exec hearth /app/hearth migrate up
```

---

## Storage

### Local Filesystem
```
STORAGE_PATH=/data/uploads
```

### S3-Compatible (MinIO, AWS S3, Backblaze B2)
```
STORAGE_URL=s3://s3.amazonaws.com/my-bucket
STORAGE_ACCESS_KEY=AKIAIOSFODNN7EXAMPLE
STORAGE_SECRET_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
STORAGE_REGION=us-east-1
```

---

## Voice/Video (WebRTC)

### TURN Server
Hearth includes a built-in TURN server for NAT traversal.

Required ports:
- UDP 3478 (STUN/TURN)
- UDP 49152-65535 (RTP media)

### External TURN
For high-scale deployments, use coturn or Twilio:
```
TURN_ENABLED=false
TURN_SERVERS=turn:turn.example.com:3478
TURN_USERNAME=hearth
TURN_CREDENTIAL=secret
```

---

## Backup & Restore

### Database Backup (PostgreSQL)
```bash
docker exec hearth-db pg_dump -U hearth hearth > backup.sql
```

### Database Restore
```bash
cat backup.sql | docker exec -i hearth-db psql -U hearth hearth
```

### Full Backup (Docker Volumes)
```bash
docker run --rm \
  -v hearth_postgres-data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/postgres-backup.tar.gz /data
```

### Media Backup
Just copy the uploads directory or sync from S3.

---

## Upgrading

### Docker
```bash
docker-compose pull
docker-compose up -d
```

Migrations run automatically. Check release notes for breaking changes.

### Rollback
```bash
docker-compose down
docker tag ghcr.io/ghndrx/hearth:latest ghcr.io/ghndrx/hearth:rollback
docker pull ghcr.io/ghndrx/hearth:v0.2.0  # previous version
docker-compose up -d
```

---

## Monitoring

### Health Check
```bash
curl http://localhost:8080/health
```

### Prometheus Metrics
```bash
curl http://localhost:8080/metrics
```

### Logs
```bash
docker logs -f hearth
```

---

## Troubleshooting

### Can't connect to voice
- Check UDP ports 3478 and 49152-65535 are open
- Ensure TURN server is reachable
- Check browser console for WebRTC errors

### Messages not sending
- Check WebSocket connection in browser dev tools
- Verify Redis is running (if configured)
- Check server logs for errors

### Uploads failing
- Verify storage path is writable
- Check file size against MAX_UPLOAD_SIZE_MB
- Ensure S3 credentials are correct

### Performance issues
- Enable Redis for caching
- Use PostgreSQL instead of SQLite
- Increase database connection pool
- Check for slow queries in logs

---

*End of Self-Hosting Guide*
