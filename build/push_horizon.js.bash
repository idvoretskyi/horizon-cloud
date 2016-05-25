#!/bin/bash
set -eu
cd "$(dirname "$0")"

TMPDIR=$(mktemp -d)
trap 'rm -rf $TMPDIR' EXIT

gsutil -h "Cache-Control:private,max-age=5" cp -a public-read \
    ../kube-config/docker/horizon/horizon/client/dist/horizon.js \
    "gs://`cat /secrets/names/storage-bucket`/horizon/horizon.js"
