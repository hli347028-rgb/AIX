#!/bin/bash
# Apply server binary + admin dist only (no web replace).
set -euo pipefail
TS=$(date +%Y%m%d%H%M%S)

echo "=== backup and replace server ==="
sudo systemctl stop aix
sudo cp /opt/aix/bin/server "/opt/aix/bin/server.bak.${TS}"
sudo install -m 755 /tmp/server-linux.new /opt/aix/bin/server

echo "=== deploy admin ==="
sudo mkdir -p /opt/aix/www/admin
sudo rsync -a --delete /tmp/admin-dist-new/ /opt/aix/www/admin/
sudo chown -R ubuntu:ubuntu /opt/aix/www/admin
sudo chmod -R a+rX /opt/aix/www

echo "=== start aix ==="
sudo systemctl start aix
sleep 3
systemctl is-active aix

echo "=== wait for :9000 ==="
for i in $(seq 1 36); do
  if ss -lntp | grep -q ':9000'; then
    echo READY after ${i}
    curl -sS -o /dev/null -w 'api:%{http_code}\n' http://127.0.0.1:9000/v1/announcements
    curl -skS -o /dev/null -w 'admin:%{http_code}\n' -H 'Host: aixai.pro' https://127.0.0.1/admin/
    echo APPLY_ADMIN_OK
    exit 0
  fi
  echo "waiting ${i}"
  sleep 5
done
echo STILL_DOWN
tail -20 /opt/aix/logs/backend.err.log
exit 1
