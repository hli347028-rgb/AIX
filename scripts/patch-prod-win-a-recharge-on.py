#!/usr/bin/env python3
from pathlib import Path

p = Path("/opt/aix/configs/config.yaml")
lines = p.read_text().splitlines()
out = []
for line in lines:
    if line.startswith("  win_a_recharge_enabled:"):
        out.append("  win_a_recharge_enabled: true")
        continue
    out.append(line)
if not any(l.startswith("  win_a_recharge_enabled:") for l in out):
    for i, line in enumerate(out):
        if line.startswith("  win_a_deposit_contract:"):
            out.insert(i + 1, "  win_a_recharge_enabled: true")
            break
p.write_text("\n".join(out) + "\n")
print("win_a_recharge_enabled=true")
