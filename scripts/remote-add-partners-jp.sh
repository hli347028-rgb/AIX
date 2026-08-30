#!/bin/bash
# Append the transfer_partners block to the JP server config.yaml.
# The partner secret is delivered separately into /opt/aix/secrets (never inlined here).
set -eu

CFG=/opt/aix/configs/config.yaml
TS=$(date +%Y%m%d%H%M%S)
BACKUP_DIR="$HOME/aix-backups"
mkdir -p "$BACKUP_DIR"

cp "$CFG" "${BACKUP_DIR}/config.yaml.${TS}"
echo "config backed up to ${BACKUP_DIR}/config.yaml.${TS}"

if grep -q '^transfer_partners:' "$CFG"; then
  echo "transfer_partners already present, leaving config untouched"
else
  printf '\n# 合作方转账加款接口 POST /v1/transfer/credit（HMAC-SHA256 双向签名）\ntransfer_partners:\n  timestamp_skew: 300s\n  ip_rate_limit_per_sec: 20\n  partners:\n    - partner_id: AIX10001\n      enabled: true\n      secret_keys_file: /opt/aix/secrets/partner-AIX10001.prod.key\n      min_amount: ""\n      max_amount: ""\n      daily_limit: ""\n      rate_limit_per_sec: 10\n' >> "$CFG"
  echo "transfer_partners appended"
fi

echo "=== key file presence (content never printed) ==="
if [ -s /opt/aix/secrets/partner-AIX10001.prod.key ]; then
  stat -c '%n mode=%a size=%s' /opt/aix/secrets/partner-AIX10001.prod.key
else
  echo "MISSING: /opt/aix/secrets/partner-AIX10001.prod.key"
fi

echo "=== restart & verify ==="
sudo systemctl restart aix
sleep 6
systemctl is-active aix
journalctl -u aix -n 40 --no-pager | grep -E 'transfer partners|transfer_partners' | tail -5

echo PARTNERS_OK
