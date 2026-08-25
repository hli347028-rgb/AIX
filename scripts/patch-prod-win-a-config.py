#!/usr/bin/env python3
from pathlib import Path

p = Path("/opt/aix/configs/config.yaml")
lines = p.read_text().splitlines()
out = []
for line in lines:
    if line.startswith("  win_deposit_contract:") and not any("win_a_deposit_contract:" in l for l in lines):
        out.append(line)
        out.append('  win_a_deposit_contract: "0xcaa39A8E23F5548AD85d9e2B9B21F63E99505040"')
        continue
    if line.startswith("  recharge_monitor_enabled:"):
        out.append("  recharge_monitor_enabled: false")
        continue
    if line.startswith("  win_price_oracle_enabled:"):
        out.append("  win_price_oracle_enabled: false")
        continue
    if line.startswith("  recharge_scan_queries_per_cycle:"):
        out.append("  recharge_scan_queries_per_cycle: 10")
        continue
    if line.startswith("  recharge_scan_query_interval_seconds:"):
        out.append("  recharge_scan_query_interval_seconds: 5")
        continue
    if line.startswith("  win_decimals:") and not any("win_a_contract:" in l for l in lines):
        out.append(line)
        out.append('  win_a_contract: "0x3dce8Ef2646e3082f1e4afbC26Cf747e4e238A8D"')
        out.append("  win_a_decimals: 18")
        continue
    out.append(line)
p.write_text("\n".join(out) + "\n")
print("config patched ok")
