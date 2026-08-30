#!/bin/bash
# Enable on-chain withdraw payout on the JP host. The private key already lives in
# /opt/aix/secrets/withdraw.key (mode 600) and is only referenced by path here.
set -eu

CFG=/opt/aix/configs/config.yaml
TS=$(date +%Y%m%d%H%M%S)
BACKUP_DIR="$HOME/aix-backups"
mkdir -p "$BACKUP_DIR"
cp "$CFG" "${BACKUP_DIR}/config.yaml.${TS}"
echo "config backed up to ${BACKUP_DIR}/config.yaml.${TS}"

ensure_wallet_key() {
  local key="$1" val="$2"
  if grep -qE "^\s*${key}:" "$CFG"; then
    sed -i "s#^\(\s*\)${key}:.*#\1${key}: ${val}#" "$CFG"
    echo "updated: ${key}: ${val}"
  else
    sed -i "0,/^\(\s*\)win_decimals:.*/s##&\n\1${key}: ${val}#" "$CFG"
    echo "added: ${key}: ${val}"
  fi
}

ensure_wallet_key withdraw_payout_enabled 'true'
ensure_wallet_key withdraw_private_key_file '"/opt/aix/secrets/withdraw.key"'
ensure_wallet_key withdraw_payout_queries_per_cycle '10'
ensure_wallet_key withdraw_payout_query_interval_seconds '5'

echo "=== yaml sanity ==="
python3 - "$CFG" <<'PY'
import sys, yaml
w = yaml.safe_load(open(sys.argv[1], encoding='utf-8'))['wallet']
for k in ('withdraw_payout_enabled','withdraw_private_key_file','withdraw_payout_queries_per_cycle','withdraw_payout_query_interval_seconds'):
    print(f"  wallet.{k} = {w.get(k)!r}")
PY

echo "=== key file present (content never printed) ==="
stat -c '%n mode=%a size=%s' /opt/aix/secrets/withdraw.key

echo "=== restart & verify ==="
sudo systemctl restart aix
sleep 6
systemctl is-active aix

echo "=== payout endpoint responds (no pending withdrawal expected) ==="
curl -fsS --max-time 20 http://127.0.0.1:9000/api/admin_dhb/withdraw_payout || echo "(endpoint call failed)"
echo
sleep 3
journalctl -u aix -n 60 --no-pager | grep -iE 'withdraw payout' | tail -8 || echo "(no payout log lines)"

echo "=== cron (payout trigger intentionally NOT added yet) ==="
crontab -l | grep -vE '^\s*#|^\s*$'

echo PAYOUT_OK
