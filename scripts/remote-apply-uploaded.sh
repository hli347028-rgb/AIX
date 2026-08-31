#!/bin/bash
# Apply artifacts uploaded to /tmp by local deploy (do not overwrite configs).
set -euo pipefail

TS=$(date +%Y%m%d%H%M%S)

echo "=== backup & replace server ==="
cp /opt/aix/bin/server "/opt/aix/bin/server.bak.${TS}"
mv /tmp/server-linux.new /opt/aix/bin/server
chmod +x /opt/aix/bin/server

echo "=== update scripts ==="
install -m 755 /tmp/aix-chain-jobs.sh /opt/aix/scripts/cron/aix-chain-jobs.sh
install -m 755 /tmp/remote-sync-www.sh /opt/aix/scripts/remote-sync-www.sh

echo "=== deploy web dist (preserve www/admin during rsync) ==="
rm -rf /tmp/web-dist-extract
mkdir -p /tmp/web-dist-extract
tar -xzf /tmp/web-dist-deploy.tar.gz -C /tmp/web-dist-extract
rsync -a --delete --no-owner --no-group --exclude admin /tmp/web-dist-extract/dist/ /opt/aix/www/

echo "=== deploy admin dist ==="
ADMIN_SRC=/tmp/admin-dist-new
if [ -f "${ADMIN_SRC}/dist/index.html" ]; then
  ADMIN_SRC="${ADMIN_SRC}/dist"
fi
sudo rsync -a --delete --no-owner --no-group "${ADMIN_SRC}/" /opt/aix/www/admin/
sudo chown -R ubuntu:ubuntu /opt/aix/www/admin

echo "=== restart aix (config.yaml not touched) ==="
sudo systemctl restart aix
sleep 3
systemctl is-active aix

bash /opt/aix/scripts/remote-sync-www.sh

echo APPLY_UPLOADED_OK
