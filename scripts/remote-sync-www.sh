#!/bin/bash
# Sync built frontend into /opt/aix/www and fix nginx-readable permissions.
# Usage on server:
#   rsync -a --delete --exclude admin /tmp/www-new/ /opt/aix/www/
#   bash /opt/aix/scripts/remote-sync-www.sh
set -euo pipefail

WWW_ROOT="${WWW_ROOT:-/opt/aix/www}"
ADMIN_SRC="${ADMIN_SRC:-/opt/aix/admin/dist}"

if [ ! -d "$WWW_ROOT" ]; then
  echo "WWW_ROOT not found: $WWW_ROOT" >&2
  exit 1
fi

if [ ! -f "$WWW_ROOT/admin/index.html" ] && [ -d "$ADMIN_SRC" ]; then
  echo "=== restore admin from $ADMIN_SRC ==="
  mkdir -p "$WWW_ROOT/admin"
  rsync -a "$ADMIN_SRC/" "$WWW_ROOT/admin/"
fi

echo "=== fix www perms ($WWW_ROOT) ==="
sudo chmod -R a+rX "$WWW_ROOT"

echo "=== verify index.html readable ==="
test -r "$WWW_ROOT/index.html" || { echo "index.html not readable" >&2; exit 1; }

if [ -f "$WWW_ROOT/admin/index.html" ]; then
  test -r "$WWW_ROOT/admin/index.html" || { echo "admin/index.html not readable" >&2; exit 1; }
fi

echo WWW_SYNC_OK
