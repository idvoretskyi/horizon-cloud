#!/bin/bash
set -eu

mkdir -p /data/rethinkdb_data
chown -R rethinkdb:rethinkdb /data/rethinkdb_data

RDB_CACHE_SIZE=${RDB_CACHE_SIZE:-128}

exec su -c /bin/sh rethinkdb -c '/rethinkdb --bind all -d /data/rethinkdb_data --cache-size '"$RDB_CACHE_SIZE"
