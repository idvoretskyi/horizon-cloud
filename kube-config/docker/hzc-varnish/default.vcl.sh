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

sub vcl_backend_response {
    // TODO: Does this need to be filtered on status, method, or Vary header?
    // Caching everything for a short time gives us some weak protection for our backend.
    set beresp.ttl = 1s;
}

sub vcl_synth {
    if (resp.status == 850) {
        set resp.http.Location = req.http.x-redir;
        set resp.status = 302;
        return (deliver);
    }
}

sub vcl_deliver {
    set resp.http.Strict-Transport-Security =
        "max-age=10886400; includeSubDomains; preload";
    if (resp.status >= 200 && resp.status < 500) {
        // TODO: Add a longer s-maxage header to this on the order of minutes
        // when CDN invalidation is implemented
        set resp.http.Cache-Control = "public,max-age=5";
    }
}

EOF
