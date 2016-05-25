#!/bin/bash
set -eu

cd /horizon/test && bash -x setupDev.sh
cd /horizon/cli && npm install -g
su -s /bin/sh horizon -c 'cd ~ && hz init app'
