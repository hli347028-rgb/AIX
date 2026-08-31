#!/bin/bash
set -euo pipefail

python3 /tmp/fix-transfer-partners-config.py

timeout 8 /opt/aix/bin/server -conf /opt/aix/configs > /tmp/aix-boot.log 2>&1 || true
if grep -qE 'Failed to config decode|panic|FATAL' /tmp/aix-boot.log; then
  echo BOOT_FAIL
  head -30 /tmp/aix-boot.log
  exit 1
fi
echo BOOT_OK
grep -E 'transfer partners|http.timeout|HTTP server' /tmp/aix-boot.log || true

sudo systemctl start aix
sleep 3
systemctl is-active aix

curl -sS -o /tmp/credit_probe.json -w 'HTTP=%{http_code}\n' \
  -X POST http://127.0.0.1:9000/v1/transfer/credit \
  -H 'Content-Type: application/json' \
  -d '{}'
head -c 300 /tmp/credit_probe.json; echo
curl -sS -o /dev/null -w 'admin=%{http_code}\n' http://127.0.0.1:8080/admin/
mysql -uroot -proot aix -N -e "SHOW TABLES LIKE 'partner_nonces';" 2>/dev/null || true

echo DONE
