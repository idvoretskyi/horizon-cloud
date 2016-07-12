import * as uuid from 'uuid';

import assert from 'assert';
import r from 'rethinkdb'

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

export function projectSync(...args) {
  return copySync.apply(this, args.concat(['projects']));
}

export function domainSync(...args) {
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

export function userSync(sys, web, generation) {
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
    return Promise.all([
      userSync(sys, web, generation),
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
