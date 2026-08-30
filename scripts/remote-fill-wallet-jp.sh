#!/bin/bash
# Fill the wallet keys that are absent on the JP server, using the values from the
# sibling prod host (same EOEO chain). Existing keys are left untouched here:
# deposit_contract / win_deposit_contract / deposit_address differ per host and are
# decided separately because they drive on-chain recharge scanning.
set -eu

CFG=/opt/aix/configs/config.yaml
TS=$(date +%Y%m%d%H%M%S)
BACKUP_DIR="$HOME/aix-backups"
mkdir -p "$BACKUP_DIR"
cp "$CFG" "${BACKUP_DIR}/config.yaml.${TS}"
echo "config backed up to ${BACKUP_DIR}/config.yaml.${TS}"

# key:value pairs to ensure under the wallet: section
add_after_win_decimals() {
  local key="$1" val="$2"
  if grep -qE "^\s*${key}:" "$CFG"; then
    echo "keep existing: ${key}"
  else
    # insert right after the wallet-level win_decimals line to stay inside wallet:
    # '#' delimiter because values contain slashes (URLs)
    sed -i "0,/^\(\s*\)win_decimals:.*/s##&\n\1${key}: ${val}#" "$CFG"
    echo "added: ${key}: ${val}"
  fi
}

add_after_win_decimals win_a_contract '"0x3dce8Ef2646e3082f1e4afbC26Cf747e4e238A8D"'
add_after_win_decimals win_a_decimals '18'
add_after_win_decimals win_a_deposit_contract '"0xcaa39A8E23F5548AD85d9e2B9B21F63E99505040"'
add_after_win_decimals win_a_recharge_enabled 'true'
add_after_win_decimals sdt_contract '"0x314D550572a0fA001B465a9EBc1dd04D834a0688"'
add_after_win_decimals sdt_decimals '18'
add_after_win_decimals bsc_rpc_url '"https://bsc-dataseed.binance.org"'

# win_contract exists but is empty on this host; align it with the sibling value.
if grep -qE '^\s*win_contract:\s*""\s*$' "$CFG"; then
  sed -i 's|^\(\s*\)win_contract:\s*""\s*$|\1win_contract: "0x193013574dacbd38bf26ecb654b3fd787b94d216"|' "$CFG"
  echo 'set: win_contract (was empty)'
else
  echo 'keep existing: win_contract'
fi

echo "=== wallet keys after ==="
grep -nE '^\s*(deposit_contract|win_deposit_contract|win_a_deposit_contract|win_a_recharge_enabled|deposit_address|usdt_contract|usdt_decimals|win_contract|win_decimals|win_a_contract|win_a_decimals|sdt_contract|sdt_decimals|rpc_url|bsc_rpc_url):' "$CFG"

echo "=== yaml sanity (python) ==="
python3 - "$CFG" <<'PY'
import sys, yaml
cfg = yaml.safe_load(open(sys.argv[1], encoding='utf-8'))
w = cfg['wallet']
for k in ('win_a_contract','win_a_decimals','win_a_deposit_contract','win_a_recharge_enabled','sdt_contract','sdt_decimals','bsc_rpc_url','win_contract'):
    print(f"  wallet.{k} = {w.get(k)!r}")
print("  partners =", [p.get('partner_id') for p in (cfg.get('transfer_partners') or {}).get('partners', [])])
PY

echo "=== restart & verify ==="
sudo systemctl restart aix
sleep 6
systemctl is-active aix
journalctl -u aix -n 40 --no-pager | grep -iE 'error|panic|transfer partners' | tail -8 || true

echo FILL_WALLET_OK
