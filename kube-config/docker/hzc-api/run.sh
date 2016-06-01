#!/bin/sh
set -e

exec su -s /bin/sh hzc -c 'exec /hzc-api'
