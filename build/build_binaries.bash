#!/bin/bash
set -eu
MYDIR="$(dirname "$0")"

usage() {
    echo "Usage: $0 outdir programname [programname ...]"
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

BUILD_DIR="$(mktemp -d)"
trap 'rm -rf $BUILD_DIR' EXIT

echo "Building using $(go version)"

echo "Using build dir $BUILD_DIR"
OLDGOPATH="$GOPATH"
export GOPATH="$BUILD_DIR"

mkdir -p "$BUILD_DIR/src/$PROJECT"
rsync -a "$MYDIR/../" "$BUILD_DIR/src/$PROJECT/"

echo "Copying pre-existing dependency repos..."
# NB: This step is not neccessary for correctness, but it avoids re-cloning
# entire repositories for each build if they have already been checked out.
# glock will handle the case where they don't exist by doing clones from scratch
# later on.
cut -d ' ' -f 1 "$MYDIR/../GLOCKFILE" | while read DEP; do
    if [[ -e "$OLDGOPATH/src/$DEP" ]]; then
        mkdir -p "$BUILD_DIR/src/$DEP"
        rsync -a "$OLDGOPATH/src/$DEP/" "$BUILD_DIR/src/$DEP/"
    fi
done


GLOCK="$(which glock || true 2>/dev/null)"
if [[ ! -e $GLOCK ]]; then
    echo "Getting glock..."
    go get github.com/robfig/glock
    GLOCK="$BUILD_DIR/bin/glock"
fi

echo "glock sync..."
"$GLOCK" sync "$PROJECT"

# Disabling cgo disables the libc binding.
export CGO_ENABLED=0
export GOBIN="$OUTDIR"

export GOARCH=amd64
for GOOS in $GOOSES; do
    export GOOS
    for PROGRAM in $PROGRAMS; do
        echo "Building $PROGRAM-$GOOS-$GOARCH"
        go build -o "$OUTDIR/$PROGRAM-$GOOS-$GOARCH" "$PROJECT/cmd/$PROGRAM"
    done
done
