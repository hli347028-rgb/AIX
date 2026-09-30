#!/bin/bash
# One-shot apply for uploaded /tmp artifacts + nginx ssl conf.
set -euo pipefail
TS=$(date +%Y%m%d%H%M%S)

echo "=== backup and replace server ==="
sudo systemctl stop aix
sudo cp /opt/aix/bin/server "/opt/aix/bin/server.bak.${TS}"
sudo install -m 755 /tmp/server-linux.new /opt/aix/bin/server

echo "=== deploy web (preserve admin) ==="
sudo rsync -a --delete --exclude admin /tmp/web-dist-new/ /opt/aix/www/

echo "=== deploy admin ==="
sudo mkdir -p /opt/aix/www/admin
sudo rsync -a --delete /tmp/admin-dist-new/ /opt/aix/www/admin/
sudo chown -R ubuntu:ubuntu /opt/aix/www
sudo chmod -R a+rX /opt/aix/www

echo "=== apply nginx ssl config ==="
sudo sed -i 's/\r$//' /opt/aix/scripts/aix.nginx.conf /opt/aix/scripts/aix.nginx.ssl.conf
sudo cp /opt/aix/scripts/aix.nginx.ssl.conf /etc/nginx/sites-available/aix
sudo nginx -t
sudo systemctl reload nginx

echo "=== start aix ==="
sudo systemctl start aix
sleep 3
systemctl is-active aix
systemctl is-active nginx

echo "=== verify ==="
test -f /opt/aix/www/favicon.ico
ls -la /opt/aix/www/favicon.ico /opt/aix/www/index.html /opt/aix/www/admin/index.html
curl -sS -o /dev/null -w 'home:%{http_code} favicon:%{http_code} admin:%{http_code} api:%{http_code}\n' \
  http://127.0.0.1/ \
  http://127.0.0.1/favicon.ico \
  http://127.0.0.1/admin/ \
  http://127.0.0.1:9000/v1/announcements
curl -sS -o /tmp/favicon.check -D /tmp/favicon.hdr -w 'favicon_bytes:%{size_download} http:%{http_code}\n' http://127.0.0.1/favicon.ico
grep -i '^Content-Type:' /tmp/favicon.hdr || true
file /tmp/favicon.check || true

echo APPLY_OK
