#!/bin/bash
set -e

# Disabling cgo avoids linking to libc for DNS resolution support, and thus makes
# a truly static binary.

CGO_ENABLED=0 go build -o fusion-ops-http-proxy \
    github.com/rethinkdb/fusion-ops/cmd/fusion-ops-http-proxy
