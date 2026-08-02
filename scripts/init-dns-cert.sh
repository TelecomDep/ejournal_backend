#!/bin/bash
set -e

DOMAIN="${1:-lms.signal.qlabs.pro}"
EMAIL="${2:-rpetrov2006mail.ru@gmail.com}"
HTPASSWD_FILE="./nginx_htpasswd"

echo "======================================================="
echo " SSL Certbot DNS-01 Challenge Setup for $DOMAIN"
echo "======================================================="

# 1. Создание файла авторизции Nginx .htpasswd (если его еще нет)
if [ ! -f "$HTPASSWD_FILE" ]; then
    echo "### Создание файла авторизации администраторов Nginx ($HTPASSWD_FILE)..."
    read -p "Введите имя администратора для Nginx (по умолчанию: admin): " ADMIN_USER
    ADMIN_USER="${ADMIN_USER:-admin}"
    
    read -sp "Введите пароль администратора: " ADMIN_PASS
    echo ""
    
    if [ -z "$ADMIN_PASS" ]; then
        echo "Пароль не может быть пустым!"
        exit 1
    fi

    # Генерация htpasswd с помощью Docker alpine openssl
    docker run --rm alpine sh -c "apk add --no-cache apache2-utils >/dev/null 2>&1 && htpasswd -nb '$ADMIN_USER' '$ADMIN_PASS'" > "$HTPASSWD_FILE"
    echo "Файл $HTPASSWD_FILE успешно создан для пользователя '$ADMIN_USER'."
else
    echo "### Файл $HTPASSWD_FILE уже существует."
fi

# 2. Выпуск SSL сертификата через Certbot DNS-01
echo ""
echo "### Запуск Certbot в режиме DNS-01 challenge..."
echo "ВАЖНО: Certbot сгенерирует TXT-запись."
echo "Вам нужно будет добавить её на FreeDNS (afraid.org) под именем:"
echo "   _acme-challenge.$DOMAIN"
echo "И нажать Enter ПОСЛЕ того как добавите запись."
echo ""

docker compose run --rm -it --entrypoint "\
  certbot certonly --manual --preferred-challenges dns \
    --email '$EMAIL' -d '$DOMAIN' \
    --rsa-key-size 2048 --agree-tos --no-eff-email" certbot

echo ""
echo "### Перезапуск Nginx для применения SSL сертификата..."
docker compose exec web nginx -s reload || docker compose up -d web

echo "======================================================="
echo " Успешно! Сертификат выпущен."
echo " Доступ к системе: https://$DOMAIN"
echo " Админ-панели (Grafana/Prometheus/Azimutt) защищены паролем."
echo "======================================================="
