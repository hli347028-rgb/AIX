#!/bin/bash
set -u
sudo systemctl start mysql nginx cron aix
sudo systemctl enable mysql nginx cron aix >/dev/null 2>&1
echo "=== status ==="
for s in mysql nginx cron aix; do
  printf '%-8s %s\n' "$s" "$(systemctl is-active "$s")"
done
echo "=== listen ==="
ss -lntp | grep -E ':80 |:443 |:8080 |:9000 |:3306 ' || true
echo "=== crontab ==="
crontab -l 2>/dev/null || echo NO_CRONTAB
