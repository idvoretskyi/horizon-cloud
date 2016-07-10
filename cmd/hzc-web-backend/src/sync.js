import * as github from './github';

import assert from 'assert';
import * as uuid from 'uuid';

const generation = uuid.v4();

function checkErr(summary) {
  assert.equal(summary.errors, 0);
}

function userNeedsUpdate(user) {
  return (user.groups.indexOf('authenticated') != -1
          && !(user.data && user.data.status == 'ready'));
}

function updateUser(conn, db, userId, authId) {
  return github.loginFromId(authId).then((login) => {
    return github.keysFromLogin(login).then((keys) => {
      return db.table('users').get(userId).update({
        data: {githubLogin: login, githubId: authId, keys: keys, status: 'ready'}
      }).run(conn).then(checkErr);
    });
  });
}

export function userSync(hz) {
  const r = hz._r;
  return hz._reql_conn.ready().then(() => {
    console.log('Syncing users...');
    const conn = hz._reql_conn.connection();
    const db = r.db(hz._reql_conn.metadata()._internal_db);
    return db.table('users').changes({includeInitial: true}).run(conn).then((cursor) => {
      return cursor.eachAsync((c) => {
        const user = c.new_val
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
  })
}

function syncTo(r, src, srcConn, dest, destConn) {
  const opts = {includeStates: true, includeTypes: true, includeInitial: true}
  return src.changes().run(srcConn, opts).then((cursor) => {
    return cursor.eachAsync((c) => {
      console.log(c);
      switch (c.type) {
      case 'error':
        throw new Error(c.error);
        break;
      case 'state':
        // Once we've caught up with what was in the table to begin
        // with, go back and delete everything from the previous
        // generation (which must have been deleted in the source
        // table).
        if (c.state == 'ready') {
          console.log('Clearing old generation...');
          const q = dest.filter((row) => row('gen').ne(generation)).delete()
          return q.run(destConn).then(checkErr);
        }
        break;
      case 'initial':
      case 'add':
      case 'change':
        const q = dest.insert(r.expr(c.new_val).merge({gen: generation}),
                              {conflict: 'replace'})
        return q.run(destConn).then(checkErr);
      case 'uninital':
      case 'remove':
        return dest.get(c.old_val.id).delete().run(destConn).then(checkErr);
      }
    });
  });
}

export function projectSync(hz, apiRdbConnOpts) {
  const r = hz._r;
  return Promise.all([r.connect(apiRdbConnOpts), hz._reql_conn.ready()]).then((x) => {
    const apiRdbConn = x[0];
    // We ignore `x[1]` because the Horizon promise interface is
    // really more of a signal for some reason.
    console.log(`Syncing projects (generation ${generation})...`);
    return syncTo(
      r,
      r.db('hzc_api').table('projects'),
      apiRdbConn,
      r.db(hz._reql_conn.metadata()._db).table('projects'),
      hz._reql_conn.connection());
  });
}
