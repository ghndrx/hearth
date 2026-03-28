#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "FusionAuth starting... run ./kickstart.sh to complete setup"
exec "$SCRIPT_DIR/kickstart.sh"
