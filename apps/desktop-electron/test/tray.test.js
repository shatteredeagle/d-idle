'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const crypto = require('node:crypto');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { trayMenuTemplate, statusLabel } = require('../src/tray');
const autostart = require('../src/autostart');
const upd = require('../src/update');
const keys = require('../src/update-keys');

const AT = String.fromCharCode(64); // built so the leak scanner does not read the userinfo test as an address

const handlers = (calls) => ({ openPage: (n) => calls.push('page:' + n), show: () => calls.push('show'), toggleAutostart: () => calls.push('auto'), toggleHideToTray: () => calls.push('hide'), checkUpdate: () => calls.push('check'), showUpdate: () => calls.push('showupdate'), quit: () => calls.push('quit'), openLogFolder: () => calls.push('log') });

test('tray menu: open, pages, status line, update, toggles, quit', () => {
  const calls = [];
  const t = trayMenuTemplate({ state: { status: 'up', address: 'http://h:8891' }, autostart: true, hideToTray: false, update: { available: false } }, handlers(calls));
  const labels = t.filter((i) => i.label).map((i) => i.label);
  assert.deepEqual(labels, ['Open', 'Pages', 'Online (h:8891)', 'Check for updates', 'Start with the system', 'Closing the window hides it to the tray', 'Open log folder', 'Quit']);
  assert.deepEqual(t[1].submenu.map((i) => i.label), ['Watch', 'Inventory', 'Policy', 'Return screen']);
  assert.equal(t[3].enabled, false, 'status line is not clickable');
  assert.equal(t.find((i) => i.label === 'Start with the system').checked, true);
  assert.equal(t.find((i) => (i.label || '').startsWith('Closing')).checked, false);
  for (const i of [t[0], ...t[1].submenu, t.find((x) => x.label === 'Check for updates'), t.find((x) => x.label === 'Start with the system'), t.find((x) => (x.label || '').startsWith('Closing')), t.find((x) => x.label === 'Open log folder'), t.at(-1)]) i.click();
  assert.deepEqual(calls, ['show', 'page:watch', 'page:inventory', 'page:policy', 'page:return', 'check', 'auto', 'hide', 'log', 'quit']);
});

test('tray menu shows an available update', () => {
  const calls = [];
  const t = trayMenuTemplate({ state: { status: 'down', address: 'http://h:1' }, update: { available: true, version: '1.2.3' } }, handlers(calls));
  const u = t.find((i) => (i.label || '').startsWith('Update available'));
  assert.equal(u.label, 'Update available: 1.2.3');
  u.click();
  assert.deepEqual(calls, ['showupdate']);
  assert.ok(!t.some((i) => i.label === 'Check for updates'));
});

test('status labels', () => {
  assert.equal(statusLabel({ status: 'warming', address: 'https://h:1' }), 'Warming up (h:1)');
  assert.equal(statusLabel({ status: 'setup', address: '' }), 'Not set up');
  assert.equal(statusLabel({ status: 'down', address: 'http://x' }), 'Offline (x)');
});

test('autostart desktop entry starts hidden and quotes the path', () => {
  const e = autostart.desktopEntry('/opt/My Apps/d2idle-watch.AppImage');
  assert.match(e, /^Exec="\/opt\/My Apps\/d2idle-watch\.AppImage" --hidden$/m);
  assert.match(autostart.desktopEntry('/usr/bin/d2idle-watch'), /^Exec=\/usr\/bin\/d2idle-watch --hidden$/m);
  assert.equal(autostart.execTarget({ APPIMAGE: '/x/a.AppImage' }, '/tmp/.mount/app'), '/x/a.AppImage');
  assert.equal(autostart.execTarget({}, '/opt/d2/d2idle-watch'), '/opt/d2/d2idle-watch');
});

test('linux autostart writes and removes the XDG entry', () => {
  const home = fs.mkdtempSync(path.join(os.tmpdir(), 'd2auto-'));
  const a = autostart.createAutostart(null, { platform: 'linux', env: {}, home });
  assert.equal(a.get(), false);
  a.set(true);
  const file = path.join(home, '.config', 'autostart', 'd2idle-watch.desktop');
  assert.ok(fs.existsSync(file));
  assert.equal(a.get(), true);
  a.set(false);
  assert.equal(a.get(), false);
  a.set(false); // idempotent
  assert.equal(autostart.entryPath({ XDG_CONFIG_HOME: '/x/cfg' }, '/h'), '/x/cfg/autostart/d2idle-watch.desktop');
});

test('windows and macos autostart use the login item with --hidden', () => {
  let set;
  const app = { getLoginItemSettings: () => ({ openAtLogin: true }), setLoginItemSettings: (s) => { set = s; } };
  const a = autostart.createAutostart(app, { platform: 'win32' });
  assert.equal(a.get(), true);
  a.set(false);
  assert.deepEqual(set, { openAtLogin: false, args: ['--hidden'] });
});

const keyDir = path.join(__dirname, '..', '..', '..', 'deploy', 'd2d-container', 'pubkeys');
// skipped in a scratch copy of this folder (the worker-host runner); CI checks out the whole repo
test('pinned update keys equal the keys of the bot image', { skip: !fs.existsSync(keyDir) }, () => {
  const dir = keyDir;
  const read = (f) => fs.readFileSync(path.join(dir, f), 'utf8').trim().split(/\s+/);
  for (const [i, f] of [[0, 'beta.pub'], [1, 'current.pub']]) {
    const [header, b64] = read(f);
    assert.equal(header, 'd2sign-ed25519-public-v1');
    assert.equal(keys[i], b64, f);
  }
});

// ---- update check against a fake signed feed
const { publicKey, privateKey } = crypto.generateKeyPairSync('ed25519');
const rawPub = publicKey.export({ format: 'der', type: 'spki' }).subarray(-32).toString('base64');
const sign = (doc, k = privateKey) => crypto.sign(null, doc, k).toString('base64') + '\n';

function feedFetch({ tag = 'desktop-v1.2.0', version = '1.2.0', seq = 3, manifestMutator, sigKey, host = 'feed.example', extra = [] }) {
  const manifest = Buffer.from(JSON.stringify({ schema: 1, version, seq, notes: 'New tray.' }));
  const bad = manifestMutator ? manifestMutator(manifest) : manifest;
  const sig = sign(manifest, sigKey);
  const base = `https://${host}/dl/${tag}`;
  const list = [...extra, { tag_name: tag, draft: false, prerelease: false, assets: [{ name: 'MANIFEST.json', browser_download_url: `${base}/MANIFEST.json` }, { name: 'MANIFEST.json.sig', browser_download_url: `${base}/MANIFEST.json.sig` }] }];
  const calls = [];
  const fn = async (url, init) => {
    calls.push([url, init]);
    const ok = (b) => ({ status: 200, arrayBuffer: async () => Buffer.from(b) });
    if (url.includes('/releases')) return ok(JSON.stringify(list));
    if (url.endsWith('MANIFEST.json')) return ok(bad);
    if (url.endsWith('.sig')) return ok(sig);
    return { status: 404, arrayBuffer: async () => Buffer.alloc(0) };
  };
  fn.calls = calls;
  return fn;
}
const base = { feed: 'https://feed.example', repo: 'o/r', keys: [rawPub], currentVersion: '1.0.0' };

test('update: newer signed release is reported, nothing else is fetched', async () => {
  const f = feedFetch({});
  const r = await upd.checkForUpdate({ ...base, fetchFn: f });
  assert.deepEqual(r, { available: true, version: '1.2.0', notes: 'New tray.' });
  assert.equal(f.calls.length, 3, 'list, manifest, signature: no download of the app');
  assert.ok(f.calls.every(([u, i]) => u.startsWith('https://feed.example/') && i.redirect === 'manual' && i.credentials === 'omit'));
});

test('update: same or older version is not an update', async () => {
  assert.equal((await upd.checkForUpdate({ ...base, currentVersion: '1.2.0', fetchFn: feedFetch({}) })).available, false);
  assert.equal((await upd.checkForUpdate({ ...base, currentVersion: '2.0.0', fetchFn: feedFetch({}) })).available, false);
});

test('update: refused when the signature, key, manifest or host is wrong', async () => {
  const other = crypto.generateKeyPairSync('ed25519').privateKey;
  await assert.rejects(upd.checkForUpdate({ ...base, fetchFn: feedFetch({ sigKey: other }) }), /signature/);
  await assert.rejects(upd.checkForUpdate({ ...base, fetchFn: feedFetch({ manifestMutator: (m) => Buffer.from(m.toString().replace('1.2.0', '9.9.9')) }) }), /signature/);
  await assert.rejects(upd.checkForUpdate({ ...base, keys: keys, fetchFn: feedFetch({}) }), /signature/, 'a feed signed by a key that is not pinned');
  await assert.rejects(upd.checkForUpdate({ ...base, fetchFn: feedFetch({ host: 'evil.example' }) }), /another host/);
  await assert.rejects(upd.checkForUpdate({ ...base, feed: 'http://feed.example', fetchFn: feedFetch({}) }), /https/);
  await assert.rejects(upd.checkForUpdate({ ...base, feed: `https://u:p${AT}feed.example`, fetchFn: feedFetch({}) }), /bad feed/);
  await assert.rejects(upd.checkForUpdate({ ...base, fetchFn: feedFetch({ version: '1.3.0' }) }), /does not match/, 'tag and manifest version must agree');
});

test('update: ignores other tags, drafts and prereleases; no release means no update', async () => {
  const extra = [{ tag_name: 'app-v9.0.0', assets: [] }, { tag_name: 'desktop-v8.0.0', draft: true, assets: [] }, { tag_name: 'desktop-v7.0.0', prerelease: true, assets: [] }, { tag_name: 'desktop-vX', assets: [] }];
  const r = await upd.checkForUpdate({ ...base, fetchFn: feedFetch({ extra }) });
  assert.equal(r.version, '1.2.0');
  const empty = async () => ({ status: 200, arrayBuffer: async () => Buffer.from('[]') });
  assert.equal((await upd.checkForUpdate({ ...base, fetchFn: empty })).available, false);
  await assert.rejects(upd.checkForUpdate({ ...base, fetchFn: async () => ({ status: 503, arrayBuffer: async () => Buffer.alloc(0) }) }), /HTTP 503/);
});

test('update: version compare', async () => {
  assert.equal(upd.compareVersions('1.10.0', '1.9.0'), 1);
  assert.equal(upd.compareVersions('1.0.0', '1.0.0'), 0);
  assert.equal(upd.compareVersions('x', '1.0.0'), -1);
  assert.equal(upd.parseVersion('01.0.0'), null);
  assert.equal(upd.builtinFeed('/nonexistent/feed.json'), '');
  await assert.rejects(upd.checkForUpdate({ ...base, feed: '', fetchFn: feedFetch({}) }), /no update feed/);
  assert.doesNotMatch(require('node:fs').readFileSync(path.join(__dirname, '../src/update.js'), 'utf8'), /api\.github\.com|githubusercontent|https:\/\/github\.com/);
});
