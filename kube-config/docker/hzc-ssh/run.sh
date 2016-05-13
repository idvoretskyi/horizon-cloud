#!/bin/sh
set -e

exec /hzc-ssh \
    -host-key "$HOST_KEY" \
    -listen "$LISTEN" \
    -api-server "$API_SERVER" \
    -api-server-secret "$API_SERVER_SECRET"
