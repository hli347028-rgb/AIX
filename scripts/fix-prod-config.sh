#!/bin/bash
set -euo pipefail
CFG=/opt/aix/configs/config.yaml
python3 - <<'PY'
from pathlib import Path
p = Path("/opt/aix/configs/config.yaml")
lines = p.read_text().splitlines()
out = []
skip_next_sdt_decimals = False
for line in lines:
    if line.startswith("  sdt_contract:"):
        out.append('  sdt_contract: "0x4277289b67cc85C98067b653B91A071D0f4CB3b2"')
        out.append("  sdt_decimals: 18")
        skip_next_sdt_decimals = True
        continue
    if skip_next_sdt_decimals and line.strip().startswith("sdt_decimals:"):
        skip_next_sdt_decimals = False
        continue
    skip_next_sdt_decimals = False
    out.append(line)
text = "\n".join(out) + "\n"
if 'sdt_contract:' not in text:
    fixed = []
    for line in out:
        fixed.append(line)
        if line.strip() == "win_decimals: 18":
            fixed.append('  sdt_contract: "0x4277289b67cc85C98067b653B91A071D0f4CB3b2"')
            fixed.append("  sdt_decimals: 18")
    text = "\n".join(fixed) + "\n"
p.write_text(text)
print("config fixed")
PY
grep -n "sdt_\|win_decimals" "$CFG" | head -10
