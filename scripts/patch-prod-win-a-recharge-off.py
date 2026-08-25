#!/usr/bin/env python3
from pathlib import Path

p = Path("/opt/aix/configs/config.yaml")
lines = p.read_text().splitlines()
out = []
inserted = any("win_a_recharge_enabled:" in l for l in lines)
for line in lines:
    if line.startswith("  win_a_deposit_contract:") and not inserted:
        out.append(line)
        out.append("  win_a_recharge_enabled: false")
        inserted = True
        continue
    if line.startswith("  win_a_recharge_enabled:"):
        out.append("  win_a_recharge_enabled: false")
        continue
    out.append(line)
if not inserted:
    for i, line in enumerate(out):
        if line.startswith("  win_a_contract:"):
            out.insert(i, "  win_a_recharge_enabled: false")
            break
p.write_text("\n".join(out) + "\n")
print("win_a_recharge_enabled=false")
