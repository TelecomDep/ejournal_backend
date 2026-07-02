#!/bin/sh
set -e

export DOMAIN="${DOMAIN:-lms.signal.qlabs.pro}"
envsubst '$DOMAIN' < /etc/nginx/nginx.conf.template > /etc/nginx/nginx.conf

( while :; do sleep 6h; nginx -s reload; done ) &