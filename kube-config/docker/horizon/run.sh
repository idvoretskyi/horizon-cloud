#!/bin/bash
set -eu

exec su -s /bin/sh horizon -c 'exec hz serve'
