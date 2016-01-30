#!/bin/bash
set -e
set -x

(cd $GOPATH/src/github.com/rethinkdb/fusion && \
    git archive --format=tar HEAD) | gzip > setup/fusion-tarball.tar.gz
