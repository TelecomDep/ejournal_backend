#!/bin/sh
set -e

rm -f /etc/nginx/conf.d/default.conf

export DOMAIN="${DOMAIN:-lms.signal.qlabs.pro}"

# 1. Ensure SSL certificates directory and self-signed certificate if Let's Encrypt cert is not present
SSL_DIR="/etc/letsencrypt/live/${DOMAIN}"
if [ ! -f "${SSL_DIR}/fullchain.pem" ] || [ ! -f "${SSL_DIR}/privkey.pem" ]; then
    echo "[Nginx Setup] Certificates for ${DOMAIN} not found. Generating temporary self-signed certificate..."
    mkdir -p "${SSL_DIR}"
    openssl req -x509 -nodes -newkey rsa:2048 -days 365 \
        -keyout "${SSL_DIR}/privkey.pem" \
        -out "${SSL_DIR}/fullchain.pem" \
        -subj "/CN=${DOMAIN}" 2>/dev/null || true
fi

# 2. Ensure .htpasswd file exists for Admin Area protection
HTPASSWD_FILE="/etc/nginx/.htpasswd"
if [ ! -s "${HTPASSWD_FILE}" ]; then
    echo "[Nginx Setup] Creating default .htpasswd for Admin Area (user: admin, pass: admin)..."
    echo 'admin:{SHA}0DPiKuNIrrVmD8IUCuw1hQxNqZc=' > "${HTPASSWD_FILE}"
fi

# 3. Process nginx.conf.template
envsubst '$DOMAIN' < /etc/nginx/nginx.conf.template > /etc/nginx/nginx.conf

# 4. Background auto-reload to pick up updated Let's Encrypt certs
( while :; do sleep 6h; nginx -s reload 2>/dev/null || true; done ) &