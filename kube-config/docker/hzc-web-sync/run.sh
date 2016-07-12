#!/bin/sh
set -eu

cd /hzc-web-sync
exec node dist/main.js
