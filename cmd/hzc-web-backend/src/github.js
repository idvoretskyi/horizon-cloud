import * as fs from 'fs';
import * as https from 'https';
import * as qs from 'querystring';

const authObjPromise = new Promise((resolve, reject) => {
  fs.readFile('/secrets/dev/api-keys/github', (err, data) => {
    if (err) {
      reject(err);
    } else {
      resolve(data);
    }
  });
}).then((data) => JSON.parse(data));

function apiReq(basePath, baseReqOpts = {}) {
  return authObjPromise.then((authObj) => {
    const reqOpts = Object.assign({}, baseReqOpts, authObj)
    const opts = {
      hostname: 'api.github.com',
      path: basePath + '?' + qs.stringify(reqOpts),
      // GitHub requires this for some reason.
      headers: {'user-agent': 'node.js'},
    };
    return new Promise((resolve, reject) => {
      https.get(opts, (res) => {
        if (res.statusCode < 200 || res.statusCode > 299) {
          reject(new Error(JSON.stringify({
            path: path,
            reqOpts: reqOpts,
            url: url,
            code: res.statusCode,
            message: res.statusMessage
          })));
        }
        let data = '';
        res.on('data', (x) => data += x);
        res.on('end', () => resolve(data));
      }).on('error', (err) => reject(err));
    }).then((data) => JSON.parse(data));
  });
}

export function loginFromId(id) {
  return apiReq('/users', {since: id-1, per_page: 1}).then((res) => {
    return res[0]['login'];
  });
}

export function keysFromLogin(login) {
  return apiReq(`/users/${login}/keys`).then((res) => {
    return res.map((res) => res['key'].split(' ')[1]);
  });
}
