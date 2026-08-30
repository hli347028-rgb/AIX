#!/bin/bash
# Point the JP host at the sibling host's deposit contracts.
#
# Order matters: the deposit cursors for the incoming contracts are seeded FIRST so the
# scanner starts at the sibling's current position. With a fresh cursor of 0 it would walk
# the whole historical deposit ledger (491 USDT records there) and re-credit all of it.
set -eu

CFG=/opt/aix/configs/config.yaml
TS=$(date +%Y%m%d%H%M%S)
BACKUP_DIR="$HOME/aix-backups"
mkdir -p "$BACKUP_DIR"

NEW_USDT_DEPOSIT="0xa5A438Bb1D0F702c684B4d7bAAE2C520aFb4aE86"
NEW_WIN_DEPOSIT="0x94db6bb040107ef9a2F1e9DB9d84dD8D6D98997e"
USDT_CURSOR=491
WIN_CURSOR=3

DBUSER=$(grep -m1 -E '^\s+user:' "$CFG" | sed -E 's/.*user:\s*//; s/"//g')
DBPASS=$(grep -m1 -E '^\s+password:' "$CFG" | sed -E 's/.*password:\s*//; s/"//g')
DBNAME=$(grep -m1 -E '^\s+dbname:' "$CFG" | sed -E 's/.*dbname:\s*//; s/"//g')
export MYSQL_PWD="$DBPASS"

echo "=== backup db + config ==="
mysqldump -u"$DBUSER" --single-transaction "$DBNAME" | gzip > "${BACKUP_DIR}/${DBNAME}-${TS}.sql.gz"
cp "$CFG" "${BACKUP_DIR}/config.yaml.${TS}"
ls -la "${BACKUP_DIR}/${DBNAME}-${TS}.sql.gz" "${BACKUP_DIR}/config.yaml.${TS}"

echo "=== seed deposit cursors for incoming contracts ==="
for pair in "$(echo "$NEW_USDT_DEPOSIT" | tr 'A-Z' 'a-z'):${USDT_CURSOR}" "$(echo "$NEW_WIN_DEPOSIT" | tr 'A-Z' 'a-z'):${WIN_CURSOR}"; do
  addr="${pair%:*}"
  cur="${pair##*:}"
  mysql -u"$DBUSER" -e "INSERT INTO ${DBNAME}.settings (\`key\`, value) VALUES ('deposit_only_cursor:${addr}', '${cur}') ON DUPLICATE KEY UPDATE value = VALUES(value)"
  echo "  cursor deposit_only_cursor:${addr} = ${cur}"
done

echo "=== cursors now ==="
mysql -u"$DBUSER" -N -e "SELECT \`key\`, value FROM ${DBNAME}.settings WHERE \`key\` LIKE 'deposit_only_cursor%'"

echo "=== swap deposit contracts ==="
sed -i "s#^\(\s*\)deposit_contract:.*#\1deposit_contract: \"${NEW_USDT_DEPOSIT}\"#" "$CFG"
sed -i "s#^\(\s*\)win_deposit_contract:.*#\1win_deposit_contract: \"${NEW_WIN_DEPOSIT}\"#" "$CFG"

echo "=== clear deposit_address / deposit_addresses (match sibling) ==="
sed -i "s#^\(\s*\)deposit_address:.*#\1deposit_address: \"\"#" "$CFG"
# drop the deposit_addresses key and its list items
python3 - "$CFG" <<'PY'
import re, sys
path = sys.argv[1]
lines = open(path, encoding='utf-8').read().splitlines(keepends=True)
out, skipping = [], False
for line in lines:
    if re.match(r'^\s*deposit_addresses:', line):
        skipping = True
        continue
    if skipping:
        if re.match(r'^\s*-\s', line):
            continue
        skipping = False
    out.append(line)
open(path, 'w', encoding='utf-8').writelines(out)
print("  deposit_addresses removed")
PY

echo "=== yaml sanity ==="
python3 - "$CFG" <<'PY'
import sys, yaml
w = yaml.safe_load(open(sys.argv[1], encoding='utf-8'))['wallet']
for k in ('deposit_contract','win_deposit_contract','deposit_address','deposit_addresses','usdt_contract','rpc_url'):
    print(f"  wallet.{k} = {w.get(k)!r}")
PY

echo "=== restart & verify ==="
sudo systemctl restart aix
sleep 6
systemctl is-active aix

echo "=== one scan cycle, then check nothing back-credited ==="
curl -fsS --max-time 20 http://127.0.0.1:9000/api/admin_dhb/deposit_only || echo "(usdt trigger failed)"
echo
curl -fsS --max-time 20 http://127.0.0.1:9000/api/admin_dhb/deposit_only_win || echo "(win trigger failed)"
echo
sleep 12
mysql -u"$DBUSER" -N -e "SELECT 'recharges_total', COUNT(*), IFNULL(SUM(amount),0) FROM ${DBNAME}.recharges"
mysql -u"$DBUSER" -N -e "SELECT 'created_last_10min', COUNT(*) FROM ${DBNAME}.recharges WHERE created_time > NOW() - INTERVAL 10 MINUTE"
journalctl -u aix -n 60 --no-pager | grep -iE 'depositOnly synchronized|deposit cursor|error' | tail -8 || true

echo SWITCH_OK
