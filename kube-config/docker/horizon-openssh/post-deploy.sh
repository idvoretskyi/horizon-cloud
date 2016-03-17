#!/bin/sh
set -e

if [ "$DIR" = "" ]; then
    echo "A DIR must be passed"
    exit 1
fi

if [ ! -e /data/"$DIR" ]; then
    echo "Deploy $DIR does not exist in /data"
    exit 1
fi

chmod -R a+rX /data/"$DIR"

# NB: this is atomic
ln -nsf /data/"$DIR" /data/current_new
mv -T /data/current_new /data/current
