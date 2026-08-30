#!/bin/bash
# Finish the AIX10001 key rotation: keep only the current secret (line 1) and drop the
# transitional shared one (line 2). Verified afterwards by signing with each key.
set -eu

KEYFILE=/opt/aix/secrets/partner-AIX10001.prod.key
TS=$(date +%Y%m%d%H%M%S)
BACKUP_DIR="$HOME/aix-backups"
mkdir -p "$BACKUP_DIR"
umask 077

cp "$KEYFILE" "${BACKUP_DIR}/partner-AIX10001.prod.key.${TS}"
echo "pre-change backup: ${BACKUP_DIR}/partner-AIX10001.prod.key.${TS}"

LINES=$(grep -cve '^\s*$' "$KEYFILE")
echo "keys before: ${LINES}"

CURRENT=$(head -n1 "$KEYFILE")
RETIRED=$(sed -n 2p "$KEYFILE")

printf '%s\n' "$CURRENT" > "$KEYFILE"
chmod 600 "$KEYFILE"

echo "=== key file now ==="
stat -c '%n mode=%a size=%s' "$KEYFILE"
awk '{ printf "  line %d: %s...%s (len=%d)\n", NR, substr($0,1,6), substr($0,length($0)-5), length($0) }' "$KEYFILE"

echo "=== restart ==="
sudo systemctl restart aix
sleep 6
systemctl is-active aix
journalctl -u aix -n 30 --no-pager | grep -E 'transfer partners loaded' | tail -2

echo "=== verify: current key accepted, retired key rejected ==="
CURRENT="$CURRENT" RETIRED="$RETIRED" python3 <<'PY'
import hashlib, hmac, json, os, time, urllib.request, urllib.error

def call(secret, label):
    fields = {
        "partner_id": "AIX10001",
        "address": "0x000000000000000000000000000000000000dEaD",
        "amount": "1",
        "timestamp": str(int(time.time() * 1000)),
        "nonce": "rot%s%s" % (label, int(time.time() * 1000) % 100000),
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
        out = json.loads(urllib.request.urlopen(req, timeout=20).read())
    except urllib.error.HTTPError as e:
        out = json.loads(e.read())
    return str(out.get("code")), out.get("msg")

code, msg = call(os.environ["CURRENT"], "cur")
print(f"  current key -> code={code} msg={msg}")
print("    ", "FAIL: current key rejected" if code == "1001" else "OK: signature accepted")

code2, msg2 = call(os.environ["RETIRED"], "old")
print(f"  retired key -> code={code2} msg={msg2}")
print("    ", "OK: retired key now rejected" if code2 == "1001" else "FAIL: retired key still accepted")
PY

echo DROP_OLD_OK
