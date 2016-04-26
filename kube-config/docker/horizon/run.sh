#!/bin/bash
set -eu

# Strip all environment variables that aren't whitelisted
unset $(env | cut -d= -f1 | grep -Ev '^(HOME|PATH|HOSTNAME|HZ_.*)$')

exec su -s /bin/sh horizon -c 'cd ~ && exec hz serve app'
