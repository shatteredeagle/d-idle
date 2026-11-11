'use strict';
// Update CHECK only. Fetches the release list of our own feed, verifies the Ed25519 signature of the newest desktop
// manifest against pinned keys, and reports a newer version. It never downloads or runs anything, never contacts
// github.com, and never follows a redirect (an asset or redirect on another host is refused).
// Feed shape and manifest rules are the bot updater's (internal/selfupdate, docs/friend-feed.md, docs/self-update-release.md).
const crypto = require('node:crypto');
const defaultKeys = require('./update-keys');

const FEED = 'https://updates.birbhub.com';
const REPO = 'shatteredeagle/d-idle'; // DefaultRepo of internal/selfupdate
const TAG_PREFIX = 'desktop-v';
const MAX_LIST = 4 << 20;
const MAX_MANIFEST = 1 << 20;
const SPKI_ED25519 = Buffer.from('302a300506032b6570032100', 'hex');

const parseVersion = (s) => {
  const p = String(s).split('.');
  if (p.length !== 3) return null;
  const v = p.map((x) => (/^(0|[1-9]\d*)$/.test(x) ? Number(x) : NaN));
  return v.some(Number.isNaN) ? null : v;
};
function compareVersions(a, b) {
  const va = parseVersion(a);
  const vb = parseVersion(b);
  if (!va && !vb) return 0;
  if (!va) return -1;
  if (!vb) return 1;
  for (let i = 0; i < 3; i++) if (va[i] !== vb[i]) return va[i] < vb[i] ? -1 : 1;
  return 0;
}

// verifySignature checks sig (base64 line, as d2sign writes it) over the exact bytes of doc against any pinned key.
function verifySignature(keysB64, doc, sigText) {
  const sig = Buffer.from(String(sigText).trim(), 'base64');
  if (sig.length !== 64) return false;
  return keysB64.some((k) => {
    const raw = Buffer.from(k, 'base64');
    if (raw.length !== 32) return false;
    const key = crypto.createPublicKey({ key: Buffer.concat([SPKI_ED25519, raw]), format: 'der', type: 'spki' });
    return crypto.verify(null, doc, key, sig);
  });
}

async function getCapped(fetchFn, url, max, host) {
  if (new URL(url).host !== host) throw new Error('asset on another host refused');
  const res = await fetchFn(url, { method: 'GET', redirect: 'manual', credentials: 'omit', signal: AbortSignal.timeout(10000) });
  if (res.status !== 200) throw new Error(`HTTP ${res.status}`);
  const buf = Buffer.from(await res.arrayBuffer());
  if (buf.length > max) throw new Error('response too large');
  return buf;
}

// checkForUpdate resolves to {available:false} or {available:true, version, notes}; it throws on any failure (the
// caller ignores failures: no feed yet, offline). opts: feed, repo, keys, fetchFn, currentVersion.
async function checkForUpdate({ feed = FEED, repo = REPO, keys = defaultKeys, fetchFn, currentVersion }) {
  const u = new URL(feed);
  if (u.protocol !== 'https:' && !(u.protocol === 'http:' && (u.hostname === '127.0.0.1' || u.hostname === 'localhost'))) throw new Error('feed must be https');
  if (u.username || u.password || u.search || u.hash) throw new Error('bad feed address');
  const base = feed.replace(/\/+$/, '');
  const list = JSON.parse((await getCapped(fetchFn, `${base}/repos/${repo}/releases?per_page=10`, MAX_LIST, u.host)).toString('utf8'));
  let best = null;
  for (const r of Array.isArray(list) ? list : []) {
    if (r.draft || r.prerelease || typeof r.tag_name !== 'string' || !r.tag_name.startsWith(TAG_PREFIX)) continue;
    const v = r.tag_name.slice(TAG_PREFIX.length);
    if (!parseVersion(v)) continue;
    if (!best || compareVersions(v, best.version) > 0) best = { version: v, assets: r.assets || [] };
  }
  if (!best) return { available: false };
  const asset = (name) => (best.assets.find((a) => a.name === name) || {}).browser_download_url;
  const mu = asset('MANIFEST.json');
  const su = asset('MANIFEST.json.sig');
  if (!mu || !su) throw new Error('release without manifest or signature');
  const raw = await getCapped(fetchFn, mu, MAX_MANIFEST, u.host);
  const sig = await getCapped(fetchFn, su, 4096, u.host);
  if (!verifySignature(keys, raw, sig.toString('utf8'))) throw new Error('signature does not verify');
  const m = JSON.parse(raw.toString('utf8'));
  if (m.schema !== 1 || m.version !== best.version || !(m.seq >= 1)) throw new Error('manifest does not match the release');
  if (compareVersions(m.version, currentVersion) <= 0) return { available: false };
  return { available: true, version: m.version, notes: String(m.notes || '').slice(0, 600) };
}

module.exports = { FEED, REPO, TAG_PREFIX, parseVersion, compareVersions, verifySignature, checkForUpdate };
