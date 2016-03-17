#!/bin/sh
set -e

cat /secrets/hzcio-ssl/hzc-crt /secrets/hzcio-ssl/hzc-crt-bundle \
    > /tmp/hzc-combined-crt

exec /hzc-http \
    --listen_tls :443 \
    --tls_cert /tmp/hzc-combined-crt \
    --tls_key /secrets/hzcio-ssl/hzc-key
