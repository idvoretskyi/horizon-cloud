#!/bin/bash
set -eu

cd /horizon/test && bash -x setupDev.sh
su -s /bin/sh horizon -c 'cd ~ && hz init app'
