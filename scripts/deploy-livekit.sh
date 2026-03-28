#!/bin/bash
# deploy-livekit.sh - Deploy LiveKit to Hearth server
# Usage: ./deploy-livekit.sh <server-ip>

set -e

SERVER=${1:-"5.78.74.173"}
SSH_USER=${2:-"root"}
REMOTE_DIR="/root/hearth"

echo "🎙️ Deploying LiveKit to Hearth..."
echo "   Server: $SSH_USER@$SERVER"

# Check SSH connectivity
echo "📡 Testing SSH connection..."
if ! ssh -o ConnectTimeout=10 "$SSH_USER@$SERVER" "echo ok" >/dev/null 2>&1; then
    echo "❌ Cannot connect to $SERVER via SSH"
    echo "   Please ensure:"
    echo "   1. SSH port 22 is open in firewall"
    echo "   2. SSH key is configured"
    exit 1
fi

echo "✅ SSH connection successful"

# Create remote directory
echo "📁 Creating directories..."
ssh "$SSH_USER@$SERVER" "mkdir -p $REMOTE_DIR/livekit"

# Copy LiveKit config files
echo "📤 Uploading LiveKit configuration..."
scp deploy/livekit/livekit.yaml "$SSH_USER@$SERVER:$REMOTE_DIR/livekit/"
scp deploy/livekit/docker-compose.livekit.yml "$SSH_USER@$SERVER:$REMOTE_DIR/livekit/"

# Create Docker network if it doesn't exist
echo "🌐 Setting up Docker network..."
ssh "$SSH_USER@$SERVER" "docker network create hearth-network 2>/dev/null || true"

# Start LiveKit
echo "🚀 Starting LiveKit server..."
ssh "$SSH_USER@$SERVER" "cd $REMOTE_DIR/livekit && docker compose -f docker-compose.livekit.yml pull && docker compose -f docker-compose.livekit.yml up -d"

# Update Caddy config
echo "🔧 Updating Caddy configuration..."
ssh "$SSH_USER@$SERVER" 'cat > /tmp/caddyfile-livekit <<EOF
hearth.gregh.dev {
    handle /livekit* {
        reverse_proxy localhost:7880
    }
    handle /api/* {
        reverse_proxy localhost:8080
    }
    handle /ws {
        reverse_proxy localhost:8080
    }
    handle {
        reverse_proxy localhost:3000
    }
}
EOF
sudo cp /tmp/caddyfile-livekit /etc/caddy/Caddyfile
sudo systemctl reload caddy'

# Update backend environment
echo "🔐 Configuring backend with LiveKit credentials..."
ssh "$SSH_USER@$SERVER" "cd $REMOTE_DIR && \
    echo 'LIVEKIT_API_KEY=APIcKPP67nS9WWXH9mWx3XGc' >> .env && \
    echo 'LIVEKIT_API_SECRET=8fDdP39mJYSBcW4w2hDFP8pUB0WYR0tvbVZ0tFSdHXTzDPap' >> .env && \
    echo 'LIVEKIT_URL=wss://hearth.gregh.dev/livekit' >> .env"

# Restart backend to pick up new config
echo "🔄 Restarting backend..."
ssh "$SSH_USER@$SERVER" "docker restart hearth-backend-1"

# Verify deployment
echo "✅ Verifying deployment..."
sleep 5
ssh "$SSH_USER@$SERVER" "docker ps | grep livekit"

echo ""
echo "🎉 LiveKit deployment complete!"
echo ""
echo "📝 Next steps:"
echo "   1. Test token generation: curl -X POST https://hearth.gregh.dev/api/v1/voice/token"
echo "   2. Verify WebSocket: wscat -c wss://hearth.gregh.dev/livekit"
echo ""
