import * as uuid from 'uuid';

import assert from 'assert';
import r from 'rethinkdb';

import * as github from './github';

function checkErr(summary) {
  if (summary.skipped != 0) {
    console.log("WARNING: skipped write");
  }
  assert.equal(summary.errors, 0);
}
function expectEmpty(x) {
  assert.equal(x.constructor, Object);
  assert.equal(Object.keys(x).length, 0);
}

const usersTbl = r.db('web_backend_internal').table('users');
const authTbl = r.db('web_backend_internal').table('users_auth');

function isManagedUser(user) {
  return user.groups.indexOf('authenticated') != -1;
}

function userStatus(user) {
  if (!user) return 'error';
  if (!isManagedUser(user)) return 'unmanaged';
  if (!user.data) return 'new';
  return user.data.status;
}

function newToReady(conn, user) {
  return authTbl.getAll(user.id, {index: 'user_id'}).run(conn).then(cursor => {
    return cursor.toArray();
  }).then(arr => {
    assert.equal(arr.length, 1);
    return arr[0].id[1];
  }).then(authId => {
    return github.loginFromId(authId).then(login => {
      return github.keysFromLogin(login).then(keys => {
        return usersTbl.get(user.id).update({
          data: {githubLogin: login, githubId: authId, keys: keys, status: 'ready'},
        }).run(conn).then(checkErr);
      });
    });
  });
}

function userPush(conn) {
  console.log(`Initializing users in background...`);
  const opts = {includeInitial: true};
  return usersTbl.changes(opts).run(conn).then(cursor => {
    return cursor.eachAsync(c => {
      const user = c.new_val;
      switch (userStatus(user)) {
      case 'new': return newToReady(conn, user);

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

const host = process.env.HZC_WEB_SYNC_WEB_HOST || 'rethinkdb-web'
const port = process.env.HZC_WEB_SYNC_WEB_PORT || 28015
function main(startTime) {
  return r.connect({host, port}).then(conn => {
    const generation = uuid.v4();
    console.log(`GENERATION: ${generation}`);
    return userPush(conn);
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
