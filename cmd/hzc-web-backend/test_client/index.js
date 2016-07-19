'use strict';


const hz = Horizon({host: 'web-backend.hzc-dev.io', secure: true, authType: 'token'});
hz._horizonPath = 'https://web-backend.hzc-dev.io/horizon'

const Observable = hz.currentUser().watch().constructor

var userReadyObserver
const userReady = Observable.create((observer) => {
  userReadyObserver = observer;
}).publish()
userReady.connect();
console.assert(userReadyObserver);

///

if (!hz.hasAuthToken()) {
  console.log('foo');
  hz.authEndpoint('github').subscribe((endpoint) => {
    const loc = 'https://web-backend.hzc-dev.io' + endpoint;
    console.log("Redirecting to " + loc);
    window.location = loc;
  });
} else {
  hz.connect();
  hz.onReady().subscribe(() => {
    try {
      console.log('bar');
      console.debug("Authenticated.");
      window.jwt = hz.utensils.tokenStorage._storage['horizon-jwt'];
      console.debug("jwt: " + jwt);
      console.debug('watching currentUser...');
      hz.currentUser().watch().takeUntil(userReady).subscribe((user) => {
        try {
          console.debug("user: " + JSON.stringify(user));
          if (user['data'] && user['data']['status'] == 'ready') {
            userReadyObserver.next(user);
            userReadyObserver.complete();
          }
        } catch (e) {
          console.log("*** " + e.stack);
        }
      });
    } catch (e) {
      console.log("*** " + e.stack);
    }
  });
}

userReady.subscribe((user) => {
  console.debug('in userReady: ' + JSON.stringify(user));
})
userReady.subscribe((user) => {
  console.debug('in userReady2: ' + JSON.stringify(user));
})

const xhr = new XMLHttpRequest
xhr.onreadystatechange = () => {
  if (xhr.readyState == XMLHttpRequest.DONE) {
    console.log([xhr.status, xhr.responseText]);
  }
}
xhr.open('POST', 'https://web-backend.hzc-dev.io/api/domains/add');
xhr.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
xhr.send(JSON.stringify({
  jwt: JSON.parse(localStorage['horizon-jwt']).horizon,
  project: "test",
  domain: "foobar.hzc-dev.io",
}));

const xhr2 = new XMLHttpRequest
xhr2.onreadystatechange = () => {
  if (xhr2.readyState == XMLHttpRequest.DONE) {
    console.log([xhr2.status, xhr2.responseText]);
  }
}
xhr2.open('POST', 'https://web-backend.hzc-dev.io/api/domains/del');
xhr2.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
xhr2.send(JSON.stringify({
  jwt: JSON.parse(localStorage['horizon-jwt']).horizon,
  project: "test",
  domain: "foobar.hzc-dev.io",
}));
