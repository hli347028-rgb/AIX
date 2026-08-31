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
sleep 4
systemctl is-active aix

echo "==== dashboard field check ===="
curl -sS -m 8 -o /tmp/all.json -w 'all_http=%{http_code}\n' http://127.0.0.1:9000/api/admin_dhb/all || true
grep -o 'totalAixAsset' /tmp/all.json || echo 'totalAixAsset not in body (likely auth-gated)'
head -c 200 /tmp/all.json; echo

echo "==== db sanity ===="
mysql -uroot -proot aix -N -e "SELECT COALESCE(SUM(aix_balance),0) AS total_aix FROM users;" 2>/dev/null || true

echo "==== logs ===="
sudo journalctl -u aix -n 15 --no-pager
echo DONE
