#!/bin/bash
set -euo pipefail
export MYSQL_PWD=root

echo "=== import schema ==="
mysql --user=root --execute="CREATE DATABASE IF NOT EXISTS aix DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
mysql --user=root aix < /opt/aix/scripts/aix-schema.sql
mysql --user=root --execute="SHOW TABLES FROM aix;"

echo "=== nginx ==="
sudo cp /opt/aix/scripts/aix.nginx.conf /etc/nginx/sites-available/aix
sudo ln -sfn /etc/nginx/sites-available/aix /etc/nginx/sites-enabled/aix
sudo rm -f /etc/nginx/sites-enabled/default
sudo nginx -t
sudo systemctl reload nginx

echo "=== systemd ==="
sudo cp /opt/aix/scripts/aix.service /etc/systemd/system/aix.service
sudo systemctl daemon-reload
sudo systemctl enable aix
sudo systemctl restart aix
sleep 3
sudo systemctl --no-pager --full status aix || true

echo "=== crontab ==="
chmod +x /opt/aix/scripts/cron/aix-chain-jobs.sh
(crontab -l 2>/dev/null | grep -v 'aix-chain-jobs.sh' || true; echo '* * * * * AIX_HTTP=http://127.0.0.1:9000 /opt/aix/scripts/cron/aix-chain-jobs.sh >> /opt/aix/logs/cron.log 2>&1') | crontab -
crontab -l

echo DEPLOY_OK
