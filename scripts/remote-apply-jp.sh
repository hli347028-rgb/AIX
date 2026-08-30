#!/bin/bash
# Apply artifacts uploaded to /tmp onto the JP server (54.150.130.182).
# Layout differs from the other prod host: binary /opt/aix/aix-server, web root /opt/aix/web.
# config.yaml is environment-specific here and must never be overwritten.
set -eu

TS=$(date +%Y%m%d%H%M%S)
BACKUP_DIR="$HOME/aix-backups"
mkdir -p "$BACKUP_DIR"

echo "=== backup & replace backend binary ==="
cp /opt/aix/aix-server "${BACKUP_DIR}/aix-server.${TS}"
install -m 755 /tmp/server-linux.new /opt/aix/aix-server
rm -f /tmp/server-linux.new

echo "=== deploy web dist (keep web/admin) ==="
rm -rf /tmp/web-dist-extract
mkdir -p /tmp/web-dist-extract
tar -xzf /tmp/web-dist-deploy.tar.gz -C /tmp/web-dist-extract
sudo rsync -a --delete --no-owner --no-group --exclude admin /tmp/web-dist-extract/dist/ /opt/aix/web/
sudo chown -R ubuntu:ubuntu /opt/aix/web
sudo chmod -R a+rX /opt/aix/web

echo "=== restart aix ==="
sudo systemctl restart aix
sleep 8
systemctl is-active aix

echo "=== deployed assets ==="
grep -o 'index\.[a-z0-9]*\.js' /opt/aix/web/index.html | head -2
grep -o 'app\.[a-z0-9]*\.js' /opt/aix/web/admin/index.html | head -1
ls -la /opt/aix/aix-server

echo APPLY_JP_OK
