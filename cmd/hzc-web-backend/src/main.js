import * as fs from 'fs';
import * as https from 'https';

import assert from 'assert';
import bodyParser from 'body-parser';
import cors from 'cors';
import express from 'express';
import horizon from '@horizon/server';

import * as endpoints from './endpoints';
import * as horizonShim from './horizonShim';

require('source-map-support').install();

const app = express();
app.use(bodyParser.json());
app.use(function(req, res, next) {
  res.setHeader("Access-Control-Allow-Origin", "*");
  res.setHeader("Access-Control-Allow-Headers",
                "Origin, X-Requested-With, Content-Type, Accept");
  next();
});

const port = process.env.HZC_PORT || 4433
const httpsServer = https.createServer({
  key: fs.readFileSync('/secrets/wildcard-ssl/key'),
  cert: fs.readFileSync('/secrets/wildcard-ssl/crt') +
    fs.readFileSync('/secrets/wildcard-ssl/crt-bundle'),
}, app).listen(port)
httpsServer.on('listening', () => {
  console.log(`Listening on ${JSON.stringify(httpsServer.address())}`)
});

const rdbHost = process.env.HZC_RDB_HOST || 'rethinkdb-web'
const rdbPort = process.env.HZC_RDB_PORT || 28015
// RSI: ready this shit out of secrets.
// RSI: turn off all this dev mode shit.
const options = {
  project_name: 'web_backend',
  auth: {
    token_secret: 'XvKNYSyetDxrRGqIiHayP9IpMnP5J',
    // RSI: turn this off.
    allow_unauthenticated: true,
    success_redirect: 'http://localhost:8000/',
    failure_redirect: 'http://localhost:8000/',
  },
  rdb_host: rdbHost,
  rdb_port: rdbPort,

  // Dev Mode
  auto_create_collection: true,
  auto_create_index: true,
  permissions: false,
};
const hz = horizon(httpsServer, options)

const githubApiKeys = JSON.parse(fs.readFileSync('/secrets/api-keys/github'))
// RSI: read these out of secrets.
hz.add_auth_provider(horizon.auth.github, {
  id: githubApiKeys.client_id,
  secret: githubApiKeys.client_secret,
  path: 'github',
})

// 1337 h4x
const listeners = httpsServer.listeners('request').slice(0);
assert.equal(listeners.length, 1);
const hzListener = listeners[0];
httpsServer.removeAllListeners('request');
httpsServer.on('request', (req, res) => {
  res.setHeader("Access-Control-Allow-Origin", "*");
  res.setHeader("Access-Control-Allow-Headers",
                "Origin, X-Requested-With, Content-Type, Accept");
  hzListener.call(httpsServer, req, res);
});

endpoints.attach(hz, app)

app.use(express.static('test_client'))
