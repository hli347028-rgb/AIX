#!/bin/bash
set -u

for i in $(seq 1 20); do
  if ss -lnt | grep -q ':9000'; then break; fi
  sleep 1
done

systemctl is-active aix
ss -lnt | grep ':9000' || echo 'port 9000 not listening'

curl -sS -m 8 -o /tmp/all.json -w 'all_http=%{http_code}\n' http://127.0.0.1:9000/api/admin_dhb/all || true
if grep -q 'totalAixAsset' /tmp/all.json 2>/dev/null; then
  echo 'FIELD_PRESENT totalAixAsset'
else
  echo 'field not visible in this response:'
fi
head -c 250 /tmp/all.json 2>/dev/null; echo

curl -sS -m 8 -o /dev/null -w 'admin=%{http_code}\n' http://127.0.0.1:8080/admin/ || true
mysql -uroot -proot aix -N -e "SELECT COALESCE(SUM(aix_balance),0) FROM users;" 2>/dev/null || true

sudo journalctl -u aix -n 12 --no-pager
echo DONE
