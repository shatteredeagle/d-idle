'use strict';
const fs = require('node:fs');
const path = require('node:path');

// config.json is {"address"} plus optional keys the app adds itself: "window" {width,height} and "hideToTray".
// Unknown keys are kept on save. 0600, no passphrase ever (the login is typed on the watch page).
const FILE = 'config.json';

function load(dir) {
  let raw;
  try {
    raw = fs.readFileSync(path.join(dir, FILE), 'utf8');
  } catch (e) {
    if (e.code === 'ENOENT') return {};
    throw e;
  }
  const c = JSON.parse(raw);
  if (c === null || typeof c !== 'object' || Array.isArray(c)) throw new Error('config.json is not an object');
  return c;
}

function save(dir, config) {
  fs.mkdirSync(dir, { recursive: true, mode: 0o700 });
  const tmp = path.join(dir, FILE + '.tmp');
  fs.writeFileSync(tmp, JSON.stringify(config, null, 2) + '\n', { mode: 0o600 });
  fs.chmodSync(tmp, 0o600); // writeFile's mode is masked by umask and ignored on an existing file
  fs.renameSync(tmp, path.join(dir, FILE));
}

// windowSize returns a sane stored size or the default; the app never trusts the file blindly.
function windowSize(config, def = { width: 1280, height: 800 }) {
  const w = config.window;
  const ok = (n) => Number.isInteger(n) && n >= 900 && n <= 10000;
  const okH = (n) => Number.isInteger(n) && n >= 600 && n <= 10000;
  return w && ok(w.width) && okH(w.height) ? { width: w.width, height: w.height } : def;
}

module.exports = { load, save, windowSize, FILE };
