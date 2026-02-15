#!/bin/bash
#
# Hearth Bootstrap Script (Traefik + Let's Encrypt)
#
# Usage:
#   curl -sSL https://get.hearth.chat/traefik | bash -s -- --domain hearth.gregh.dev --email admin@gregh.dev
#

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

INSTALL_DIR="${HEARTH_DIR:-$HOME/hearth}"
DOMAIN=""
EMAIL=""
WITH_FUSIONAUTH=false

echo -e "${BLUE}"
cat << 'EOF'
    __  __                __  __  
   / / / /__  ____ ______/ /_/ /_ 
  / /_/ / _ \/ __ `/ ___/ __/ __ \
 / __  /  __/ /_/ / /  / /_/ / / /
/_/ /_/\___/\__,_/_/   \__/_/ /_/ 
                                  
      Self-hosted (Traefik Edition)
EOF
echo -e "${NC}"

while [[ $# -gt 0 ]]; do
    case $1 in
        --domain) DOMAIN="$2"; shift 2 ;;
        --email) EMAIL="$2"; shift 2 ;;
        --with-fusionauth) WITH_FUSIONAUTH=true; shift ;;
        --dir) INSTALL_DIR="$2"; shift 2 ;;
        --help|-h)
            echo "Usage: $0 --domain DOMAIN --email EMAIL [--with-fusionauth]"
            exit 0
            ;;
        *) echo -e "${RED}Unknown option: $1${NC}"; exit 1 ;;
    esac
done

if [ -z "$DOMAIN" ]; then
    echo -e "${RED}Error: --domain is required${NC}"
    exit 1
fi

if [ -z "$EMAIL" ]; then
    EMAIL="admin@${DOMAIN}"
fi

echo -e "${BLUE}Setting up in ${INSTALL_DIR}...${NC}"
mkdir -p "$INSTALL_DIR"
cd "$INSTALL_DIR"

# Generate secrets
SECRET_KEY=$(openssl rand -base64 32)
DB_PASSWORD=$(openssl rand -base64 24 | tr -d '/+=')
MINIO_ACCESS_KEY=$(openssl rand -hex 16)
MINIO_SECRET_KEY=$(openssl rand -base64 32 | tr -d '/+=')

# Create .env
cat > .env << EOF
PUBLIC_URL=https://${DOMAIN}
SECRET_KEY=${SECRET_KEY}
DATABASE_URL=postgres://hearth:${DB_PASSWORD}@db:5432/hearth
DB_PASSWORD=${DB_PASSWORD}
REDIS_URL=redis://redis:6379
STORAGE_BACKEND=s3
STORAGE_ENDPOINT=http://minio:9000
STORAGE_BUCKET=hearth
STORAGE_ACCESS_KEY=${MINIO_ACCESS_KEY}
STORAGE_SECRET_KEY=${MINIO_SECRET_KEY}
ACME_EMAIL=${EMAIL}
DOMAIN=${DOMAIN}
EOF

# Create Traefik config
mkdir -p traefik
cat > traefik/traefik.yml << EOF
api:
  dashboard: true
  insecure: true

entryPoints:
  web:
    address: ":80"
    http:
      redirections:
        entryPoint:
          to: websecure
          scheme: https
  websecure:
    address: ":443"

providers:
  docker:
    endpoint: "unix:///var/run/docker.sock"
    exposedByDefault: false

certificatesResolvers:
  letsencrypt:
    acme:
      email: ${EMAIL}
      storage: /letsencrypt/acme.json
      httpChallenge:
        entryPoint: web
EOF

# Create docker-compose.yml
cat > docker-compose.yml << EOF
version: '3.8'

services:
  traefik:
    image: traefik:v3.0
    container_name: traefik
    ports:
      - "80:80"
      - "443:443"
      - "8081:8080"  # Dashboard
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./traefik/traefik.yml:/etc/traefik/traefik.yml:ro
      - traefik-certs:/letsencrypt
    restart: unless-stopped

  hearth:
    image: ghcr.io/ghndrx/hearth:latest
    container_name: hearth
    environment:
      - SECRET_KEY=\${SECRET_KEY}
      - PUBLIC_URL=\${PUBLIC_URL}
      - DATABASE_URL=\${DATABASE_URL}
      - REDIS_URL=\${REDIS_URL}
      - STORAGE_BACKEND=\${STORAGE_BACKEND}
      - STORAGE_ENDPOINT=\${STORAGE_ENDPOINT}
      - STORAGE_BUCKET=\${STORAGE_BUCKET}
      - STORAGE_ACCESS_KEY=\${STORAGE_ACCESS_KEY}
      - STORAGE_SECRET_KEY=\${STORAGE_SECRET_KEY}
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.hearth.rule=Host(\`${DOMAIN}\`)"
      - "traefik.http.routers.hearth.entrypoints=websecure"
      - "traefik.http.routers.hearth.tls=true"
      - "traefik.http.routers.hearth.tls.certresolver=letsencrypt"
      - "traefik.http.services.hearth.loadbalancer.server.port=8080"
      # WebSocket support
      - "traefik.http.middlewares.hearth-headers.headers.customrequestheaders.Connection=keep-alive, Upgrade"
      - "traefik.http.routers.hearth.middlewares=hearth-headers"
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
      - POSTGRES_PASSWORD=\${DB_PASSWORD}
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
      - MINIO_ROOT_USER=\${STORAGE_ACCESS_KEY}
      - MINIO_ROOT_PASSWORD=\${STORAGE_SECRET_KEY}
    volumes:
      - minio-data:/data
    restart: unless-stopped

volumes:
  postgres-data:
  redis-data:
  minio-data:
  traefik-certs:
EOF

# Start
echo -e "${BLUE}Starting Hearth with Traefik...${NC}"
docker compose pull
docker compose up -d

sleep 10

echo ""
echo -e "${GREEN}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║                   Hearth is running!                       ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "  ${BLUE}Hearth:${NC}      https://${DOMAIN}"
echo -e "  ${BLUE}Traefik:${NC}    http://localhost:8081 (dashboard)"
echo -e "  ${BLUE}SSL:${NC}        Let's Encrypt (auto-managed)"
echo ""
echo -e "${GREEN}Enjoy! 🔥${NC}"
