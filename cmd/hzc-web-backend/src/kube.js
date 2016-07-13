import * as fs from 'fs';
import * as https from 'https';

import assert from 'assert';

const domainPromise = new Promise((resolve, reject) => {
  fs.readFile('/secrets/names/domain', (err, data) => {
    if (err) {
      reject(err);
    } else {
      resolve(data);
    }
  });
})

const secretPromise = new Promise((resolve, reject) => {
  fs.readFile('/secrets/api-shared-secret/api-shared-secret', (err, data) => {
    if (err) {
      reject(err);
    } else {
      resolve(data);
    }
  });
})

const baseOptsPromise = Promise.all([domainPromise, secretPromise]).then((ds) => ({
  hostname: `api.${ds[0]}`,
  headers: {'X-Horizon-Cloud-Shared-Secret': ds[1]},
  method: 'POST',
}));

export function apiReq(path, obj) {
  return baseOptsPromise.then((baseOpts) => {
    const opts = Object.assign({}, baseOpts, {path: "/v1" + path});
    return new Promise((resolve, reject) => {
      try {
        const req = https.request(opts, (res) => {
          let data = '';
          res.on('data', (x) => data += x);
          res.on('end', () => {
            try {
              const parsed = JSON.parse(data);
              if (parsed.Success) {
                resolve(parsed.Content);
              } else {
                reject(new Error('API error: ' + parsed.Error));
              }
            } catch (e) {
              reject(e);
            }
          });
        })
        req.write(JSON.stringify(obj));
        req.end();
        req.on('error', (err) => reject(err));
      } catch (e) {
        reject(e);
      }
    });
  });
}
