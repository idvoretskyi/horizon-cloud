#!/bin/bash
set -e

cat <<EOF
user www-data;
worker_processes 4;
pid /run/nginx.pid;
daemon off;

events {
    worker_connections 768;
}

http {
    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout 65;
    include /etc/nginx/mime.types;
    default_type application/octet-stream;
    access_log /var/log/nginx/access.log;
    error_log /var/log/nginx/error.log;
    gzip on;
    gzip_disable "msie6";
    server {
        listen 80 default_server;
        listen [::]:80 default_server ipv6only=on;

        root /usr/share/nginx/html;
        index index.html index.htm;

        server_name localhost; # TODO

        location /fusion/ {
            proxy_pass http://$NGINX_CONNECT;
            # TODO: these can maybe be removed from this location.
            proxy_http_version 1.1;
            proxy_set_header Upgrade \$http_upgrade;
            proxy_set_header Connection "upgrade";
        }

        location = /fusion {
            proxy_pass http://$NGINX_CONNECT;
            proxy_http_version 1.1;
            proxy_set_header Upgrade \$http_upgrade;
            proxy_set_header Connection "upgrade";
        }

        location / {
            alias /data/current/;
        }
    }
}
EOF
