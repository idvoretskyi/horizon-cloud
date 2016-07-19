import * as endpoints from './endpoints';
import * as horizonShim from './horizonShim';
import * as sync from './sync';

import * as fs from 'fs';
import * as https from 'https';

import assert from 'assert';
import bodyParser from 'body-parser';
import cors from 'cors';
import express from 'express';
import horizon from '@horizon/server';

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
const options = {
  project_name: 'web_backend',
  auth: {
    token_secret: 'XvKNYSyetDxrRGqIiHayP9IpMnP5J',
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

// RSI: read these out of secrets.
hz.add_auth_provider(horizon.auth.github, {
  id: '88a6dbc480e460a19aa3',
  secret: 'bbb71ccef1d1b8a8847139bd4f9f97e262b70528',
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

//
function logErr(err) {
  if (err instanceof Error) {
    console.log('*** Error: ' + err.stack);
  } else {
    console.log('*** Bad Error: ' + err);
  }
}
sync.userPush(hz).catch(logErr);

