#!/usr/bin/env python3
from pathlib import Path

p = Path("/opt/aix/configs/config.yaml")
text = p.read_text(encoding="utf-8")
for marker in ("\n# 合作方转账加款接口", "\ntransfer_partners:"):
    start = text.find(marker)
    if start >= 0:
        text = text[:start]
        break
block = """
# 合作方转账加款接口 POST /v1/transfer/credit（HMAC-SHA256 双向签名）
transfer_partners:
  timestamp_skew: 300s
  ip_rate_limit_per_sec: 20
  partners:
    - partner_id: AIX10001
      enabled: true
      secret_keys_file: /opt/aix/secrets/partner-AIX10001.prod.key
      min_amount: ""
      max_amount: ""
      daily_limit: ""
      rate_limit_per_sec: 10
"""
if not text.endswith("\n"):
    text += "\n"
p.write_text(text + block, encoding="utf-8")
print("OK")
print(p.read_text(encoding="utf-8").split("transfer_partners:", 1)[-1])
