import assert from 'assert';
import * as github from './github';

function userNeedsUpdate(user) {
  return (user['groups'].indexOf('authenticated') != -1
          && !(user['data'] && user['data']['status'] == 'ready'));
}

function updateUser(conn, db, userId, authId) {
  return github.loginFromId(authId).then((login) => {
    return github.keysFromLogin(login).then((keys) => {
      return db.table('users').get(userId).update({
        data: {githubLogin: login, githubId: authId, keys: keys, status: 'ready'}
      }).run(conn).then((summary) => {
        assert.equal(summary['errors'], 0);
      });
    });
  });
}

export function userSync(hz) {
  const r = hz._r;
  hz._reql_conn.ready().then(() => {
    const conn = hz._reql_conn.connection();
    const db = r.db(hz._reql_conn.metadata()._internal_db);
    return db.table('users').changes({includeInitial: true}).run(conn).then((cursor) => {
      return cursor.eachAsync((c) => {
        const user = c['new_val']
        if (user && userNeedsUpdate(user)) {
          console.log(user);
          const q = db.table('users_auth').getAll(user.id, {index: 'user_id'})
          return q.run(conn).then((x) => x.toArray()).then((arr) => {
            assert.equal(arr.length, 1);
            return updateUser(conn, db, user.id, arr[0].id[1]);
          });
        }
      });
    });
  }).catch((err) => {
    console.log('Error: ' + err.stack);
  });
}
