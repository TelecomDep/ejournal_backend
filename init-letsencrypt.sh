#!/bin/bash
set -e

if [ -f .env ]; then
  set -a
  . ./.env
  set +a
fi

DOMAIN="${DOMAIN:-lms.signal.qlabs.pro}"
EMAIL="rpetrov2006mail.ru@gmail.com"
DATA_PATH="./certbot"

if [ -d "$DATA_PATH/conf/live/$DOMAIN" ]; then
  echo "Existing certificate data found for $DOMAIN, skipping bootstrap."
  echo "Delete $DATA_PATH/conf/live/$DOMAIN to force re-bootstrap."
  exit 0
fi

echo "### Creating temporary self-signed certificate for $DOMAIN ..."
mkdir -p "$DATA_PATH/conf/live/$DOMAIN"
docker compose run --rm --entrypoint "\
  openssl req -x509 -nodes -newkey rsa:2048 -days 1 \
    -keyout '/etc/letsencrypt/live/$DOMAIN/privkey.pem' \
    -out '/etc/letsencrypt/live/$DOMAIN/fullchain.pem' \
    -subj '/CN=localhost'" certbot

echo "### Starting nginx with the temporary certificate ..."
docker compose up -d web

echo "### Deleting temporary certificate ..."
docker compose run --rm --entrypoint "\
  rm -rf /etc/letsencrypt/live/$DOMAIN /etc/letsencrypt/archive/$DOMAIN /etc/letsencrypt/renewal/$DOMAIN.conf" certbot

echo "### Requesting real Let's Encrypt certificate for $DOMAIN ..."
docker compose run --rm --entrypoint "\
  certbot certonly --webroot -w /var/www/certbot \
    --email $EMAIL -d $DOMAIN \
    --rsa-key-size 2048 --agree-tos --no-eff-email" certbot

echo "### Reloading nginx with the real certificate ..."
docker compose exec web nginx -s reload

echo "### Starting certbot renewal service ..."
docker compose up -d certbot

echo "Done. https://$DOMAIN should now be serving a valid Let's Encrypt certificate."