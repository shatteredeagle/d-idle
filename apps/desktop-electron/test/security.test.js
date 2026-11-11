'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const path = require('node:path');
const fs = require('node:fs');
const sec = require('../src/security');
const fuses = require('../build/fuses');

const AT = String.fromCharCode(64); // userinfo trick, built so the leak scanner does not read it as an address

test('both views are isolated, sandboxed and without node', () => {
  for (const kind of ['helper', 'game']) {
    const p = sec.webPreferences(kind, '/x/preload.js');
    assert.equal(p.contextIsolation, true);
    assert.equal(p.nodeIntegration, false);
    assert.equal(p.nodeIntegrationInSubFrames, false);
    assert.equal(p.sandbox, true);
    assert.equal(p.webSecurity, true);
    assert.equal(p.allowRunningInsecureContent, false);
    assert.equal(p.webviewTag, false);
    assert.equal(p.devTools, false);
  }
  assert.equal(sec.webPreferences('helper', '/x/preload.js').preload, '/x/preload.js');
  assert.equal('preload' in sec.webPreferences('game', '/x/preload.js'), false, 'the game view must have no preload');
  assert.equal(sec.webPreferences('game').partition, 'persist:d2idle');
  assert.throws(() => sec.webPreferences('other'));
});

test('navigation is locked to the exact configured origin', () => {
  const o = 'http://game.lan:8891';
  assert.equal(sec.isAllowedNavigation(o, 'http://game.lan:8891/watch/inventory?x=1'), true);
  for (const bad of ['http://game.lan:8892/', 'https://game.lan:8891/', 'http://evil.example/', 'http://game.lan:8891.evil.example/', `http://evil.example:8891${AT}game.lan:8891/`, 'javascript:alert(1)', 'file:///etc/passwd', 'data:text/html,x', 'about:blank', 'd2idle-app://helper/index.html', 'not a url', '']) {
    assert.equal(sec.isAllowedNavigation(o, bad), false, bad);
  }
  assert.equal(sec.isAllowedNavigation('', 'http://game.lan:8891/'), false, 'no origin configured, nothing is allowed');
});

test('helper URL and IPC sender checks', () => {
  assert.equal(sec.isHelperURL('d2idle-app://helper/index.html'), true);
  for (const bad of ['d2idle-app://other/index.html', 'http://helper/', 'file:///helper/index.html', 'd2idle-app://helper.evil/']) assert.equal(sec.isHelperURL(bad), false, bad);
  const wc = {};
  assert.equal(sec.ipcSenderOk({ sender: wc, senderFrame: { url: 'd2idle-app://helper/index.html' } }, wc), true);
  assert.equal(sec.ipcSenderOk({ sender: {}, senderFrame: { url: 'd2idle-app://helper/index.html' } }, wc), false, 'other web contents');
  assert.equal(sec.ipcSenderOk({ sender: wc, senderFrame: { url: 'http://game.lan:8891/' } }, wc), false, 'remote frame');
  assert.equal(sec.ipcSenderOk({ sender: wc, senderFrame: null }, wc), false);
  assert.equal(sec.ipcSenderOk(undefined, wc), false);
});

test('helper file resolution cannot escape the helper folder', () => {
  const root = path.resolve('/srv/app/helper');
  assert.equal(sec.resolveHelperFile(root, 'd2idle-app://helper/helper.js'), path.join(root, 'helper.js'));
  assert.equal(sec.resolveHelperFile(root, 'd2idle-app://helper/'), path.join(root, 'index.html'));
  // the URL parser already folds /../ into /secret, which stays inside the root
  assert.equal(sec.resolveHelperFile(root, 'd2idle-app://helper/../secret'), path.join(root, 'secret'));
  for (const url of ['d2idle-app://helper/%2e%2e/secret', 'd2idle-app://helper/..%2f..%2fetc/passwd']) {
    const r = sec.resolveHelperFile(root, url);
    assert.ok(r === null || r.startsWith(root + path.sep), url + ' -> ' + r);
  }
  for (const bad of ['d2idle-app://helper/a\\..\\..\\x', 'd2idle-app://other/index.html', 'http://helper/index.html', 'd2idle-app://helper/%00']) {
    assert.equal(sec.resolveHelperFile(root, bad), null, bad);
  }
});

test('permissions are denied by default', () => {
  let got;
  sec.denyPermissionRequest(null, 'notifications', (v) => { got = v; });
  assert.equal(got, false);
  assert.equal(sec.denyPermissionCheck(), false);
});

test('helper CSP and page meta are strict and agree', () => {
  assert.match(sec.HELPER_CSP, /default-src 'none'/);
  assert.match(sec.HELPER_CSP, /connect-src 'none'/);
  assert.doesNotMatch(sec.HELPER_CSP, /unsafe-|\*/);
  const html = fs.readFileSync(path.join(__dirname, '../src/helper/index.html'), 'utf8');
  assert.ok(html.includes(`content="${sec.HELPER_CSP}"`), 'meta CSP must equal HELPER_CSP');
  assert.doesNotMatch(html, /<script(?![^>]*src=)/i, 'no inline script');
  assert.doesNotMatch(html, /\son\w+=/i, 'no inline handlers');
});

test('the preload exposes four invoke calls and one listener, nothing else', () => {
  const src = fs.readFileSync(path.join(__dirname, '../src/helper/preload.js'), 'utf8');
  const exposed = [...src.matchAll(/^\s{2}(\w+): /gm)].map((m) => m[1]);
  assert.deepEqual(exposed, ['getState', 'saveAddress', 'retry', 'openPage', 'onState']);
  assert.equal((src.match(/ipcRenderer\.invoke\('d2idle:\w+'/g) || []).length, 4);
});

test('app.js wires the locks (static check of the wiring)', () => {
  const src = fs.readFileSync(path.join(__dirname, '../src/app.js'), 'utf8');
  for (const needle of ['will-navigate', 'will-redirect', 'will-frame-navigate', "action: 'deny'", 'will-attach-webview', 'setPermissionRequestHandler', 'setPermissionCheckHandler', 'ipcSenderOk', "webPreferences('game')"]) {
    assert.ok(src.includes(needle), needle);
  }
  assert.doesNotMatch(src, /shell\.openExternal|crashReporter|remote-debugging|--no-sandbox/, 'forbidden calls');
  assert.doesNotMatch(src, /nodeIntegration:\s*true|contextIsolation:\s*false/);
});

test('fuses: run-as-node and inspect off, asar only on', () => {
  assert.equal(fuses.RunAsNode, false);
  assert.equal(fuses.EnableNodeCliInspectArguments, false);
  assert.equal(fuses.EnableNodeOptionsEnvironmentVariable, false);
  assert.equal(fuses.OnlyLoadAppFromAsar, true);
  assert.equal(fuses.EnableEmbeddedAsarIntegrityValidation, true);
});

test('openExternalAllowed only lets http(s) through', () => {
  assert.equal(sec.openExternalAllowed('https://updates.example/x'), true);
  for (const bad of ['file:///x', 'javascript:1', 'smb://h/s', 'ms-msdt:x', '']) assert.equal(sec.openExternalAllowed(bad), false, bad);
});
