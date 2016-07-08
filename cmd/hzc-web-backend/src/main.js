import * as endpoints from './endpoints';
import * as sync from './sync';
import * as horizonShim from './horizonShim';
import * as fs from 'fs';
import * as https from 'https';

import horizon from '@horizon/server';
import express from 'express';


require('source-map-support').install();

const app = express();
const httpsServer = https.createServer({
  key: fs.readFileSync('/secrets/dev/wildcard-ssl/key'),
  cert: fs.readFileSync('/secrets/dev/wildcard-ssl/crt') +
    fs.readFileSync('/secrets/dev/wildcard-ssl/crt-bundle'),
}, app).listen(8181)
httpsServer.on('listening', () => {
  console.log(`Listening on ${JSON.stringify(httpsServer.address())}`)
});
const options = {
  project_name: 'web_backend',
  auth: {
    token_secret: 'XvKNYSyetDxrRGqIiHayP9IpMnP5J',
    allow_unauthenticated: true,
  },
  rdb_port: 38015,

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

endpoints.attachApi(app)

app.use(express.static('test_client'))

//
sync.userSync(hz)
