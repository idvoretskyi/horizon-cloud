#!/bin/sh
set -e

cat /secrets/wildcard-ssl/crt /secrets/wildcard-ssl/crt-bundle \
    > /tmp/combined-crt

exec su -s /bin/sh daemon -c 'exec /hzc-http \
    --listen :8080 \
    --listen_tls :4433 \
    --tls_cert /tmp/combined-crt \
    --tls_key /secrets/wildcard-ssl/key'
