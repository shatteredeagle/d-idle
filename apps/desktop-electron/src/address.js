'use strict';
// Port of apps/desktop/address.go; the vectors in test/address.test.js are the same as desktop_test.go.

const PAGES = Object.freeze({
  watch: '/watch',
  inventory: '/watch/inventory',
  policy: '/watch/policy',
  return: '/watch/return',
});

// normalizeAddress turns "host:8891" or "http://host:8891/watch" into scheme://host[:port] and nothing else. The
// login's same-origin rule compares the page Origin with the server's configured origin, so the host is only
// lowercased, a port is never added or dropped, and credentials are refused (no passphrase in the config file).
function normalizeAddress(input) {
  let s = String(input ?? '').trim();
  if (s === '') throw new Error('the address is empty');
  if (!s.includes('://')) s = 'http://' + s;
  let u;
  try {
    u = new URL(s);
  } catch (e) {
    throw new Error('not a valid address');
  }
  if (u.protocol !== 'http:' && u.protocol !== 'https:') {
    throw new Error(`only http and https addresses work, not "${u.protocol.slice(0, -1)}"`);
  }
  if (u.username !== '' || u.password !== '') throw new Error('do not put a user name or passphrase in the address');
  if (u.hostname === '') throw new Error('the address has no host');
  // new URL drops a default port (":80" on http) but the Go launcher keeps what was typed, so read the typed text.
  const authority = s.replace(/^[a-z][a-z0-9+.-]*:\/\//i, '').split(/[/?#]/)[0];
  const m = /:(\d*)$/.exec(authority.replace(/^\[[^\]]*\]/, ''));
  if (m) {
    const n = Number(m[1]);
    if (m[1] === '' || n < 1 || n > 65535) throw new Error(`bad port "${m[1]}"`);
    return `${u.protocol}//${u.hostname}:${m[1]}`.toLowerCase();
  }
  return `${u.protocol}//${u.hostname}`.toLowerCase();
}

function pageURL(origin, page) {
  if (!Object.hasOwn(PAGES, page)) throw new Error(`unknown page "${page}" (watch, inventory, policy, return)`);
  return origin + PAGES[page];
}

module.exports = { PAGES, normalizeAddress, pageURL };
