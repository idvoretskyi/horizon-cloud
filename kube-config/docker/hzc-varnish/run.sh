#!/bin/sh
set -eu

sh /etc/varnish/default.vcl.sh > /etc/varnish/default.vcl

exec varnishd -f /etc/varnish/default.vcl -s malloc,${VARNISH_MEMORY} -a '0.0.0.0:80' -F
