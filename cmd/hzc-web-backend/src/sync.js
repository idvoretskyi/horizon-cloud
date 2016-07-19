import * as github from './github';
import * as kube from './kube';

import assert from 'assert';
import * as uuid from 'uuid';

const generation = uuid.v4();

function checkErr(summary) {
  assert.equal(summary.errors, 0);
}
function expectEmpty(x) {
  assert.equal(x.constructor, Object);
  assert.equal(Object.keys(x).length, 0);
}

function userNeedsActivation(user) {
  return (user.groups.indexOf('authenticated') != -1
          && !(user.data && user.data.status == 'ready'));
}

function activateUser(conn, db, userId, authId) {
  return github.loginFromId(authId).then((login) => {
    // These could be done in parallel, but the latency savings are
    // minimal and it makes the status logic more complicated if they
    // can happen in any order.
    return db.table('users').get(userId).update({
      data: {githubLogin: login, githubId: authId, status: 'apiWait'},
    }).run(conn).then(checkErr).then(() => {
      return github.keysFromLogin(login)
    }).then((keys) => {
      return kube.apiReq('/users/create', {Name: userId}).catch((err) => {
        console.log('Creation error, continuing...');
      }).then(() => {
        return kube.apiReq('/users/addKeys', {Name: userId, Keys: keys});
      });
    });
  }).then(expectEmpty);
}

export function userPush(hz) {
  const r = hz._r;
  return hz._reql_conn.ready().then(() => {
    console.log('Syncing users...');
    const conn = hz._reql_conn.connection();
    const db = r.db(hz._reql_conn.metadata()._internal_db);
    return db.table('users').changes({includeInitial: true}).run(conn).then((cursor) => {
      return cursor.eachAsync((c) => {
        const user = c.new_val
        if (user && userNeedsActivation(user)) {
          console.log(user);
          const q = db.table('users_auth').getAll(user.id, {index: 'user_id'})
          return q.run(conn).then((x) => x.toArray()).then((arr) => {
            assert.equal(arr.length, 1);
            return activateUser(conn, db, user.id, arr[0].id[1]);
          });
        }
      });
    });
  })
}
