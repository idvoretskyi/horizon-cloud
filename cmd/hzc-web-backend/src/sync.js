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

function syncTo(r, src, srcConn, dest, destConn, syncOp, removeOp) {
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
          console.log(`Clearing old generations for ${dest}...`);
          const q = removeOp(dest.filter((row) => row('gen').ne(generation)));
          return q.run(destConn).then(checkErr).then(() => {
            console.log(`Cleared old generations for ${dest}.`);
          });
        }
        break;
      case 'initial':
      case 'add':
      case 'change':
        return syncOp(dest, c.new_val).run(destConn).then(checkErr);
      case 'uninital':
      case 'remove':
        return removeOp(dest.get(c.old_val.id)).run(destConn).then(checkErr);
      }
    });
  });
}

function insertOp(destTable, newVal) {
  const o = Object.assign({}, newVal, {gen: generation})
  return destTable.insert(o, {conflict: 'replace'});
}

function delOp(destSet) {
  return destSet.delete();
}

function copySync(hz, apiRdbConnOpts, tableName) {
  const r = hz._r;
  return Promise.all([r.connect(apiRdbConnOpts), hz._reql_conn.ready()]).then((x) => {
    const apiRdbConn = x[0];
    // We ignore `x[1]` because the Horizon promise interface is
    // really more of a signal for some reason.
    console.log(`Syncing ${tableName} (generation ${generation})...`);
    return syncTo(
      r,
      r.db('hzc_api').table(tableName),
      apiRdbConn,
      r.db(hz._reql_conn.metadata()._db).table(tableName),
      hz._reql_conn.connection(),
      insertOp,
      delOp);
  });
}

export function projectSync(...args) {
  return copySync.apply(this, args.concat(['projects']));
}

export function domainSync(...args) {
  return copySync.apply(this, args.concat(['domains']));
}

function userReadyOp(destTable, newVal) {
  return destTable.get(newVal.id).update({
    data: {
      status: 'ready',
      keys: newVal.PublicSSHKeys,
    },
  });
}

function userRemoveOp(destSet, newVal) {
  return destSet.update({data: {status: 'deleted'}});
}

export function userSync(hz, apiRdbConnOpts) {
  const r = hz._r;
  return Promise.all([r.connect(apiRdbConnOpts), hz._reql_conn.ready()]).then((x) => {
    const apiRdbConn = x[0];
    // We ignore `x[1]` because the Horizon promise interface is
    // really more of a signal for some reason.
    console.log(`Syncing users (generation ${generation})...`);
    return syncTo(
      r,
      r.db('hzc_api').table('users'),
      apiRdbConn,
      r.db(hz._reql_conn.metadata()._internal_db).table('users'),
      hz._reql_conn.connection(),
      userReadyOp,
      userRemoveOp);
  });
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
    console.log('*** ' + db.table('users').get(0));
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
