'use strict';


const hz = Horizon({authType: 'token'});

const printObserver = Rx.Observer.create(
  (x) => console.debug("Next: " + x),
  (x) => console.debug("Error: " + x),
  (x) => console.debug("Completed: " + x))

///

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
    console.log("Redirecting to " + endpoint);
    window.location.pathname = endpoint;
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
xhr.open('POST', '/api/projects/del');
xhr.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
xhr.send(JSON.stringify({
  jwt: JSON.parse(localStorage['horizon-jwt']).horizon,
  project: 'foo',
}));
