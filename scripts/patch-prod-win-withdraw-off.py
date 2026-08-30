#!/usr/bin/env python3
from pathlib import Path

p = Path("/opt/aix/configs/config.yaml")
lines = p.read_text().splitlines()
out = []
inserted = any("win_withdraw_enabled:" in l for l in lines)
for line in lines:
    if line.startswith("  withdraw_payout_enabled:") and not inserted:
        out.append("  win_withdraw_enabled: false")
        inserted = True
    if line.startswith("  win_withdraw_enabled:"):
        out.append("  win_withdraw_enabled: false")
        continue
    out.append(line)
if not inserted:
    for i, line in enumerate(out):
        if line.startswith("  withdraw_payout_enabled:"):
            out.insert(i, "  win_withdraw_enabled: false")
            break
p.write_text("\n".join(out) + "\n")
print("win_withdraw_enabled=false")
