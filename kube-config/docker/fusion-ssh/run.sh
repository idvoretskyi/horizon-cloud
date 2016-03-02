#!/bin/bash
set -e

mkdir -p /var/run/sshd
ssh-keygen -A
exec /usr/sbin/sshd -D -e
