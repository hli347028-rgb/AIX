#!/bin/bash
# Reset the JP database to a clean slate: schema stays, only the genesis address survives.
#
# Deliberately kept:
#   settings.deposit_only_cursor:*  -> scanner position on the shared deposit contracts.
#     Dropping these would rewind the cursor to 0 and re-walk 491 historical deposit
#     records on the next cron tick.
#   settings.system_config          -> admin config snapshot (limits, prices, sub-accounts).
set -eu

CFG=/opt/aix/configs/config.yaml
TS=$(date +%Y%m%d%H%M%S)
BACKUP_DIR="$HOME/aix-backups"
mkdir -p "$BACKUP_DIR"

DBUSER=$(grep -m1 -E '^\s+user:' "$CFG" | sed -E 's/.*user:\s*//; s/"//g')
DBPASS=$(grep -m1 -E '^\s+password:' "$CFG" | sed -E 's/.*password:\s*//; s/"//g')
DBNAME=$(grep -m1 -E '^\s+dbname:' "$CFG" | sed -E 's/.*dbname:\s*//; s/"//g')
export MYSQL_PWD="$DBPASS"

echo "=== full backup before wipe ==="
mysqldump -u"$DBUSER" --single-transaction --routines --events "$DBNAME" | gzip > "${BACKUP_DIR}/${DBNAME}-prewipe-${TS}.sql.gz"
ls -la "${BACKUP_DIR}/${DBNAME}-prewipe-${TS}.sql.gz"

echo "=== stop service so nothing writes mid-wipe ==="
sudo systemctl stop aix
sleep 2
systemctl is-active aix || true

echo "=== truncate business tables ==="
mysql -u"$DBUSER" "$DBNAME" <<'SQL'
SET FOREIGN_KEY_CHECKS = 0;
TRUNCATE TABLE orders;
TRUNCATE TABLE recharges;
TRUNCATE TABLE withdrawals;
TRUNCATE TABLE withdrawal_payouts;
TRUNCATE TABLE transfers;
TRUNCATE TABLE exchange_records;
TRUNCATE TABLE reward_logs;
TRUNCATE TABLE mgmt_rewards;
TRUNCATE TABLE settlement_batches;
TRUNCATE TABLE aix_prices;
TRUNCATE TABLE win_prices;
TRUNCATE TABLE announcements;
TRUNCATE TABLE admin_operation_logs;
TRUNCATE TABLE partner_nonces;

-- keep only the genesis root and clear the counters left by test orders
DELETE FROM users WHERE id <> 1;
UPDATE users SET
  usdt_recharge = 0, usdt_reward = 0, usdt_withdrawable = 0, aix_balance = 0,
  win_balance = 0, win_recharge_balance = 0, win_a_recharge_balance = 0,
  static_usdt_total = 0, overflow_direct = 0, overflow_reward = 0,
  small_area_perf = 0, large_area_perf = 0, team_perf = 0,
  mgmt_level = 0, mgmt_level_locked = 0,
  is_frozen = 0, frozen_at = NULL,
  is_zero_account = 0, zero_account_set_at = NULL, zero_account_reward_total = 0,
  is_community_subsidy = 0, community_subsidy_set_at = NULL, community_subsidy_total = 0
WHERE id = 1;
ALTER TABLE users AUTO_INCREMENT = 2;
SET FOREIGN_KEY_CHECKS = 1;
SQL
echo "  truncate done"

echo "=== settings kept ==="
mysql -u"$DBUSER" -N -e "SELECT \`key\` FROM ${DBNAME}.settings"

echo "=== start service ==="
sudo systemctl start aix
sleep 8
systemctl is-active aix

echo "=== row counts after wipe ==="
for t in users orders recharges withdrawals withdrawal_payouts transfers exchange_records reward_logs mgmt_rewards settlement_batches aix_prices win_prices settings announcements admin_operation_logs partner_nonces; do
  c=$(mysql -u"$DBUSER" -N -e "SELECT COUNT(*) FROM ${DBNAME}.${t}")
  echo "  ${t}: ${c}"
done

echo "=== surviving user ==="
mysql -u"$DBUSER" -N -e "SELECT id, address, role, status, team_perf, large_area_perf FROM ${DBNAME}.users"

echo "=== tables still present ==="
mysql -u"$DBUSER" -N -e "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='${DBNAME}'"

echo WIPE_OK
