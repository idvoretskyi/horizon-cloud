import assert from 'assert';

function userNeedsUpdate(user) {
  return (user['groups'].indexOf('authenticated') != -1
          && !(user['data'] && user['data']['status'] == 'ready'));
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
            const auth = arr[0];
            return db.table('users').get(user.id).update({
              data: {username: auth.id, status: 'ready'}
            }).run(conn).then((summary) => {
              assert.equal(summary['errors'], 0);
            });
          });
        }
      });
    });
  }).catch((err) => {
    console.log('Error: ' + err.stack);
  });
}
