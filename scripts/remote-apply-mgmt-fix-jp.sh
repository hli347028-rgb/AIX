#!/bin/bash
# Install the instrumented backend and the rebuilt admin panel on JP, then reset the debug log.
set -eu

TS=$(date +%Y%m%d%H%M%S)
BACKUP_DIR="$HOME/aix-backups"
mkdir -p "$BACKUP_DIR"

cp /opt/aix/aix-server "${BACKUP_DIR}/aix-server.${TS}"
install -m 755 /tmp/server-linux.new /opt/aix/aix-server
rm -f /tmp/server-linux.new

sudo rsync -a --delete --no-owner --no-group /tmp/admin-dist-new/ /opt/aix/web/admin/
sudo chown -R ubuntu:ubuntu /opt/aix/web/admin
sudo chmod -R a+rX /opt/aix/web/admin
rm -rf /tmp/admin-dist-new

rm -f /opt/aix/debug-1ab7eb.log

sudo systemctl restart aix
sleep 8
systemctl is-active aix
grep -o 'app\.[a-z0-9]*\.js' /opt/aix/web/admin/index.html | head -1
echo MGMT_FIX_JP_OK
