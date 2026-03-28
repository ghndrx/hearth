#!/bin/bash
#
# Hearth Bootstrap Script
# One command to deploy Hearth with all dependencies
#
# Usage:
#   curl -sSL https://get.hearth.chat | bash
#   curl -sSL https://get.hearth.chat | bash -s -- --with-fusionauth
#   curl -sSL https://get.hearth.chat | bash -s -- --domain chat.example.com
#
# For Nginx instead of Caddy:
#   curl -sSL https://get.hearth.chat/nginx | bash -s -- --domain chat.example.com --email you@example.com
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Defaults
INSTALL_DIR="${HEARTH_DIR:-$HOME/hearth}"
DOMAIN=""
WITH_FUSIONAUTH=false
WITH_SSL=false
COMPOSE_PROFILES=""

# Banner
echo -e "${BLUE}"
cat << 'EOF'
    __  __                __  __  
   / / / /__  ____ ______/ /_/ /_ 
  / /_/ / _ \/ __ `/ ___/ __/ __ \
 / __  /  __/ /_/ / /  / /_/ / / /
/_/ /_/\___/\__,_/_/   \__/_/ /_/ 
                                  
      Self-hosted Discord alternative
EOF
echo -e "${NC}"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --with-fusionauth)
            WITH_FUSIONAUTH=true
            shift
            ;;
        --domain)
            DOMAIN="$2"
            WITH_SSL=true
            shift 2
            ;;
        --dir)
            INSTALL_DIR="$2"
            shift 2
            ;;
        --help|-h)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --with-fusionauth    Include FusionAuth for enterprise SSO"
            echo "  --domain DOMAIN      Set public domain (enables SSL)"
            echo "  --dir PATH           Installation directory (default: ~/hearth)"
            echo "  --help               Show this help"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

# Check requirements
echo -e "${BLUE}Checking requirements...${NC}"

check_command() {
    if ! command -v $1 &> /dev/null; then
        echo -e "${RED}Error: $1 is required but not installed.${NC}"
        echo "Please install $1 and try again."
        exit 1
    fi
}

check_command docker
check_command docker-compose || check_command "docker compose"

# Check Docker is running
if ! docker info &> /dev/null; then
    echo -e "${RED}Error: Docker is not running.${NC}"
    echo "Please start Docker and try again."
    exit 1
fi

echo -e "${GREEN}✓ All requirements met${NC}"

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

if [ "$WITH_FUSIONAUTH" = true ]; then
    FA_DATABASE_PASSWORD=$(openssl rand -base64 24 | tr -d '/+=')
fi

# Create .env file
echo -e "${BLUE}Creating configuration...${NC}"
cat > .env << EOF
# Hearth Configuration
# Generated on $(date)

# Public URL (change this to your domain)
PUBLIC_URL=${DOMAIN:-http://localhost:8080}

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

if [ "$WITH_FUSIONAUTH" = true ]; then
    cat >> .env << EOF

# FusionAuth
FUSIONAUTH_HOST=http://fusionauth:9011
FUSIONAUTH_DATABASE_PASSWORD=${FA_DATABASE_PASSWORD}
EOF
fi

# Create docker-compose.yml
if [ "$WITH_FUSIONAUTH" = true ]; then
    cat > docker-compose.yml << 'EOF'
version: '3.8'

services:
  hearth:
    image: ghcr.io/ghndrx/hearth:latest
    container_name: hearth
    ports:
      - "8080:8080"
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
      - AUTH_PROVIDER=fusionauth
      - FUSIONAUTH_HOST=${FUSIONAUTH_HOST}
    depends_on:
      db:
        condition: service_healthy
      redis:
        condition: service_started
      minio:
        condition: service_started
      fusionauth:
        condition: service_healthy
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

  # FusionAuth
  fusionauth-db:
    image: postgres:16-alpine
    container_name: fusionauth-db
    environment:
      - POSTGRES_USER=fusionauth
      - POSTGRES_PASSWORD=${FUSIONAUTH_DATABASE_PASSWORD}
      - POSTGRES_DB=fusionauth
    volumes:
      - fusionauth-db:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U fusionauth"]
      interval: 5s
      timeout: 5s
      retries: 5
    restart: unless-stopped

  fusionauth:
    image: fusionauth/fusionauth-app:latest
    container_name: fusionauth
    depends_on:
      fusionauth-db:
        condition: service_healthy
    environment:
      - DATABASE_URL=jdbc:postgresql://fusionauth-db:5432/fusionauth
      - DATABASE_ROOT_USERNAME=fusionauth
      - DATABASE_ROOT_PASSWORD=${FUSIONAUTH_DATABASE_PASSWORD}
      - DATABASE_USERNAME=fusionauth
      - DATABASE_PASSWORD=${FUSIONAUTH_DATABASE_PASSWORD}
      - FUSIONAUTH_APP_MEMORY=512M
      - FUSIONAUTH_APP_RUNTIME_MODE=development
      - FUSIONAUTH_APP_URL=http://fusionauth:9011
    ports:
      - "9011:9011"
    volumes:
      - fusionauth-config:/usr/local/fusionauth/config
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9011/api/status"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 60s
    restart: unless-stopped

volumes:
  postgres-data:
  redis-data:
  minio-data:
  fusionauth-db:
  fusionauth-config:
EOF
else
    cat > docker-compose.yml << 'EOF'
version: '3.8'

services:
  hearth:
    image: ghcr.io/ghndrx/hearth:latest
    container_name: hearth
    ports:
      - "8080:8080"
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
fi

# Add SSL/Caddy if domain specified
if [ "$WITH_SSL" = true ] && [ -n "$DOMAIN" ]; then
    echo -e "${BLUE}Configuring SSL for ${DOMAIN}...${NC}"
    
    cat >> docker-compose.yml << EOF

  caddy:
    image: caddy:2-alpine
    container_name: hearth-caddy
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy-data:/data
      - caddy-config:/config
    restart: unless-stopped

volumes:
  caddy-data:
  caddy-config:
EOF

    cat > Caddyfile << EOF
${DOMAIN} {
    reverse_proxy hearth:8080
    
    # WebSocket support
    @websocket {
        header Connection *Upgrade*
        header Upgrade websocket
    }
    reverse_proxy @websocket hearth:8080
}
EOF

    # Update PUBLIC_URL in .env
    sed -i "s|PUBLIC_URL=.*|PUBLIC_URL=https://${DOMAIN}|" .env
fi

# Create MinIO bucket initialization script
cat > init-minio.sh << 'EOF'
#!/bin/bash
# Wait for MinIO to be ready
sleep 10

# Create bucket
docker exec hearth-minio mc alias set local http://localhost:9000 $STORAGE_ACCESS_KEY $STORAGE_SECRET_KEY
docker exec hearth-minio mc mb local/hearth --ignore-existing
docker exec hearth-minio mc anonymous set download local/hearth/avatars
docker exec hearth-minio mc anonymous set download local/hearth/icons

echo "MinIO bucket created and configured"
EOF
chmod +x init-minio.sh

# Start services
echo -e "${BLUE}Starting Hearth...${NC}"
docker compose pull
docker compose up -d

# Wait for services to be healthy
echo -e "${BLUE}Waiting for services to start...${NC}"
sleep 10

# Initialize MinIO bucket
echo -e "${BLUE}Initializing storage...${NC}"
source .env
./init-minio.sh 2>/dev/null || true

# Print success message
echo ""
echo -e "${GREEN}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║                   Hearth is running!                       ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""

if [ -n "$DOMAIN" ]; then
    echo -e "  ${BLUE}Hearth:${NC}      https://${DOMAIN}"
else
    echo -e "  ${BLUE}Hearth:${NC}      http://localhost:8080"
fi

if [ "$WITH_FUSIONAUTH" = true ]; then
    echo -e "  ${BLUE}FusionAuth:${NC}  http://localhost:9011"
    echo ""
    echo -e "  ${YELLOW}Next steps for FusionAuth:${NC}"
    echo "  1. Open http://localhost:9011 and complete setup wizard"
    echo "  2. Create a new Application called 'Hearth'"
    echo "  3. Copy the Application ID and Client Secret"
    echo "  4. Update .env with the FusionAuth credentials"
    echo "  5. Restart: docker compose restart hearth"
fi

echo ""
echo -e "  ${BLUE}Useful commands:${NC}"
echo "  cd ${INSTALL_DIR}"
echo "  docker compose logs -f      # View logs"
echo "  docker compose restart      # Restart services"
echo "  docker compose down         # Stop services"
echo "  docker compose pull && up -d  # Update"
echo ""
echo -e "  ${BLUE}Configuration:${NC} ${INSTALL_DIR}/.env"
echo ""
echo -e "${GREEN}Enjoy your self-hosted chat! 🔥${NC}"
