#!/bin/bash
set -e

NAME="$1"

if [ "$NAME" == "" ]; then
    echo "Usage: $0 name"
    exit 1
fi

TMPDIR=$(mktemp -d)
TARBALL=$(mktemp)
trap "rm -rf $TMPDIR $TARBALL" EXIT

for TYPE in rsa dsa ecdsa ed25519; do
    ssh-keygen -q -t "$TYPE" -f "$TMPDIR/ssh_host_${TYPE}_key" -N '' -C 'root@host'
done

tar -czf "$TARBALL" -C "$TMPDIR" .

cat <<EOF
# Generated with gen-host-key-secret.sh on $(date)
apiVersion: v1
kind: Secret
metadata:
  name: $NAME
type: Opaque
data:
  ssh-key-tarball: $(base64 -w0 "$TARBALL")
EOF
