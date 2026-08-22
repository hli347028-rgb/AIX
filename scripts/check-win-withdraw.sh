#!/bin/bash
set -e
export MYSQL_PWD=root
mysql -uroot aix <<'SQL'
SELECT id, asset, amount, pay_amount, fee, status, to_address, tx_hash, LEFT(remark,80) remark, created_time
FROM withdrawals WHERE asset='WIN' ORDER BY id DESC LIMIT 15;
SELECT '--- pending ---' AS info;
SELECT id, pay_amount, status, to_address, remark FROM withdrawals
WHERE asset='WIN' AND status IN ('pending','doing') ORDER BY id;
SQL

echo "--- hot wallet (WIN = ERC20 token) ---"
RPC=$(grep rpc_url /opt/aix/configs/config.yaml | head -1 | awk '{print $2}' | tr -d '"')
WIN=$(grep win_contract /opt/aix/configs/config.yaml | head -1 | awk '{print $2}' | tr -d '"')
echo "win_contract: $WIN"
echo "rpc: $RPC"
echo "Set ADDR=0x... then re-run balance section, or paste address below:"
ADDR="${ADDR:-}"
if [ -n "$ADDR" ]; then
  BAL=$(curl -sS -X POST "$RPC" -H 'Content-Type: application/json' \
    -d "{\"jsonrpc\":\"2.0\",\"method\":\"eth_getBalance\",\"params\":[\"$ADDR\",\"latest\"],\"id\":1}" \
    | python3 -c "import sys,json; print(int(json.load(sys.stdin)['result'],16)/1e18)")
  echo "native_balance (gas): $BAL"
  DATA="0x70a08231$(printf '%064s' ${ADDR#0x} | tr ' ' 0)"
  BAL2=$(curl -sS -X POST "$RPC" -H 'Content-Type: application/json' \
    -d "{\"jsonrpc\":\"2.0\",\"method\":\"eth_call\",\"params\":[{\"to\":\"$WIN\",\"data\":\"$DATA\"},\"latest\"],\"id\":1}" \
    | python3 -c "import sys,json; r=json.load(sys.stdin).get('result','0x0'); print(int(r,16)/1e18)")
  echo "win_token_balance: $BAL2"
  NEED=$(export MYSQL_PWD=root; mysql -uroot aix -N -e "SELECT SUM(pay_amount) FROM withdrawals WHERE asset='WIN' AND status IN ('pending','doing')")
  echo "pending_payout_total: $NEED WIN"
fi
