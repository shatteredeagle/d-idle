'use strict';
// Pinned public keys of the signed release feed: the same two keys the bot's updater pins
// (deploy/d2d-container/pubkeys/*.pub, format "d2sign-ed25519-public-v1"). A test compares them with the files.
module.exports = [
  'oYxbchrwEKGsBYLKdbaasfuiPYdssRv+r7fJxZJrYY4=', // beta.pub: signs the friends' feed
  'sNsP+MnTQszJT2TXqPQ/X/qUO2aDypHu01eJsU6hWrc=', // current.pub: signs GitHub releases
];
