#!/bin/bash
#
# Hearth Bootstrap Script (Nginx + Let's Encrypt)
#
# Usage:
#   curl -sSL https://get.hearth.chat/nginx | bash
#   curl -sSL https://get.hearth.chat/nginx | bash -s -- --domain hearth.gregh.dev --email admin@gregh.dev
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Defaults
INSTALL_DIR="${HEARTH_DIR:-$HOME/hearth}"
DOMAIN=""
EMAIL=""
WITH_FUSIONAUTH=false

# Banner
echo -e "${BLUE}"
cat << 'EOF'
    __  __                __  __  
   / / / /__  ____ ______/ /_/ /_ 
  / /_/ / _ \/ __ `/ ___/ __/ __ \
 / __  /  __/ /_/ / /  / /_/ / / /
/_/ /_/\___/\__,_/_/   \__/_/ /_/ 
                                  
      Self-hosted (Nginx Edition)
EOF
echo -e "${NC}"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --domain)
            DOMAIN="$2"
            shift 2
            ;;
        --email)
            EMAIL="$2"
            shift 2
            ;;
        --with-fusionauth)
            WITH_FUSIONAUTH=true
            shift
            ;;
        --dir)
            INSTALL_DIR="$2"
            shift 2
            ;;
        --help|-h)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --domain DOMAIN      Your domain (e.g., hearth.gregh.dev)"
            echo "  --email EMAIL        Email for Let's Encrypt (required for SSL)"
            echo "  --with-fusionauth    Include FusionAuth for enterprise SSO"
            echo "  --dir PATH           Installation directory (default: ~/hearth)"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

# Validate
if [ -n "$DOMAIN" ] && [ -z "$EMAIL" ]; then
    echo -e "${YELLOW}Warning: --email recommended for Let's Encrypt notifications${NC}"
    EMAIL="admin@${DOMAIN}"
fi

# Check requirements
echo -e "${BLUE}Checking requirements...${NC}"

check_command() {
    if ! command -v $1 &> /dev/null; then
        echo -e "${RED}Error: $1 is required but not installed.${NC}"
        exit 1
    fi
}

check_command docker
check_command docker-compose || check_command "docker compose"

echo -e "${GREEN}✓ Requirements met${NC}"

# Create installation directory
echo -e "${BLUE}Setting up in ${INSTALL_DIR}...${NC}"
mkdir -p "$INSTALL_DIR"
cd "$INSTALL_DIR"

# Generate secrets
echo -e "${BLUE}Generating secrets...${NC}"
SECRET_KEY=$(openssl rand -base64 32)
DB_PASSWORD=$(openssl rand -base64 24 | tr -d '/+=')
MINIO_ACCESS_KEY=$(openssl rand -hex 16)
MINIO_SECRET_KEY=$(openssl rand -base64 32 | tr -d '/+=')

# Determine public URL
if [ -n "$DOMAIN" ]; then
    PUBLIC_URL="https://${DOMAIN}"
else
    PUBLIC_URL="http://localhost:8080"
fi

# Create .env file
cat > .env << EOF
# Hearth Configuration
# Generated on $(date)

# Public URL
PUBLIC_URL=${PUBLIC_URL}

# Secret key for JWT signing
SECRET_KEY=${SECRET_KEY}

# Database
DATABASE_URL=postgres://hearth:${DB_PASSWORD}@db:5432/hearth
DB_PASSWORD=${DB_PASSWORD}

# Redis
REDIS_URL=redis://redis:6379

# Storage (MinIO S3-compatible)
STORAGE_BACKEND=s3
STORAGE_ENDPOINT=http://minio:9000
STORAGE_BUCKET=hearth
STORAGE_ACCESS_KEY=${MINIO_ACCESS_KEY}
STORAGE_SECRET_KEY=${MINIO_SECRET_KEY}

# Registration
REGISTRATION_ENABLED=true
INVITE_ONLY=false
EOF

# Create docker-compose.yml
cat > docker-compose.yml << 'EOF'
version: '3.8'

services:
  hearth:
    image: ghcr.io/ghndrx/hearth:latest
    container_name: hearth
    ports:
      - "127.0.0.1:8080:8080"
      - "3478:3478/udp"
    environment:
      - SECRET_KEY=${SECRET_KEY}
      - PUBLIC_URL=${PUBLIC_URL}
      - DATABASE_URL=${DATABASE_URL}
      - REDIS_URL=${REDIS_URL}
      - STORAGE_BACKEND=${STORAGE_BACKEND}
      - STORAGE_ENDPOINT=${STORAGE_ENDPOINT}
      - STORAGE_BUCKET=${STORAGE_BUCKET}
      - STORAGE_ACCESS_KEY=${STORAGE_ACCESS_KEY}
      - STORAGE_SECRET_KEY=${STORAGE_SECRET_KEY}
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
      - MINIO_ROOT_USER=${STORAGE_ACCESS_KEY}
      - MINIO_ROOT_PASSWORD=${STORAGE_SECRET_KEY}
    volumes:
      - minio-data:/data
    restart: unless-stopped

volumes:
  postgres-data:
  redis-data:
  minio-data:
EOF

# Setup Nginx + Certbot if domain provided
if [ -n "$DOMAIN" ]; then
    echo -e "${BLUE}Setting up Nginx + Let's Encrypt for ${DOMAIN}...${NC}"
    
    # Install Nginx and Certbot if not present
    if ! command -v nginx &> /dev/null; then
        echo -e "${BLUE}Installing Nginx...${NC}"
        if command -v apt-get &> /dev/null; then
            sudo apt-get update && sudo apt-get install -y nginx certbot python3-certbot-nginx
        elif command -v yum &> /dev/null; then
            sudo yum install -y nginx certbot python3-certbot-nginx
        elif command -v dnf &> /dev/null; then
            sudo dnf install -y nginx certbot python3-certbot-nginx
        else
            echo -e "${RED}Please install nginx and certbot manually${NC}"
            exit 1
        fi
    fi
    
    # Create Nginx config
    sudo tee /etc/nginx/sites-available/hearth << EOF
# Hearth - ${DOMAIN}

# Redirect HTTP to HTTPS
server {
    listen 80;
    listen [::]:80;
    server_name ${DOMAIN};
    
    location /.well-known/acme-challenge/ {
        root /var/www/certbot;
    }
    
    location / {
        return 301 https://\$server_name\$request_uri;
    }
}

# HTTPS
server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name ${DOMAIN};
    
    # SSL (will be configured by certbot)
    ssl_certificate /etc/letsencrypt/live/${DOMAIN}/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/${DOMAIN}/privkey.pem;
    
    # SSL settings
    ssl_session_timeout 1d;
    ssl_session_cache shared:SSL:50m;
    ssl_session_tickets off;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;
    
    # HSTS
    add_header Strict-Transport-Security "max-age=63072000" always;
    
    # Proxy settings
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        
        # WebSocket timeout
        proxy_read_timeout 86400;
    }
    
    # File upload size
    client_max_body_size 100M;
}
EOF

    # Enable site
    sudo ln -sf /etc/nginx/sites-available/hearth /etc/nginx/sites-enabled/
    sudo rm -f /etc/nginx/sites-enabled/default
    
    # Create certbot webroot
    sudo mkdir -p /var/www/certbot
    
    # Test nginx config
    sudo nginx -t
    
    # Get certificate
    echo -e "${BLUE}Obtaining SSL certificate...${NC}"
    sudo certbot certonly --nginx -d "${DOMAIN}" --non-interactive --agree-tos -m "${EMAIL}" || {
        echo -e "${YELLOW}Certbot failed. Starting without SSL first...${NC}"
        # Create temporary self-signed cert
        sudo mkdir -p /etc/letsencrypt/live/${DOMAIN}
        sudo openssl req -x509 -nodes -days 1 -newkey rsa:2048 \
            -keyout /etc/letsencrypt/live/${DOMAIN}/privkey.pem \
            -out /etc/letsencrypt/live/${DOMAIN}/fullchain.pem \
            -subj "/CN=${DOMAIN}"
        echo -e "${YELLOW}Using temporary self-signed cert. Run 'sudo certbot --nginx' after DNS propagates.${NC}"
    }
    
    # Reload nginx
    sudo systemctl reload nginx
    
    # Setup auto-renewal
    echo -e "${BLUE}Setting up certificate auto-renewal...${NC}"
    (crontab -l 2>/dev/null | grep -v certbot; echo "0 3 * * * certbot renew --quiet --post-hook 'systemctl reload nginx'") | crontab -
fi

# Start services
echo -e "${BLUE}Starting Hearth...${NC}"
docker compose pull
docker compose up -d

# Wait for services
sleep 10

# Print success
echo ""
echo -e "${GREEN}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║                   Hearth is running!                       ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""

if [ -n "$DOMAIN" ]; then
    echo -e "  ${BLUE}URL:${NC}         https://${DOMAIN}"
    echo -e "  ${BLUE}SSL:${NC}         Let's Encrypt (auto-renew enabled)"
else
    echo -e "  ${BLUE}URL:${NC}         http://localhost:8080"
fi

echo ""
echo -e "  ${BLUE}Useful commands:${NC}"
echo "  cd ${INSTALL_DIR}"
echo "  docker compose logs -f      # View logs"
echo "  docker compose restart      # Restart services"
echo "  sudo nginx -t               # Test nginx config"
echo "  sudo certbot renew          # Renew SSL cert"
echo ""
echo -e "${GREEN}Enjoy your self-hosted chat! 🔥${NC}"
