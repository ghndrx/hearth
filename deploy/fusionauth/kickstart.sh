#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

ENV_FILE="$SCRIPT_DIR/.env"
EXAMPLE_FILE="$SCRIPT_DIR/.env.fusionauth.example"
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.fusionauth.yml"
FUSIONAUTH_PORT="${FUSIONAUTH_PORT:-9011}"
HEALTH_URL="http://localhost:${FUSIONAUTH_PORT}/api/status"
MAX_WAIT=120

echo "=== Hearth FusionAuth Kickstart ==="
echo ""

# --- Generate .env if it doesn't exist ---
if [ -f "$ENV_FILE" ]; then
    echo "[ok] .env already exists, skipping generation"
else
    if [ ! -f "$EXAMPLE_FILE" ]; then
        echo "[error] $EXAMPLE_FILE not found" >&2
        exit 1
    fi

    echo "[*] Generating secrets..."
    CLIENT_SECRET="$(openssl rand -hex 32)"
    API_KEY="$(openssl rand -hex 32)"
    DB_PASSWORD="$(openssl rand -hex 32)"

    cp "$EXAMPLE_FILE" "$ENV_FILE"

    # Fill in generated secrets
    sed -i "s|your-client-secret|${CLIENT_SECRET}|g" "$ENV_FILE"
    sed -i "s|your-api-key|${API_KEY}|g" "$ENV_FILE"
    sed -i "s|change-me-to-a-secure-password|${DB_PASSWORD}|g" "$ENV_FILE"

    echo "[ok] .env created with generated secrets"
fi

# --- Ensure shared network exists ---
if ! docker network inspect hearth-network >/dev/null 2>&1; then
    echo "[*] Creating hearth-network..."
    docker network create hearth-network
    echo "[ok] hearth-network created"
else
    echo "[ok] hearth-network already exists"
fi

# --- Start FusionAuth ---
echo "[*] Starting FusionAuth via docker compose..."
docker compose -f "$COMPOSE_FILE" up -d

# --- Wait for healthy ---
echo "[*] Waiting for FusionAuth to be ready (up to ${MAX_WAIT}s)..."
elapsed=0
while [ $elapsed -lt $MAX_WAIT ]; do
    if curl -sf "$HEALTH_URL" >/dev/null 2>&1; then
        echo "[ok] FusionAuth is healthy!"
        echo ""
        echo "=== FusionAuth is ready ==="
        echo ""
        echo "  Admin UI:   http://localhost:${FUSIONAUTH_PORT}"
        echo ""
        echo "  First visit: complete the setup wizard to create your admin account."
        echo ""
        echo "  Next steps:"
        echo "    1. Create an Application in FusionAuth for Hearth"
        echo "    2. Note the Application ID, Client ID, and Client Secret"
        echo "    3. Create an API Key (Settings > API Keys)"
        echo "    4. Update your Hearth .env with AUTH_PROVIDER=fusionauth and the values above"
        echo "    5. See README.md for full configuration details"
        echo ""
        exit 0
    fi
    sleep 3
    elapsed=$((elapsed + 3))
    printf "  ... %ds\n" "$elapsed"
done

echo ""
echo "[warn] FusionAuth did not become healthy within ${MAX_WAIT}s."
echo "  It may still be starting. Check logs with:"
echo "    docker compose -f $COMPOSE_FILE logs -f fusionauth"
echo ""
exit 1
