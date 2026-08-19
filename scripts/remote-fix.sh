#!/bin/bash
set -euo pipefail
export MYSQL_PWD=root

mysql --user=root <<'SQL'
CREATE USER IF NOT EXISTS 'root'@'127.0.0.1' IDENTIFIED WITH caching_sha2_password BY 'root';
GRANT ALL PRIVILEGES ON aix.* TO 'root'@'127.0.0.1';
FLUSH PRIVILEGES;
SQL

if [ -f /opt/aix/scripts/aix-chain-jobs.sh ]; then
  mv -f /opt/aix/scripts/aix-chain-jobs.sh /opt/aix/scripts/cron/aix-chain-jobs.sh
fi
chmod +x /opt/aix/scripts/cron/aix-chain-jobs.sh

(crontab -l 2>/dev/null | grep -v 'aix-chain-jobs.sh' || true
 echo '* * * * * AIX_HTTP=http://127.0.0.1:9000 /opt/aix/scripts/cron/aix-chain-jobs.sh >> /opt/aix/logs/cron.log 2>&1') | crontab -

sudo systemctl restart aix
sleep 3
sudo systemctl --no-pager --full status aix || true
curl -sS -o /dev/null -w 'WEB:%{http_code}\n' http://127.0.0.1:8080/
curl -sS -o /dev/null -w 'ADMIN:%{http_code}\n' http://127.0.0.1:8080/admin/
curl -sS -o /dev/null -w 'API:%{http_code}\n' http://127.0.0.1:9000/api/admin_dhb/deposit_only || true
mysql --user=root --execute="SELECT table_name, table_rows FROM information_schema.tables WHERE table_schema='aix';"
echo FIX_OK
