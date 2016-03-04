#!/bin/bash
set -e

mkdir -p /var/run/sshd
ssh-keygen -A
rmdir /data/lost+found || true
chown -R fusion:fusion /data
exec /usr/sbin/sshd -D -e
