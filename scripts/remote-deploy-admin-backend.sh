#!/bin/bash
set -euo pipefail

sudo mkdir -p /opt/aix/www/admin
sudo rm -rf /opt/aix/www/admin/*
sudo tar -xzf /tmp/admin-dist.tar.gz -C /opt/aix/www/admin
sudo chmod -R a+rX /opt/aix/www/admin
rm -f /tmp/admin-dist.tar.gz
test -f /opt/aix/www/admin/index.html && echo ADMIN_OK

sudo systemctl stop aix
sudo mv /tmp/server.new /opt/aix/bin/server
sudo chmod +x /opt/aix/bin/server
sudo systemctl start aix

for i in $(seq 1 20); do
  if ss -lnt | grep -q ':9000'; then break; fi
  sleep 1
done

systemctl is-active aix
ss -lnt | grep ':9000' || echo 'port 9000 missing'
curl -sS -m 5 -o /dev/null -w 'admin=%{http_code}\n' http://127.0.0.1:8080/admin/ || true
sudo journalctl -u aix -n 8 --no-pager
echo DONE
