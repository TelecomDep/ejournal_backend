#!/bin/sh
set -e

rm -f /etc/nginx/conf.d/default.conf

export DOMAIN="${DOMAIN:-ezachetka.ru}"
export SUPPORT_EMAIL="${SUPPORT_EMAIL:-support@${DOMAIN}}"
export ABUSE_EMAIL="${ABUSE_EMAIL:-${SUPPORT_EMAIL}}"
export FEEDBACK_TELEGRAM="${FEEDBACK_TELEGRAM:-}"
export ORGANIZATION_NAME="${ORGANIZATION_NAME:-Электронный журнал ${DOMAIN}}"
export DISPUTE_TIMEFRAME_DAYS="${DISPUTE_TIMEFRAME_DAYS:-7}"

# 1. Ensure SSL certificates directory and self-signed certificate if Let's Encrypt cert is not present
SSL_DIR="/etc/letsencrypt/live/${DOMAIN}"
if [ ! -f "${SSL_DIR}/fullchain.pem" ] || [ ! -f "${SSL_DIR}/privkey.pem" ]; then
    echo "[Nginx Setup] Certificates for ${DOMAIN} not found in ${SSL_DIR}."
    mkdir -p "${SSL_DIR}"
    if command -v openssl >/dev/null 2>&1; then
        openssl req -x509 -nodes -newkey rsa:2048 -days 365 \
            -keyout "${SSL_DIR}/privkey.pem" \
            -out "${SSL_DIR}/fullchain.pem" \
            -subj "/CN=${DOMAIN}" 2>/dev/null || true
    fi
fi

# 2. Ensure .htpasswd file exists for Admin Area protection
HTPASSWD_FILE="/etc/nginx/.htpasswd"
if [ ! -s "${HTPASSWD_FILE}" ]; then
    echo "[Nginx Setup] Creating default .htpasswd for Admin Area (user: admin, pass: admin)..."
    echo 'admin:{SHA}0DPiKuNIrrVmD8IUCuw1hQxNqZc=' > "${HTPASSWD_FILE}"
fi

# 3. Generate dynamic frontend runtime configuration from environment variables
echo "[Nginx Setup] Generating runtime frontend configuration..."
cat <<EOF > /usr/share/nginx/html/runtime-config.js
window.__APP_CONFIG__ = {
  DOMAIN: "${DOMAIN}",
  SUPPORT_EMAIL: "${SUPPORT_EMAIL}",
  ABUSE_EMAIL: "${ABUSE_EMAIL}",
  FEEDBACK_TELEGRAM: "${FEEDBACK_TELEGRAM}",
  ORGANIZATION_NAME: "${ORGANIZATION_NAME}",
  DISPUTE_TIMEFRAME_DAYS: "${DISPUTE_TIMEFRAME_DAYS}"
};
EOF

# 4. Process nginx.conf.template
envsubst '$DOMAIN' < /etc/nginx/nginx.conf.template > /etc/nginx/nginx.conf

# 5. Background auto-reload to pick up updated Let's Encrypt certs
( while :; do sleep 6h; nginx -s reload 2>/dev/null || true; done ) &