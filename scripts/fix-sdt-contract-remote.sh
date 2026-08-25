#!/bin/bash
set -euo pipefail
CFG="/opt/aix/configs/config.yaml"
NEW="0x314D550572a0fA001B465a9EBc1dd04D834a0688"
sed -i "s/sdt_contract: .*/sdt_contract: \"${NEW}\"/" "$CFG"
grep sdt_contract "$CFG"
sudo systemctl restart aix
sleep 6
systemctl is-active aix
