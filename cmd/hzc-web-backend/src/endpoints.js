import * as kube from './kube';

import assert from 'assert';

function stubbedOut() {
  return Promise.reject(new Error('STUBBED OUT'));
}

function projectValid() {
  // RSI
}

function domainValid() {
  // RSI
}

function usersValid() {
  // RSI
}

function keysValid() {
  // RSI
}

const endpoints = {
  '/api/projects/add': {
    valid: {project: projectValid},
    func: (user, params) => {
      return kube.apiReq('/projects/setKubeConfig', {
        Project: `${user}/${params.project}`,
        KubeConfig: {
          NumRDB: 1,
          SizeRDB: 10,
          NumHorizon: 1,
        },
      });
    },
  },
  '/api/projects/del': {
    valid: {project: projectValid},
    func: (user, params) => {
      return kube.apiReq('/projects/delete', {
        Project: `${user}/${params.project}`,
      });
    },
  },
  '/api/projects/addUsers': {
    valid: {project: projectValid, users: usersValid},
    func: (user, params) => {
      return kube.apiReq('/projects/addUsers', {
        Project: `${user}/${params.project}`,
        Users: params.users,
      });
    },
  },
  '/api/projects/delUsers': {
    valid: {project: projectValid, users: usersValid},
    func: (user, params) => {
      return kube.apiReq('/projects/delUsers', {
        Project: `${user}/${params.project}`,
        Users: params.users,
      });
    },
  },

  '/api/domains/add': {
    valid: {project: projectValid, domain: domainValid},
    func: (user, params) => {
      return kube.apiReq('/domains/set', {
        Project: `${user}/${params.project}`,
        Domain: params.domain,
      });
    },
  },
  '/api/domains/del': {
    valid: {project: projectValid, domain: domainValid},
    func: (user, params) => {
      return kube.apiReq('/domains/del', {
        Project: `${user}/${params.project}`,
        Domain: params.domain,
      });
    },
  },

  '/api/user/addKeys': {
    valid: {keys: keysValid},
    func: (user, params) => {
      return kube.apiReq('/users/addKeys', {
        Name: user,
        Keys: params.keys,
      });
    },
  },
  '/api/user/delKeys': {
    valid: {keys: keysValid},
    func: (user, params) => {
      return kube.apiReq('/users/delKeys', {
        Name: user,
        Keys: params.keys,
      });
    },
  },
}

function sendErr(res) {
  return err => {
    console.log(err.stack);
    if (process.env.NODE_ENV != 'production') {
      res.status(500).send(JSON.stringify(res.body) + "\n" + err.stack);
    } else {
      res.status(500).send(`Invalid request: ${JSON.stringify(res.body)}`);
    }
  }
}

function validateJwt(hz, jwt) {
  // Strings can be literals or objects, because JavaScript is the
  // font of all decay and chaos in the Universe.
  if (!(typeof jwt == "string" || jwt instanceof String)) {
    return Promise.reject(new Error('Bad JWT provided.'));
  }
  return hz._auth._jwt.verify(jwt).then(info => info.payload.id);
}

function validate(hz, valid, obj) {
  return Promise.all([
    validateJwt(hz, obj.jwt),
    Promise.resolve().then(() => {
      for (let key in obj) {
        if (key == 'jwt') continue;
        const f = valid[key];
        if (!f) {
          throw new Error(`Unexpected key: ${key}`);
        }
        // `f` should only communicate information by throwing if
        // there's an error.
        const fres = f(obj[key]);
        assert.equal(fres, undefined)
      }
    }),
  ]).then((ps) => ps[0]);
}

function withValidation(hz, spec) {
  return function(req, res) {
    return validate(hz, spec.valid, req.body).then((user) => {
      return spec.func(user, req.body).then((x) => res.send(x));
    }).catch(sendErr(res));
  }
}

export function attach(hz, app) {
  for (let ep in endpoints) {
    app.post(ep, withValidation(hz, endpoints[ep]));
  }
}
