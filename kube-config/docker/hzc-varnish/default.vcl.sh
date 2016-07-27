#!/bin/sh
set -eu

BUCKET="$(cat /secrets/names/storage-bucket)"
DOMAIN="$(cat /secrets/names/domain)"

cat <<EOF
vcl 4.0;

// The first backend is the default one. By having the default be a dummy address
// that is instantly rejected, we make sure we are explicitly using a backend in
// every code path that needs one.
backend dummy {
    .host = "127.0.0.1";
    .port = "65535";
}

backend storage {
    .host = "storage.googleapis.com";
    .port = "80";
}

backend api {
    .host = "hzc-api";
    .port = "80";
}

backend hzchttp {
    .host = "hzc-http.$DOMAIN";
    .port = "80";
}

sub vcl_recv {
    // Health check for Google load balancer and Kubernetes
    if (req.url == "/ebaefa90-3c6e-4eb4-b8d3-9e2d53aec696") {
        return (synth(200, "OK"));
    }

    // Always use https
    if (req.http.X-Forwarded-Proto == "http") {
        set req.http.x-redir = "https://" + req.http.host + req.url;
        return (synth(850, "Moved"));
    }

    // Update server points at a bucket by that domain name directly
    if (req.http.host == "update.$DOMAIN") {
        set req.url = "/" + req.http.host + req.url;
        set req.http.host = "storage.googleapis.com";
        set req.backend_hint = storage;
        return (hash);
    }

    // API server is direct and uncached
    if (req.http.host == "api.$DOMAIN") {
        set req.backend_hint = api;
        return (pass);
    }

    // Redirect Horizon requests to a subdomain.
    if (req.url ~ "^/horizon(/|\$)") {
        set req.http.x-redir = "https://horizon.$DOMAIN/" + req.http.host + req.url;
        return (synth(850, "Moved"));
    }

    // All other requests go to hzc-http, to be proxied to horizon or GCS
    set req.url = "/" + req.http.host + req.url;
    set req.backend_hint = hzchttp;
    return (hash);
}

sub vcl_backend_response {
    if (bereq.method == "GET") {
        // Caching everything for a short time gives us some weak protection for our backend.
        // TODO: Does this need to be filtered on response code or Vary header?
        set beresp.ttl = 1s;
    }
}

sub vcl_synth {
    if (resp.status == 850) {
        set resp.http.Location = req.http.x-redir;
        set resp.status = 307;
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
