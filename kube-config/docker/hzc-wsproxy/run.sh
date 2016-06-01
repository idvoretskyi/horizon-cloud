#!/bin/sh
set -eu

cat /secrets/wildcard-ssl/crt /secrets/wildcard-ssl/crt-bundle > /tmp/certs
/stunnel.conf.bash > /stunnel.conf

exec stunnel /stunnel.conf
