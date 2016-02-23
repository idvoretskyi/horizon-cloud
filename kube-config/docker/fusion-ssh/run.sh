#!/bin/bash
set -e

tar -xzf /secrets/ssh-key-tarball -C /etc/ssh

exec /usr/sbin/sshd -D
