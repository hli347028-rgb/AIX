#!/bin/bash
set -u
systemctl is-active aix
ss -lntp | grep -E ':9000|:8080' || true
sleep 2
curl -sS -m 5 -o /tmp/credit_probe.json -w 'HTTP=%{http_code}\n' \
  -X POST http://127.0.0.1:9000/v1/transfer/credit \
  -H 'Content-Type: application/json' \
  -d '{}' || echo CURL_FAIL
echo -n 'BODY='; cat /tmp/credit_probe.json 2>/dev/null; echo
curl -sS -m 5 -o /dev/null -w 'admin=%{http_code}\n' http://127.0.0.1:8080/admin/ || true
ls /opt/aix/www/admin/js/app.*.js 2>/dev/null | head -3
mysql -uroot -proot -N -e "SHOW TABLES FROM aix LIKE 'partner_nonces';" 2>/dev/null || true
sudo journalctl -u aix -n 25 --no-pager
