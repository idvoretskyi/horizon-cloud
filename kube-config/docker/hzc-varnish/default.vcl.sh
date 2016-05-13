#!/bin/sh
set -eu

BUCKET="$(cat /secrets/names/storage-bucket)"

cat <<EOF
vcl 4.0;

backend storage {
    .host = "storage.googleapis.com";
    .port = "80";
}

sub vcl_recv {
    if (req.url == "/ebaefa90-3c6e-4eb4-b8d3-9e2d53aec696") {
        # Health check for Google load balancer
        return (synth(200, "OK"));
    }

    if (req.http.X-Forwarded-Proto == "http") {
        set req.http.x-redir = "https://" + req.http.host + req.url;
        return (synth(850, "Moved permanently"));
    }

    if (req.url ~ "/\$") {
        set req.url = req.url + "index.html";
    }
    set req.url = "/$BUCKET/domains/" + req.http.host + req.url;
    set req.http.host = "storage.googleapis.com";
    return (hash);
}

sub vcl_synth {
    if (resp.status == 850) {
        set resp.http.Location = req.http.x-redir;
        set resp.status = 302;
        return (deliver);
    }
}
EOF
