import * as endpoints from './endpoints';
import * as horizonShim from './horizonShim';
import * as sync from './sync';

import * as fs from 'fs';
import * as https from 'https';

import horizon from '@horizon/server';
import express from 'express';
import bodyParser from 'body-parser';

require('source-map-support').install();

const app = express();
app.use(bodyParser.json());

const httpsServer = https.createServer({
  key: fs.readFileSync('/secrets/wildcard-ssl/key'),
  cert: fs.readFileSync('/secrets/wildcard-ssl/crt') +
    fs.readFileSync('/secrets/wildcard-ssl/crt-bundle'),
}, app).listen(8181)
httpsServer.on('listening', () => {
  console.log(`Listening on ${JSON.stringify(httpsServer.address())}`)
});
// RSI: Make these options work inside of Kube

const rdbHost = process.env.HZC_WEB_BACKEND_RDB_HOST || 'rethinkdb-web'
const rdbPort = process.env.HZC_WEB_BACKEND_RDB_PORT || '28015'
const options = {
  project_name: 'web_backend',
  auth: {
    token_secret: 'XvKNYSyetDxrRGqIiHayP9IpMnP5J',
    allow_unauthenticated: true,
  },
  rdb_host: rdbHost,
  rdb_port: rdbPort,

  // Dev Mode
  auto_create_collection: true,
  auto_create_index: true,
  permissions: false,
};
const hz = horizon(httpsServer, options)

hz.add_auth_provider(horizon.auth.github, {
  id: '88a6dbc480e460a19aa3',
  secret: 'bbb71ccef1d1b8a8847139bd4f9f97e262b70528',
  path: 'github',
})

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
