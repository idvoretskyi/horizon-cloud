#!/bin/bash
set -e

cat <<EOF
foreground = yes
output = /dev/null
syslog = no
pid =
setuid = daemon
debug = 2

[wsproxy]
accept = 443
cert = /tmp/certs
key = /secrets/wildcard-ssl/key
connect = $TARGET
EOF
