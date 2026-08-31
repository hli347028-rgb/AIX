#!/bin/bash
set -euo pipefail
mysql -uroot -proot aix -N -e "SELECT value FROM settings WHERE \`key\`='system_config';" 2>/dev/null > /tmp/system_config.json
python3 <<'PY'
import json
from pathlib import Path
raw = Path('/tmp/system_config.json').read_text(encoding='utf-8').strip()
d = json.loads(raw)
bonus = d.get('register_bonus', None)
print('has_register_bonus_key=', 'register_bonus' in d)
print('register_bonus=', repr(bonus))
if bonus is None:
    print('result=missing (code default 0 applies)')
elif str(bonus).strip() in ('', '0'):
    print('result=disabled (no gift)')
else:
    print('result=ENABLED gift amount=', bonus)
PY
