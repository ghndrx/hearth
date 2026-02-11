#!/bin/bash
#
# Hearth Bootstrap Script (Cloudflare Tunnel)
#
# Zero exposed ports. Cloudflare handles SSL, DDoS protection, and routing.
#
# Usage:
#   curl -sSL https://get.hearth.chat/cloudflare | bash -s -- --domain hearth.gregh.dev --tunnel-token <TOKEN>
#
# Get your tunnel token:
#   1. Go to https://one.dash.cloudflare.com
#   2. Zero Trust → Networks → Tunnels
#   3. Create a tunnel, copy the token
#

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

INSTALL_DIR="${HEARTH_DIR:-$HOME/hearth}"
DOMAIN=""
TUNNEL_TOKEN=""
TUNNEL_NAME="hearth"
WITH_FUSIONAUTH=false

echo -e "${BLUE}"
cat << 'EOF'
    __  __                __  __  
   / / / /__  ____ ______/ /_/ /_ 
  / /_/ / _ \/ __ `/ ___/ __/ __ \
 / __  /  __/ /_/ / /  / /_/ / / /
/_/ /_/\___/\__,_/_/   \__/_/ /_/ 
                                  
    Self-hosted (Cloudflare Tunnel)
EOF
echo -e "${NC}"

while [[ $# -gt 0 ]]; do
    case $1 in
        --domain) DOMAIN="$2"; shift 2 ;;
        --tunnel-token) TUNNEL_TOKEN="$2"; shift 2 ;;
        --tunnel-name) TUNNEL_NAME="$2"; shift 2 ;;
        --with-fusionauth) WITH_FUSIONAUTH=true; shift ;;
        --dir) INSTALL_DIR="$2"; shift 2 ;;
        --help|-h)
            echo "Usage: $0 --domain DOMAIN --tunnel-token TOKEN"
            echo ""
            echo "Get your tunnel token from:"
            echo "  https://one.dash.cloudflare.com → Zero Trust → Networks → Tunnels"
            exit 0
            ;;
        *) echo -e "${RED}Unknown option: $1${NC}"; exit 1 ;;
    esac
done

if [ -z "$DOMAIN" ]; then
    echo -e "${RED}Error: --domain is required${NC}"
    exit 1
fi

if [ -z "$TUNNEL_TOKEN" ]; then
    echo -e "${YELLOW}No tunnel token provided.${NC}"
    echo ""
    echo "To get a tunnel token:"
    echo "  1. Go to https://one.dash.cloudflare.com"
    echo "  2. Zero Trust → Networks → Tunnels"
    echo "  3. Create tunnel '${TUNNEL_NAME}'"
    echo "  4. Copy the token"
    echo "  5. Run again with --tunnel-token <TOKEN>"
    echo ""
    read -p "Enter tunnel token (or Ctrl+C to exit): " TUNNEL_TOKEN
    if [ -z "$TUNNEL_TOKEN" ]; then
        exit 1
    fi
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
TUNNEL_TOKEN=${TUNNEL_TOKEN}
EOF

# Create docker-compose.yml
cat > docker-compose.yml << EOF
version: '3.8'

services:
  # Cloudflare Tunnel - No ports exposed to internet!
  cloudflared:
    image: cloudflare/cloudflared:latest
    container_name: cloudflared
    command: tunnel --no-autoupdate run --token \${TUNNEL_TOKEN}
    restart: unless-stopped
    depends_on:
      - hearth

  hearth:
    image: ghcr.io/ghndrx/hearth:latest
    container_name: hearth
    # No ports exposed! Cloudflare tunnel handles routing
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
EOF

echo -e "${BLUE}Starting Hearth with Cloudflare Tunnel...${NC}"
docker compose pull
docker compose up -d

sleep 10

echo ""
echo -e "${GREEN}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║                   Hearth is running!                       ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "  ${BLUE}URL:${NC}           https://${DOMAIN}"
echo -e "  ${BLUE}Tunnel:${NC}        Cloudflare (Zero Trust)"
echo -e "  ${BLUE}SSL:${NC}           Cloudflare Edge (automatic)"
echo -e "  ${BLUE}DDoS:${NC}          Protected by Cloudflare"
echo -e "  ${BLUE}Exposed Ports:${NC} None! 🔒"
echo ""
echo -e "${YELLOW}Next step:${NC}"
echo "  Configure public hostname in Cloudflare dashboard:"
echo "  Zero Trust → Networks → Tunnels → ${TUNNEL_NAME} → Public Hostname"
echo "  Add: ${DOMAIN} → http://hearth:8080"
echo ""
echo -e "${GREEN}Enjoy! 🔥${NC}"
