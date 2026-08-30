#!/bin/bash
# Install /tmp/server-linux.new as the JP backend binary and restart the service.
set -eu

TS=$(date +%Y%m%d%H%M%S)
BACKUP_DIR="$HOME/aix-backups"
mkdir -p "$BACKUP_DIR"

cp /opt/aix/aix-server "${BACKUP_DIR}/aix-server.${TS}"
install -m 755 /tmp/server-linux.new /opt/aix/aix-server
rm -f /tmp/server-linux.new

sudo systemctl restart aix
sleep 8
systemctl is-active aix
ls -la /opt/aix/aix-server
echo INSTALL_JP_OK
