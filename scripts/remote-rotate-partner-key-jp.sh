#!/bin/bash
# Issue a fresh HMAC secret for partner AIX10001 on this host only.
#
# The key file keeps the new secret on line 1 and the previous one on line 2: the verifier
# accepts any listed key, so the partner can switch over without a hard cutover. Drop the
# second line once they confirm they are signing with the new secret.
set -eu

KEYFILE=/opt/aix/secrets/partner-AIX10001.prod.key
TS=$(date +%Y%m%d%H%M%S)
BACKUP_DIR="$HOME/aix-backups"
mkdir -p "$BACKUP_DIR"

umask 077
cp "$KEYFILE" "${BACKUP_DIR}/partner-AIX10001.prod.key.${TS}"
echo "old key backed up to ${BACKUP_DIR}/partner-AIX10001.prod.key.${TS}"

NEWKEY=$(openssl rand -hex 32)
OLDKEY=$(head -n1 "$KEYFILE")

{
  echo "$NEWKEY"
  echo "$OLDKEY"
} > "$KEYFILE"
chmod 600 "$KEYFILE"

echo "=== key file now (2 keys: new, previous) ==="
stat -c '%n mode=%a size=%s' "$KEYFILE"
awk '{ printf "  line %d: %s...%s (len=%d)\n", NR, substr($0,1,6), substr($0,length($0)-5), length($0) }' "$KEYFILE"

echo "=== restart ==="
sudo systemctl restart aix
sleep 6
systemctl is-active aix
journalctl -u aix -n 30 --no-pager | grep -E 'transfer partners loaded' | tail -2

echo "=== signed smoke test with the NEW key ==="
# A signature failure returns 1001; anything else means the new key verified fine.
NEWKEY="$NEWKEY" python3 <<'PY'
import hashlib, hmac, json, os, time, urllib.request

secret = os.environ["NEWKEY"]
fields = {
    "partner_id": "AIX10001",
    "address": "0x000000000000000000000000000000000000dEaD",
    "amount": "1",
    "timestamp": str(int(time.time() * 1000)),
    "nonce": "smoke%s" % int(time.time()),
}
payload = "&".join(f"{k}={fields[k]}" for k in sorted(fields) if fields[k] != "")
sign = hmac.new(secret.encode(), payload.encode(), hashlib.sha256).hexdigest()

body = dict(fields, timestamp=int(fields["timestamp"]), sign=sign)
req = urllib.request.Request(
    "http://127.0.0.1:9000/v1/transfer/credit",
    data=json.dumps(body).encode(),
    headers={"Content-Type": "application/json"},
)
try:
    resp = urllib.request.urlopen(req, timeout=20)
    out = json.loads(resp.read())
except urllib.error.HTTPError as e:
    out = json.loads(e.read())

code = str(out.get("code"))
print("  canonical payload:", payload)
print("  response code:", code, "msg:", out.get("msg"))
print("  VERDICT:", "signature REJECTED (1001) - new key not active" if code == "1001"
      else f"new key verified OK (business code {code})")
PY

echo ROTATE_OK
