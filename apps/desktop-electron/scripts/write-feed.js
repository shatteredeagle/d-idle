'use strict';
// Package-time step: writes build/feed.json from D2IDLE_UPDATE_FEED (the friends' feed address, a deployment value
// that is not stored in the repo). Without the variable nothing is written and the update check stays off.
const fs = require('node:fs');
const path = require('node:path');

const feed = (process.env.D2IDLE_UPDATE_FEED || '').trim();
const file = path.join(__dirname, '..', 'build', 'feed.json');
if (!feed) {
  fs.rmSync(file, { force: true });
  console.log('no D2IDLE_UPDATE_FEED: update check off in this build');
} else {
  const u = new URL(feed);
  if (u.protocol !== 'https:' || u.username || u.password || u.search || u.hash) throw new Error('D2IDLE_UPDATE_FEED must be a plain https address');
  fs.writeFileSync(file, JSON.stringify({ feed: feed.replace(/\/+$/, '') }) + '\n');
  console.log('update feed written to build/feed.json');
}
