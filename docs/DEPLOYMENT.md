# Hearth — Deployment Options

**Version:** 1.0  
**Last Updated:** 2026-02-11

---

## Overview

Hearth supports multiple deployment methods, from simple single-binary installs to full Kubernetes clusters. Choose based on your needs:

| Method | Best For | Complexity |
|--------|----------|------------|
| Docker Compose | Most users, homelab | ⭐ Easy |
| Helm Chart | Kubernetes clusters | ⭐⭐ Medium |
| Systemd Service | Bare metal, VPS | ⭐⭐ Medium |
| Binary | Testing, development | ⭐ Easy |

---

## 1. Docker Compose (Recommended)

The easiest way to run Hearth with all dependencies.

### Quick Start
```bash
# Download compose file
mkdir hearth && cd hearth
curl -O https://raw.githubusercontent.com/ghndrx/hearth/main/deploy/docker-compose/docker-compose.yml
curl -O https://raw.githubusercontent.com/ghndrx/hearth/main/deploy/docker-compose/.env.example
cp .env.example .env

# Generate secret key
echo "SECRET_KEY=$(openssl rand -base64 32)" >> .env

# Start
docker-compose up -d

# View logs
docker-compose logs -f hearth
```

### Minimal (SQLite, Local Storage)
```yaml
# docker-compose.yml
version: '3.8'

services:
  hearth:
    image: ghcr.io/ghndrx/hearth:latest
    container_name: hearth
    ports:
      - "8080:8080"
      - "3478:3478/udp"
    volumes:
      - ./data:/data
    environment:
      - SECRET_KEY=${SECRET_KEY}
      - PUBLIC_URL=https://chat.example.com
    restart: unless-stopped
```

### Full Stack (PostgreSQL, Redis, MinIO)
```yaml
# docker-compose.yml
version: '3.8'

services:
  hearth:
    image: ghcr.io/ghndrx/hearth:latest
    container_name: hearth
    ports:
      - "8080:8080"
      - "3478:3478/udp"
      - "49152-49252:49152-49252/udp"  # WebRTC
    environment:
      - SECRET_KEY=${SECRET_KEY}
      - PUBLIC_URL=${PUBLIC_URL}
      - DATABASE_URL=postgres://hearth:${DB_PASSWORD}@db:5432/hearth
      - REDIS_URL=redis://redis:6379
      - STORAGE_BACKEND=s3
      - STORAGE_ENDPOINT=http://minio:9000
      - STORAGE_BUCKET=hearth
      - STORAGE_ACCESS_KEY=${MINIO_ACCESS_KEY}
      - STORAGE_SECRET_KEY=${MINIO_SECRET_KEY}
    depends_on:
      db:
        condition: service_healthy
      redis:
        condition: service_started
      minio:
        condition: service_started
    restart: unless-stopped

  db:
    image: postgres:16-alpine
    container_name: hearth-db
    environment:
      - POSTGRES_USER=hearth
      - POSTGRES_PASSWORD=${DB_PASSWORD}
      - POSTGRES_DB=hearth
    volumes:
      - postgres-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U hearth"]
      interval: 5s
      timeout: 5s
      retries: 5
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    container_name: hearth-redis
    command: redis-server --appendonly yes
    volumes:
      - redis-data:/data
    restart: unless-stopped

  minio:
    image: minio/minio
    container_name: hearth-minio
    command: server /data --console-address ":9001"
    environment:
      - MINIO_ROOT_USER=${MINIO_ACCESS_KEY}
      - MINIO_ROOT_PASSWORD=${MINIO_SECRET_KEY}
    volumes:
      - minio-data:/data
    restart: unless-stopped

  # Optional: Reverse proxy with auto-SSL
  caddy:
    image: caddy:2-alpine
    container_name: hearth-caddy
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile
      - caddy-data:/data
      - caddy-config:/config
    restart: unless-stopped

volumes:
  postgres-data:
  redis-data:
  minio-data:
  caddy-data:
  caddy-config:
```

### Environment File
```bash
# .env
SECRET_KEY=your-32-byte-secret-key-here
PUBLIC_URL=https://chat.example.com
DB_PASSWORD=secure-database-password
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
```

### Caddyfile (Auto-SSL)
```
chat.example.com {
    reverse_proxy hearth:8080
}
```

---

## 2. Helm Chart (Kubernetes)

Full Kubernetes deployment with Helm.

### Prerequisites
- Kubernetes 1.25+
- Helm 3.x
- kubectl configured

### Quick Install
```bash
# Add Hearth repo
helm repo add hearth https://ghndrx.github.io/hearth
helm repo update

# Install with defaults (SQLite, ephemeral)
helm install hearth hearth/hearth

# Install with PostgreSQL and persistence
helm install hearth hearth/hearth \
  --set postgresql.enabled=true \
  --set persistence.enabled=true \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=chat.example.com
```

### values.yaml (Full Options)
```yaml
# Hearth Helm Chart values

replicaCount: 1

image:
  repository: ghcr.io/ghndrx/hearth
  tag: latest
  pullPolicy: IfNotPresent

# Application config
config:
  publicUrl: https://chat.example.com
  logLevel: info
  registration:
    enabled: true
    inviteOnly: false

# Database
database:
  # Use embedded SQLite (not recommended for production)
  sqlite:
    enabled: false
  # Use external PostgreSQL
  external:
    enabled: false
    host: postgres.example.com
    port: 5432
    database: hearth
    username: hearth
    existingSecret: hearth-db-secret
    secretKey: password

# Deploy PostgreSQL subchart
postgresql:
  enabled: true
  auth:
    username: hearth
    database: hearth
    existingSecret: hearth-postgresql
  primary:
    persistence:
      enabled: true
      size: 10Gi

# Redis
redis:
  enabled: true
  architecture: standalone
  auth:
    enabled: false

# Storage
storage:
  # Local persistent volume
  local:
    enabled: true
    size: 50Gi
    storageClass: ""
  # S3-compatible storage
  s3:
    enabled: false
    endpoint: https://s3.amazonaws.com
    bucket: hearth
    region: us-east-1
    existingSecret: hearth-s3-secret

# Ingress
ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/proxy-body-size: 100m
  hosts:
    - host: chat.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: hearth-tls
      hosts:
        - chat.example.com

# Voice/Video (WebRTC)
voice:
  enabled: true
  # TURN server
  turn:
    enabled: true
    # Use external TURN (recommended for production)
    external:
      enabled: false
      servers:
        - url: turn:turn.example.com:3478
          username: hearth
          credential: secret

# Resources
resources:
  requests:
    cpu: 100m
    memory: 256Mi
  limits:
    cpu: 1000m
    memory: 1Gi

# Autoscaling (for large deployments)
autoscaling:
  enabled: false
  minReplicas: 2
  maxReplicas: 10
  targetCPUUtilizationPercentage: 80

# Pod security
securityContext:
  runAsNonRoot: true
  runAsUser: 1000
  fsGroup: 1000

# Probes
livenessProbe:
  httpGet:
    path: /health
    port: http
  initialDelaySeconds: 10
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /health
    port: http
  initialDelaySeconds: 5
  periodSeconds: 5
```

### Commands
```bash
# Upgrade
helm upgrade hearth hearth/hearth -f values.yaml

# Rollback
helm rollback hearth 1

# Uninstall
helm uninstall hearth

# Get status
helm status hearth
kubectl get pods -l app.kubernetes.io/name=hearth
```

---

## 3. Systemd Service (Bare Metal)

Run Hearth as a native systemd service.

### Prerequisites
- Linux with systemd
- PostgreSQL (optional, can use SQLite)
- Redis (optional)

### Installation Script
```bash
#!/bin/bash
# install-hearth.sh

set -e

HEARTH_VERSION="latest"
HEARTH_USER="hearth"
HEARTH_HOME="/opt/hearth"
HEARTH_DATA="/var/lib/hearth"

# Create user
sudo useradd -r -s /bin/false -d $HEARTH_HOME $HEARTH_USER || true

# Create directories
sudo mkdir -p $HEARTH_HOME $HEARTH_DATA/{uploads,db}
sudo chown -R $HEARTH_USER:$HEARTH_USER $HEARTH_HOME $HEARTH_DATA

# Download binary
curl -L "https://github.com/ghndrx/hearth/releases/download/${HEARTH_VERSION}/hearth-linux-amd64" \
  -o /tmp/hearth
sudo mv /tmp/hearth $HEARTH_HOME/hearth
sudo chmod +x $HEARTH_HOME/hearth

# Create config
sudo tee $HEARTH_HOME/config.yaml > /dev/null <<EOF
server:
  port: 8080
  public_url: https://chat.example.com

database:
  url: sqlite://${HEARTH_DATA}/db/hearth.db

storage:
  backend: local
  path: ${HEARTH_DATA}/uploads

auth:
  secret_key: $(openssl rand -base64 32)
EOF

sudo chown $HEARTH_USER:$HEARTH_USER $HEARTH_HOME/config.yaml
sudo chmod 600 $HEARTH_HOME/config.yaml

# Create systemd service
sudo tee /etc/systemd/system/hearth.service > /dev/null <<EOF
[Unit]
Description=Hearth - Self-hosted Discord alternative
Documentation=https://github.com/ghndrx/hearth
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=$HEARTH_USER
Group=$HEARTH_USER
WorkingDirectory=$HEARTH_HOME
ExecStart=$HEARTH_HOME/hearth serve --config $HEARTH_HOME/config.yaml
Restart=always
RestartSec=5

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$HEARTH_DATA
PrivateTmp=true
PrivateDevices=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true

# Resource limits
LimitNOFILE=65535
MemoryMax=2G

[Install]
WantedBy=multi-user.target
EOF

# Enable and start
sudo systemctl daemon-reload
sudo systemctl enable hearth
sudo systemctl start hearth

echo "Hearth installed! Check status with: sudo systemctl status hearth"
```

### Service Commands
```bash
# Start
sudo systemctl start hearth

# Stop
sudo systemctl stop hearth

# Restart
sudo systemctl restart hearth

# Status
sudo systemctl status hearth

# Logs
sudo journalctl -u hearth -f

# Enable on boot
sudo systemctl enable hearth

# Disable on boot
sudo systemctl disable hearth
```

### Config File (/opt/hearth/config.yaml)
```yaml
server:
  host: 0.0.0.0
  port: 8080
  public_url: https://chat.example.com

database:
  # SQLite (simple)
  url: sqlite:///var/lib/hearth/db/hearth.db
  
  # PostgreSQL (recommended for production)
  # url: postgres://hearth:password@localhost:5432/hearth?sslmode=require

redis:
  # Optional, for caching and pub/sub
  url: redis://localhost:6379

storage:
  backend: local
  path: /var/lib/hearth/uploads
  max_upload_size_mb: 25
  
  # S3 alternative
  # backend: s3
  # endpoint: https://s3.amazonaws.com
  # bucket: hearth-uploads
  # access_key: AKIAIOSFODNN7EXAMPLE
  # secret_key: wJalrXUtnFEMI/K7MDENG

auth:
  secret_key: ${SECRET_KEY}  # Set via environment
  registration_enabled: true
  invite_only: false
  mfa_required_for_admins: false

voice:
  enabled: true
  turn:
    enabled: true
    port: 3478
```

### Uninstall
```bash
sudo systemctl stop hearth
sudo systemctl disable hearth
sudo rm /etc/systemd/system/hearth.service
sudo systemctl daemon-reload
sudo userdel hearth
sudo rm -rf /opt/hearth /var/lib/hearth
```

---

## 4. Binary (Manual)

For testing or custom setups.

### Download
```bash
# Linux amd64
curl -LO https://github.com/ghndrx/hearth/releases/latest/download/hearth-linux-amd64
chmod +x hearth-linux-amd64

# Linux arm64
curl -LO https://github.com/ghndrx/hearth/releases/latest/download/hearth-linux-arm64

# macOS
curl -LO https://github.com/ghndrx/hearth/releases/latest/download/hearth-darwin-amd64

# Windows
curl -LO https://github.com/ghndrx/hearth/releases/latest/download/hearth-windows-amd64.exe
```

### Run
```bash
# With environment variables
SECRET_KEY=your-secret-key \
PUBLIC_URL=http://localhost:8080 \
./hearth-linux-amd64 serve

# With config file
./hearth-linux-amd64 serve --config config.yaml
```

---

## Storage Backend Options

### Local Filesystem
```yaml
storage:
  backend: local
  path: /data/uploads
```

### AWS S3
```yaml
storage:
  backend: s3
  endpoint: https://s3.amazonaws.com
  bucket: my-hearth-bucket
  region: us-east-1
  access_key: AKIAIOSFODNN7EXAMPLE
  secret_key: wJalrXUtnFEMI/K7MDENG
```

### MinIO (Self-hosted S3)
```yaml
storage:
  backend: s3
  endpoint: http://minio:9000
  bucket: hearth
  access_key: minioadmin
  secret_key: minioadmin
  use_path_style: true  # Required for MinIO
```

### Backblaze B2
```yaml
storage:
  backend: s3
  endpoint: https://s3.us-west-000.backblazeb2.com
  bucket: my-hearth-bucket
  region: us-west-000
  access_key: your-key-id
  secret_key: your-application-key
```

### Cloudflare R2
```yaml
storage:
  backend: s3
  endpoint: https://account-id.r2.cloudflarestorage.com
  bucket: hearth
  access_key: your-access-key
  secret_key: your-secret-key
```

### Wasabi
```yaml
storage:
  backend: s3
  endpoint: https://s3.wasabisys.com
  bucket: hearth
  region: us-east-1
  access_key: your-access-key
  secret_key: your-secret-key
```

---

## Database Backend Options

### SQLite (Default)
Best for: Small instances, testing, single-server deployments
```yaml
database:
  url: sqlite:///data/hearth.db
```

### PostgreSQL (Recommended)
Best for: Production, multi-instance, high traffic
```yaml
database:
  url: postgres://user:pass@host:5432/hearth?sslmode=require
```

### CockroachDB
Best for: Distributed, global deployments
```yaml
database:
  url: postgres://user:pass@cockroach:26257/hearth?sslmode=verify-full
```

### MySQL/MariaDB
Best for: Existing MySQL infrastructure
```yaml
database:
  url: mysql://user:pass@host:3306/hearth
```

---

## Backup Strategies

### SQLite Backup
```bash
# Simple copy (stop service first)
sudo systemctl stop hearth
cp /var/lib/hearth/db/hearth.db /backup/hearth-$(date +%Y%m%d).db
sudo systemctl start hearth

# Online backup with sqlite3
sqlite3 /var/lib/hearth/db/hearth.db ".backup /backup/hearth-$(date +%Y%m%d).db"
```

### PostgreSQL Backup
```bash
# Full dump
pg_dump -h localhost -U hearth hearth > /backup/hearth-$(date +%Y%m%d).sql

# Compressed
pg_dump -h localhost -U hearth hearth | gzip > /backup/hearth-$(date +%Y%m%d).sql.gz

# Restore
psql -h localhost -U hearth hearth < backup.sql
```

### Media Backup
```bash
# Local storage
rsync -av /var/lib/hearth/uploads/ /backup/uploads/

# S3 to local
aws s3 sync s3://hearth-bucket /backup/uploads/

# S3 to S3 (cross-region)
aws s3 sync s3://hearth-bucket s3://hearth-backup-bucket
```

### Automated Backup (Cron)
```bash
# /etc/cron.d/hearth-backup
0 3 * * * root /opt/hearth/backup.sh >> /var/log/hearth-backup.log 2>&1
```

```bash
#!/bin/bash
# /opt/hearth/backup.sh
BACKUP_DIR=/backup/hearth/$(date +%Y%m%d)
mkdir -p $BACKUP_DIR

# Database
sqlite3 /var/lib/hearth/db/hearth.db ".backup $BACKUP_DIR/hearth.db"

# Uploads
rsync -a /var/lib/hearth/uploads/ $BACKUP_DIR/uploads/

# Cleanup old backups (keep 7 days)
find /backup/hearth -maxdepth 1 -type d -mtime +7 -exec rm -rf {} \;

echo "Backup completed: $BACKUP_DIR"
```

---

## Quick Reference

| Task | Docker Compose | Helm | Systemd |
|------|---------------|------|---------|
| Start | `docker-compose up -d` | `helm install hearth hearth/hearth` | `systemctl start hearth` |
| Stop | `docker-compose down` | `helm uninstall hearth` | `systemctl stop hearth` |
| Logs | `docker-compose logs -f` | `kubectl logs -f deploy/hearth` | `journalctl -u hearth -f` |
| Upgrade | `docker-compose pull && up -d` | `helm upgrade hearth hearth/hearth` | Download new binary, restart |
| Backup | Volume copy | PVC snapshot | File copy |

---

*End of Deployment Guide*
