#!/bin/bash
# AIX chain jobs for crontab — each HTTP call starts one server-side cycle (10 runs, 5s apart).
# withdraw_payout covers WIN native payout + AIX-SDT ERC20 payout (same cycle params).
# Prerequisite: recharge_monitor_enabled=false and win_price_oracle_enabled=false
set -u
BASE="${AIX_HTTP:-http://127.0.0.1:9000}"
TS="$(date -Is 2>/dev/null || date)"
echo "[$TS] start BASE=$BASE"

curl -fsS --max-time 15 "$BASE/api/admin_dhb/deposit_only" || echo "[$TS] deposit_only FAILED"
echo
curl -fsS --max-time 15 "$BASE/api/admin_dhb/deposit_only_win" || echo "[$TS] deposit_only_win FAILED"
echo
curl -fsS --max-time 15 "$BASE/api/admin_dhb/win_price_oracle" || echo "[$TS] win_price_oracle FAILED"
echo
curl -fsS --max-time 15 "$BASE/api/admin_dhb/withdraw_payout" || echo "[$TS] withdraw_payout FAILED"
echo
echo "[$TS] trigger done (cycles run in server background)"
