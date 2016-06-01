#!/bin/sh
set -e

exec su -s /bin/sh daemon -c 'exec /hzc-http --listen :8000'
