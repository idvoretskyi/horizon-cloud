#!/bin/bash
set -eu
cd "$(dirname "$0")"

TMPDIR=$(mktemp -d)
trap 'rm -rf $TMPDIR' EXIT

../templates/horizon.js.sh > $TMPDIR/horizon.js
gsutil -h "Cache-Control:private,max-age=5" cp -a public-read $TMPDIR/horizon.js \
    "gs://`cat /secrets/names/storage-bucket`/horizon/horizon.js"
