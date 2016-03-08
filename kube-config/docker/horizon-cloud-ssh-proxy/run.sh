#!/bin/sh
set -e

exec /horizon-cloud-ssh-proxy \
    -client-key "$CLIENT_KEY" \
    -host-key "$HOST_KEY" \
    -listen "$LISTEN" \
    -api-server "$API_SERVER" \
    -api-server-secret "$API_SERVER_SECRET"
