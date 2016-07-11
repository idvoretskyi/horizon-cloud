import * as kube from './kube';

function projectCreate(user, params) {
  return kube.apiReq('/projects/setKubeConfig', {
    Project: `${user}/${params.project}`,
    KubeConfig: {
      NumRDB: 1,
      SizeRDB: 10,
      NumHorizon: 1,
    },
  });
}

endpoints = {
  '/api/projects/create': {
    valid: {project: projectValidator},
    func: projectCreate,
  },
}

function sendErr(res, err) {
  console.log(err);
  if (process.env.NODE_ENV != 'production') {
    res.status(500).send(JSON.stringify(res.body) + "\n" + err.stack);
  } else {
    res.status(500).send(`Invalid request: ${JSON.stringify(res.body)}`);
  }
}

function validateJwt(jwt) {

}

function validate(valid, obj) {
  const jwt = obj.jwt;
  if (!jwt) {
    throw new Error(`No JWT provided.`);
  }
  const user = validateJwt(jwt);
  for (let key in obj) {
    if (key == 'jwt') continue;
    const f = valid[key];
    if (!f) {
      throw new Error(`Unexpected key: ${key}`);
    }
    f(obj[key]);
  }
  return user;
}

function withValidation(req, res, spec) {
  try {
    user = validate(spec.valid, req.body);
    spec.func(user, req.body).then((x) => res.send(x)).catch((e) => sendErr(res, e));
  } catch (e) {
    sendErr(res, e);
  }
}

export function attach(app) {
  for (let ep in endpoints) {
    app.post(ep, (req, res) => withValidation(req, res, endpoints[ep]))
  }
}
