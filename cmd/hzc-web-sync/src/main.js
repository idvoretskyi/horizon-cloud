import * as uuid from 'uuid';

import assert from 'assert';
import r from 'rethinkdb';

import * as github from './github';

function checkErr(summary) {
  assert.equal(summary.errors, 0);
}
function expectEmpty(x) {
  assert.equal(x.constructor, Object);
  assert.equal(Object.keys(x).length, 0);
}

function syncTo(generation, src, srcConn, dest, destConn, syncOp, removeOp) {
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
          const q = removeOp(generation, dest.filter(row => row('gen').ne(generation)));
          return q.run(destConn).then(checkErr).then(() => {
            console.log(`Cleared old generations for ${dest}.`);
          });
        }
        break;
      case 'initial':
      case 'add':
      case 'change':
        return syncOp(generation, dest, c.new_val).run(destConn).then(checkErr);
      case 'uninital':
      case 'remove':
        return removeOp(generation, dest.get(c.old_val.id)).run(destConn).then(checkErr);
      }
    });
  });
}

function insertOp(generation, destTable, newVal) {
  const o = Object.assign({}, newVal, {gen: generation})
  return destTable.insert(o, {conflict: 'replace'});
}

function delOp(generation, destSet) {
  return destSet.delete();
}

function copySync(sys, web, generation, tableName) {
  console.log(`Syncing ${tableName} (generation ${generation})...`);
  return syncTo(
    generation,
    r.db('hzc_api').table(tableName),
    sys,
    r.db('web_backend').table(tableName),
    web,
    insertOp,
    delOp);
}

function projectSync(...args) {
  return copySync.apply(this, args.concat(['projects']));
}

function domainSync(...args) {
  return copySync.apply(this, args.concat(['domains']));
}

function userReadyOp(generation, destTable, newVal) {
  return destTable.get(newVal.id).update({
    gen: generation,
    data: {
      status: 'ready',
      keys: newVal.PublicSSHKeys,
    },
  });
}

function userRemoveOp(generation, destSet, newVal) {
  return destSet.update({
    gen: generation,
    data: {
      status: 'deleted',
    },
  });
}

function userSync(sys, web, generation) {
  console.log(`Syncing users (generation ${generation})...`);
  return syncTo(
    generation,
    r.db('hzc_api').table('users'),
    sys,
    r.db('web_backend_internal').table('users'),
    web,
    userReadyOp,
    userRemoveOp);
}

const usersTbl = r.db('web_backend_internal').table('users');
const authTbl = r.db('web_backend_internal').table('users_auth');
const pushTbl = r.db('hzc_api').table('users');

function isManagedUser(user) {
  return user.groups.indexOf('authenticated') != -1;
}

function userStatus(user) {
  if (!user) return 'error';
  if (!isManagedUser(user)) return 'unmanaged';
  if (!user.data) return 'new';
  return user.data.status;
}

function newToApiWait(web, user) {
  return authTbl.getAll(user.id, {index: 'user_id'}).run(web).then(cursor => {
    return cursor.toArray();
  }).then(arr => {
    assert.equal(arr.length, 1);
    return arr[0].id[1];
  }).then(authId => {
    return github.loginFromId(authId).then(login => {
      return usersTbl.get(user.id).update({
        data: {githubLogin: login, githubId: authId, status: 'apiWait'},
      }).run(web).then(checkErr);
    });
  });
}

function apiWaitToReady(sys, user) {
  return github.keysFromLogin(user.data.githubLogin).then(keys => {
    return pushTbl.insert({Name: user.id, PublicSSHKeys: keys}).run(sys).then(checkErr);
  });
}

function userPush(sys, web) {
  console.log(`Initializing users in background...`);
  const opts = {includeInitial: true};
  return usersTbl.changes(opts).run(web).then(cursor => {
    return cursor.eachAsync(c => {
      const user = c.new_val;
      switch (userStatus(user)) {
      case 'new': return newToApiWait(web, user);
      case 'apiWait': return apiWaitToReady(sys, user);

      case 'ready': return;
      case 'unmanaged': return;
      case 'deleted': return;

      case 'error': // fallthru
      default: throw new Error('unexpected change: ' + JSON.stringify(
        [c, user, userStatus(user)]));
      }
    });
  });
}

const sysHost = process.env.HZC_WEB_SYNC_SYS_HOST || 'rethinkdb-sys'
const sysPort = process.env.HZC_WEB_SYNC_SYS_PORT || 28015
const webHost = process.env.HZC_WEB_SYNC_WEB_HOST || 'rethinkdb-web'
const webPort = process.env.HZC_WEB_SYNC_WEB_PORT || 28015
function main(startTime) {
  return Promise.all([
    r.connect({host: sysHost, port: sysPort}),
    r.connect({host: webHost, port: webPort}),
  ]).then(([sys, web]) => {
    const generation = uuid.v4();
    console.log(`GENERATION: ${generation}`);
    return Promise.all([
      userSync(sys, web, generation),
      userPush(sys, web),
      projectSync(sys, web, generation),
      domainSync(sys, web, generation),
    ]);
  }).catch(e => {
    const curTime = new Date();
    const secondsUp = (curTime - startTime)/1000;
    console.error(`Error after ${secondsUp} seconds: ${e}\n${e.stack}`);
    if (secondsUp < 5 * 60) {
      console.error('Up for less than five minutes, crashing...');
      process.exit(1);
    }
    setTimeout(main, 5000, curTime);
  });
}

main(new Date());
