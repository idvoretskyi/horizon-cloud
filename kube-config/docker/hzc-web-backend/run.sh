#!/bin/sh
set -eu

cd /hzc-web-backend
exec node dist/main.js
