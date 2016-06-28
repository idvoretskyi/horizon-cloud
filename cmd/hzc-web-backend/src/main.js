require('source-map-support').install();

(() => console.log('foo'))();

throw new Error("foo");
