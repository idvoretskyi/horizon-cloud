#!/bin/bash
set -eu

cd /horizon/client && npm link
cd /horizon/server && npm link @horizon/client
cd /horizon/server && npm link
cd /horizon/cli && npm link @horizon/server
cd /horizon/cli && npm install -g
su -s /bin/sh horizon -c 'cd ~ && hz init app'
