#!/bin/bash
# Hearth Uninstallation Script
# Run as root or with sudo

set -e

HEARTH_USER="hearth"
HEARTH_HOME="/opt/hearth"
HEARTH_DATA="/var/lib/hearth"

echo "Uninstalling Hearth..."

# Stop and disable service
systemctl stop hearth 2>/dev/null || true
systemctl disable hearth 2>/dev/null || true
rm -f /etc/systemd/system/hearth.service
systemctl daemon-reload

# Remove user
userdel "$HEARTH_USER" 2>/dev/null || true

echo ""
read -p "Remove application files ($HEARTH_HOME)? [y/N] " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    rm -rf "$HEARTH_HOME"
    echo "Removed: $HEARTH_HOME"
fi

echo ""
read -p "Remove data files ($HEARTH_DATA)? WARNING: This deletes all data! [y/N] " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    rm -rf "$HEARTH_DATA"
    echo "Removed: $HEARTH_DATA"
fi

echo ""
echo "Hearth uninstalled."
