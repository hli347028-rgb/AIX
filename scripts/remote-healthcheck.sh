#!/bin/bash
set -u
BASE="https://aixai.pro"
CURL='curl -sS --max-time 8 -o /tmp/aix_hc_body -w %{http_code}'

echo "======== services ========"
for s in nginx mysql cron aix; do
  st=$(systemctl is-active "$s" 2>/dev/null || echo missing)
  printf '%-8s %s\n' "$s" "$st"
done
echo
echo "======== listen ========"
ss -lntp | grep -E ':80 |:443 |:8080 |:9000 |:3306 ' || true
echo
echo "======== mysql ========"
export MYSQL_PWD=root
if mysql --user=root --execute="SELECT 1 AS ok;" >/dev/null 2>&1; then
  echo "mysql ping: OK"
  mysql --user=root --execute="SHOW DATABASES LIKE 'aix'; SELECT COUNT(*) AS tables FROM information_schema.tables WHERE table_schema='aix';"
else
  echo "mysql ping: FAIL"
fi
echo
echo "======== crontab ========"
crontab -l 2>/dev/null || echo NO_CRONTAB
echo "--- cron.log last 12 ---"
tail -n 12 /opt/aix/logs/cron.log 2>/dev/null || echo NO_CRON_LOG
echo
echo "======== nginx -t ========"
sudo nginx -t 2>&1 | tail -n 5
echo
echo "======== backend logs (error/panic) ========"
if [ -f /opt/aix/logs/backend.err.log ]; then
  grep -E 'panic|ERROR|failed' /opt/aix/logs/backend.err.log | tail -n 20 || echo no_err_matches
else
  echo NO_ERR_LOG
fi
echo
echo "======== API probe ========"
printf '%-8s %-6s %s\n' "EXPECT" "CODE" "PATH"

probe() {
  local method="$1" expect="$2" path="$3"
  local extra="${4-}"
  local code
  if [ "$method" = POST ]; then
    code=$(curl -skS --max-time 8 -o /tmp/aix_hc_body -w '%{http_code}' -X POST -H 'Content-Type: application/json' -d '{}' $extra "${BASE}${path}")
  else
    code=$(curl -skS --max-time 8 -o /tmp/aix_hc_body -w '%{http_code}' $extra "${BASE}${path}")
  fi
  local mark="OK"
  if [ "$code" = "000" ]; then
    mark="FAIL"
  elif [ "$code" -ge 500 ] 2>/dev/null; then
    mark="FAIL"
  elif [ "$expect" = "2xx" ] && [ "$code" -lt 200 -o "$code" -ge 300 ]; then
    mark="WARN"
  elif [ "$expect" = "401" ] && [ "$code" != "401" ]; then
    mark="WARN"
  fi
  printf '%-8s %-6s %s\n' "$mark" "$code" "$method $path"
  if [ "$mark" != "OK" ]; then
    echo "         body=$(head -c 180 /tmp/aix_hc_body | tr '\n' ' ')"
  fi
}

# pages
probe GET 2xx /
probe GET 2xx /admin/
probe GET 2xx /static/index.53177691.js
probe GET 2xx /admin/js/app.a1e5a4df.js

# public/cron
probe GET 2xx /api/admin_dhb/deposit_only
probe GET 2xx /api/admin_dhb/deposit_only_win
probe GET 2xx /api/admin_dhb/win_price_oracle

# open api without key
probe GET 401 /v1/open/subscribe-orders

# auth
probe GET 2xx '/v1/auth/challenge?address=0x0000000000000000000000000000000000000001'
probe POST 401 /v1/auth/login
probe GET 401 /v1/auth/profile
probe GET 401 /v1/auth/invitees

# wallet GET (need login)
for p in \
  /v1/wallet/balance \
  /v1/wallet/recharges \
  /v1/wallet/recharges-win \
  /v1/wallet/withdrawals \
  /v1/wallet/orders \
  /v1/wallet/releases \
  /v1/wallet/referral-rewards \
  /v1/wallet/transfer-records/self \
  /v1/wallet/transfer-records/lineal \
  /v1/wallet/exchange-records \
  /v1/wallet/aix-price \
  /v1/wallet/rewards \
  /v1/wallet/management-rewards \
  /v1/wallet/aix-profile \
  /v1/wallet/points-records
do
  probe GET 401 "$p"
done

# wallet POST empty (need login or 400)
for p in \
  /v1/wallet/recharge \
  /v1/wallet/recharge/confirm \
  /v1/wallet/withdraw \
  /v1/wallet/claim \
  /v1/wallet/subscribe \
  /v1/wallet/subscribe-aix \
  /v1/wallet/transfer \
  /v1/wallet/recharge-to-reward \
  /v1/wallet/withdraw-aix \
  /v1/wallet/exchange-aix-to-win \
  /v1/wallet/withdraw-win \
  /v1/wallet/recharge-win \
  /v1/wallet/recharge-win/confirm
do
  probe POST 401 "$p"
done

# admin GET without session
for p in \
  /api/admin_dhb/all \
  /api/admin_dhb/user_list \
  /api/admin_dhb/config \
  /api/admin_dhb/buy_list \
  /api/admin_dhb/withdraw_list \
  /api/admin_dhb/exchange_list \
  /api/admin_dhb/reward_list \
  /api/admin_dhb/record_list \
  /api/admin_dhb/good_list \
  /api/admin_dhb/settlement_list \
  /api/admin_dhb/my_auth_list \
  /api/admin_dhb/user_recommend \
  /api/admin_dhb/sub_money \
  /api/admin_dhb/location_list
do
  probe GET 401 "$p"
done

probe POST 401 /api/admin_dhb/login

# kratos admin
for p in \
  /v1/admin/users \
  /v1/admin/config \
  /v1/admin/orders
do
  probe GET 401 "$p"
done
probe POST 401 /v1/admin/settlement/trigger

echo
echo HEALTHCHECK_DONE
