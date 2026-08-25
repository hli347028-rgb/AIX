#!/bin/bash
set -euo pipefail
python3 <<'PY'
from pathlib import Path
import re
p = Path("/opt/aix/configs/config.yaml")
text = p.read_text()
text = re.sub(
    r'^  usdt_contract:.*$',
    '  usdt_contract: "0x926632975149221891f1b9B56Efd125Dfe90ba2f"  # EOEO USDT',
    text,
    count=1,
    flags=re.M,
)
text = re.sub(
    r'^  sdt_contract:.*$',
    '  sdt_contract: "0x314D550572a0fA001B465a9EBc1dd04D834a0688"',
    text,
    count=1,
    flags=re.M,
)
p.write_text(text)
print("contracts fixed")
PY
grep -E '^  usdt_contract|^  sdt_contract' /opt/aix/configs/config.yaml
sudo systemctl restart aix
sleep 6
systemctl is-active aix
