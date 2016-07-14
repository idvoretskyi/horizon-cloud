#!/bin/bash
set -eu

mkdir -p /hzc-web-backend
cd /hzc-web-backend
git clone -b hzc_v1.1.3_1 https://github.com/rethinkdb/horizon

pushd horizon/client
npm install
npm run prepublish
popd

pushd horizon/server
mkdir -p node_modules/@horizon/
ln -s /hzc-web-backend/horizon/client node_modules/@horizon/client
npm install
popd

mkdir -p node_modules/@horizon
ln -s /hzc-web-backend/horizon/client node_modules/@horizon/client
ln -s /hzc-web-backend/horizon/server node_modules/@horizon/server
