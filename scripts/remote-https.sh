#!/bin/bash
set -euo pipefail

sudo mkdir -p /var/www/html
sudo cp /opt/aix/scripts/aix.nginx.conf /etc/nginx/sites-available/aix
sudo ln -sfn /etc/nginx/sites-available/aix /etc/nginx/sites-enabled/aix
sudo rm -f /etc/nginx/sites-enabled/default
sudo nginx -t
sudo systemctl reload nginx

echo "=== requesting certificate ==="
sudo certbot certonly --nginx \
  -d aixai.pro -d www.aixai.pro \
  --non-interactive --agree-tos \
  --register-unsafely-without-email \
  --keep-until-expiring

sudo cp /opt/aix/scripts/aix.nginx.ssl.conf /etc/nginx/sites-available/aix
sudo nginx -t
sudo systemctl reload nginx

echo "=== certbot timer ==="
sudo systemctl enable --now certbot.timer 2>/dev/null || true
systemctl is-active certbot.timer || true

echo HTTPS_OK
