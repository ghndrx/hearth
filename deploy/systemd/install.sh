#!/bin/bash
# Hearth Installation Script for systemd
# Run as root or with sudo

set -e

HEARTH_VERSION="${1:-latest}"
HEARTH_USER="hearth"
HEARTH_HOME="/opt/hearth"
HEARTH_DATA="/var/lib/hearth"

echo "Installing Hearth ${HEARTH_VERSION}..."

# Create user
if ! id "$HEARTH_USER" &>/dev/null; then
    useradd -r -s /bin/false -d "$HEARTH_HOME" "$HEARTH_USER"
    echo "Created user: $HEARTH_USER"
fi

# Create directories
mkdir -p "$HEARTH_HOME" "$HEARTH_DATA"/{uploads,db}
chown -R "$HEARTH_USER:$HEARTH_USER" "$HEARTH_HOME" "$HEARTH_DATA"

# Download binary
ARCH=$(uname -m)
case $ARCH in
    x86_64) ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

if [ "$HEARTH_VERSION" = "latest" ]; then
    DOWNLOAD_URL="https://github.com/ghndrx/hearth/releases/latest/download/hearth-linux-${ARCH}"
else
    DOWNLOAD_URL="https://github.com/ghndrx/hearth/releases/download/${HEARTH_VERSION}/hearth-linux-${ARCH}"
fi

echo "Downloading from: $DOWNLOAD_URL"
curl -L "$DOWNLOAD_URL" -o "$HEARTH_HOME/hearth"
chmod +x "$HEARTH_HOME/hearth"

# Create config if it doesn't exist
if [ ! -f "$HEARTH_HOME/config.yaml" ]; then
    SECRET_KEY=$(openssl rand -base64 32)
    cat > "$HEARTH_HOME/config.yaml" <<EOF
server:
  host: 0.0.0.0
  port: 8080
  public_url: http://localhost:8080

database:
  url: sqlite://${HEARTH_DATA}/db/hearth.db

storage:
  backend: local
  path: ${HEARTH_DATA}/uploads

auth:
  secret_key: ${SECRET_KEY}
  registration_enabled: true
  invite_only: false

voice:
  enabled: true
  turn:
    enabled: true
    port: 3478
EOF
    chown "$HEARTH_USER:$HEARTH_USER" "$HEARTH_HOME/config.yaml"
    chmod 600 "$HEARTH_HOME/config.yaml"
    echo "Created config: $HEARTH_HOME/config.yaml"
fi

# Install systemd service
cp "$(dirname "$0")/hearth.service" /etc/systemd/system/hearth.service
systemctl daemon-reload
systemctl enable hearth

echo ""
echo "Hearth installed successfully!"
echo ""
echo "Next steps:"
echo "  1. Edit config: sudo nano $HEARTH_HOME/config.yaml"
echo "  2. Set PUBLIC_URL to your domain"
echo "  3. Start service: sudo systemctl start hearth"
echo "  4. Check status: sudo systemctl status hearth"
echo "  5. View logs: sudo journalctl -u hearth -f"
echo ""
echo "Default URL: http://localhost:8080"
