#!/bin/bash
set -eu
cd "$(dirname "$0")"

TMPDIR=$(mktemp -d)
trap 'rm -rf $TMPDIR' EXIT

./build_binaries.bash "$TMPDIR" hzc-client

gsutil cp gs://update.hzc.io/metadata.json "$TMPDIR/metadata.json"

cat <<EOF > "$TMPDIR/update.json"
{
    "linux-amd64": {
        "url": "http://update.hzc.io/hzc-client-linux-amd64",
        "sha256": "$(sha256sum "$TMPDIR/hzc-client-linux-amd64" | cut -d ' ' -f 1 | tr -d '\n')"
    },
    "darwin-amd64": {
        "url": "http://update.hzc.io/hzc-client-darwin-amd64",
        "sha256": "$(sha256sum "$TMPDIR/hzc-client-darwin-amd64" | cut -d ' ' -f 1 | tr -d '\n')"
    }
}
EOF

cat "$TMPDIR/metadata.json" "$TMPDIR/update.json" \
    | jq -s '.[0]["hzc-client"].version += 1 | .[0]["hzc-client"].binaries = .[1] | .[0]' \
    > "$TMPDIR/newmetadata.json"
mv "$TMPDIR/newmetadata.json" "$TMPDIR/metadata.json"
rm "$TMPDIR/update.json"

for file in "$TMPDIR/"*; do
    gsutil -h "Cache-Control: public, max-age=15" cp "$file" gs://update.hzc.io/"$(basename "$file")"
done
