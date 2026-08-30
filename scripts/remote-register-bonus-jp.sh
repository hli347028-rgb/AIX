#!/bin/bash
# Install the new backend binary on JP and turn on the 5000 register bonus.
# The bonus lives in settings.system_config so it stays host-specific.
set -eu

BONUS=${BONUS:-5000}
TS=$(date +%Y%m%d%H%M%S)
BACKUP_DIR="$HOME/aix-backups"
mkdir -p "$BACKUP_DIR"

CFG=/opt/aix/configs/config.yaml
DB=$(grep -m1 'dbname:' "$CFG" | awk '{print $2}')
DBUSER=$(grep -m1 'user:' "$CFG" | awk '{print $2}')
DBPASS=$(grep -m1 'password:' "$CFG" | awk '{print $2}')
MYSQL="mysql -u$DBUSER -p$DBPASS $DB"
echo "db=$DB bonus=$BONUS"

echo "=== set register_bonus in settings.system_config ==="
$MYSQL -N -r -e "SELECT value FROM settings WHERE \`key\`='system_config'" > /tmp/sysconf.json
cp /tmp/sysconf.json "${BACKUP_DIR}/system_config.${TS}.json"
python3 - "$BONUS" <<'PY'
import json, sys
cfg = json.loads(open('/tmp/sysconf.json').read().strip())
cfg['register_bonus'] = sys.argv[1]
val = json.dumps(cfg, ensure_ascii=False)
sql = "UPDATE settings SET value=%s WHERE `key`='system_config';" % json.dumps(val)
open('/tmp/sysconf.sql', 'w').write(sql)
print('register_bonus ->', cfg['register_bonus'])
PY
$MYSQL < /tmp/sysconf.sql

echo "=== restart aix ==="
sudo systemctl restart aix
sleep 8
systemctl is-active aix

echo "=== verify ==="
$MYSQL -N -r -e "SELECT value FROM settings WHERE \`key\`='system_config'" \
  | python3 -c "import sys,json; print('register_bonus =', json.load(sys.stdin).get('register_bonus'))"
ls -la /opt/aix/aix-server

echo REGISTER_BONUS_JP_OK
