#!/bin/bash
set -eu
MYDIR="$(dirname "$0")"

usage() {
    echo "Usage: $0 outdir programname[,programname,...] [goos,goos,...]'"
    echo
    echo "If no goos options are given, they default to linux,darwin"
    exit 1
}

if [[ $# -lt 2 ]]; then
    usage
fi

OUTDIR="${1:-.}"
PROGRAMS="$(echo "$2" | sed -e 's/,/ /g')"
GOOSES="$(echo "${3:-linux,darwin}" | sed -e 's/,/ /g')"

PROJECT=github.com/rethinkdb/horizon-cloud

echo "glock sync..."
glock sync "$PROJECT"

echo "Building using $(go version)"

# Disabling cgo disables the libc binding.
export CGO_ENABLED=0

export GOARCH=amd64
for GOOS in $GOOSES; do
    export GOOS
    for PROGRAM in $PROGRAMS; do
        echo "Building $PROGRAM-$GOOS-$GOARCH"
        go build -v -i -o "$OUTDIR/$PROGRAM-$GOOS-$GOARCH" "$PROJECT/cmd/$PROGRAM"
    done
done
