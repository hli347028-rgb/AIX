#!/bin/bash
set -euo pipefail
export MYSQL_PWD=root

echo "=== mysql login ==="
mysql --user=root --execute="SELECT 1 AS ok;"

echo "=== mysql tune ==="
sudo tee /etc/mysql/mysql.conf.d/aix-small.cnf >/dev/null <<'EOF'
[mysqld]
innodb_buffer_pool_size=256M
performance_schema=OFF
max_connections=50
innodb_flush_log_at_trx_commit=2
skip_name_resolve=ON
EOF
sudo systemctl restart mysql
mysql --user=root --execute="SHOW VARIABLES LIKE 'innodb_buffer_pool_size';"

echo "=== dirs ==="
sudo mkdir -p /opt/aix/bin /opt/aix/configs /opt/aix/scripts/cron /opt/aix/www /opt/aix/logs
sudo chown -R ubuntu:ubuntu /opt/aix

echo "=== packages ==="
sudo apt-get update -y
sudo DEBIAN_FRONTEND=noninteractive apt-get install -y curl
echo PREP_OK
