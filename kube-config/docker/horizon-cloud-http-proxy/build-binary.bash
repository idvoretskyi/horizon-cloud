#!/bin/bash
set -e

# Disabling cgo avoids linking to libc for DNS resolution support, and thus makes
# a truly static binary.

CGO_ENABLED=0 go build -o horizon-cloud-http-proxy \
    github.com/rethinkdb/horizon-cloud/cmd/horizon-cloud-http-proxy
