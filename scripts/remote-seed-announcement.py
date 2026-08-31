#!/usr/bin/env python3
"""Seed the legacy WIN-AIX announcement into production DB. Does not print credentials."""
import re
import subprocess
import sys

CFG = "/opt/aix/configs/config.yaml"
SQL = "/tmp/seed-announcement.sql"

text = open(CFG, "r", encoding="utf-8").read()
m = re.search(
    r"database:\s*\n(?:\s+\w+:.*\n)*?\s+password:\s*[\"']?([^\"'\n]+)",
    text,
)
if not m:
    print("NO_DB_PASS", file=sys.stderr)
    sys.exit(1)
password = m.group(1).strip()
env = dict(**{k: v for k, v in __import__("os").environ.items()}, MYSQL_PWD=password)

def mysql(*args, input_text=None):
    cmd = ["mysql", "--user=root", "--database=aix", "--default-character-set=utf8mb4", *args]
    return subprocess.run(cmd, input=input_text, text=True, capture_output=True, env=env, check=False)

# ensure table
r = mysql("-e", "SHOW TABLES LIKE 'announcements';")
if r.returncode != 0:
    print(r.stderr, file=sys.stderr)
    sys.exit(r.returncode)
print(r.stdout.strip() or "announcements_table_missing")

# create if missing
r = mysql("-N", "-e", "SELECT COUNT(*) FROM announcements;")
if r.returncode != 0:
    create_sql = """
CREATE TABLE IF NOT EXISTS announcements (
  id bigint NOT NULL AUTO_INCREMENT PRIMARY KEY,
  title varchar(256) NOT NULL,
  content longtext NOT NULL,
  status int NOT NULL DEFAULT 1,
  created_time datetime(3) NULL,
  updated_time datetime(3) NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
"""
    r2 = mysql("-e", create_sql)
    if r2.returncode != 0:
        print(r2.stderr, file=sys.stderr)
        sys.exit(r2.returncode)
    count = 0
else:
    count = int((r.stdout or "0").strip() or "0")
print(f"count_before={count}")

# username column
r = mysql("-N", "-e", "SHOW COLUMNS FROM users LIKE 'username';")
if r.returncode == 0 and not (r.stdout or "").strip():
    r2 = mysql("-e", "ALTER TABLE users ADD COLUMN username varchar(64) NOT NULL DEFAULT '' AFTER invite_code;")
    print("USERNAME_COL_ADDED" if r2.returncode == 0 else r2.stderr)

if count == 0:
    r = mysql(
        "-e",
        "INSERT INTO announcements (title, content, status, created_time, updated_time) "
        "VALUES ('placeholder', 'placeholder', 1, '2026-08-30 22:00:00', '2026-08-30 22:00:00');",
    )
    if r.returncode != 0:
        print(r.stderr, file=sys.stderr)
        sys.exit(r.returncode)

with open(SQL, "r", encoding="utf-8") as f:
    seed = f.read()
r = mysql(input_text=seed)
if r.returncode != 0:
    print(r.stderr, file=sys.stderr)
    sys.exit(r.returncode)

r = mysql("-N", "-e", "SELECT id, CHAR_LENGTH(title), status, created_time FROM announcements;")
print(r.stdout.strip())
print("SEED_OK")
