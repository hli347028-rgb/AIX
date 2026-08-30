#!/usr/bin/env python3
"""Insert admin_sub_accounts into /opt/aix/configs/config.yaml if missing. Preserve secrets."""
from pathlib import Path

path = Path("/opt/aix/configs/config.yaml")
text = path.read_text(encoding="utf-8")
if "admin_sub_accounts:" in text:
    print("admin_sub_accounts already present; skip insert")
    raise SystemExit(0)

block = """  admin_sub_accounts:
    - account: user1
      password: user1
      modules: []
    - account: user2
      password: user2
      modules: []
    - account: user3
      password: user3
      modules: []
"""

needle = None
for line in text.splitlines(True):
    if line.lstrip().startswith("admin_password:"):
        needle = line
        break
if not needle:
    raise SystemExit("admin_password not found")

# Insert after admin_password line
idx = text.find(needle)
if idx < 0:
    raise SystemExit("admin_password line not found")
end = idx + len(needle)
path.write_text(text[:end] + block + text[end:], encoding="utf-8")
print("INSERTED_ADMIN_SUB_ACCOUNTS_OK")
