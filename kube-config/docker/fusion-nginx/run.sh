#!/bin/bash
set -e

/etc/nginx/nginx.conf.sh > /etc/nginx/nginx.conf
exec nginx
