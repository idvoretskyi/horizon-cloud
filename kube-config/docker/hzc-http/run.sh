#!/bin/sh
set -e

cat /secrets/hzcio-ssl/hzc-crt /secrets/hzcio-ssl/hzc-crt-bundle \
    > /tmp/hzc-combined-crt

exec su -s /bin/sh daemon -c 'exec /hzc-http \
    --listen :8080 \
    --listen_tls :4433 \
    --tls_cert /tmp/hzc-combined-crt \
    --tls_key /secrets/hzcio-ssl/hzc-key'
