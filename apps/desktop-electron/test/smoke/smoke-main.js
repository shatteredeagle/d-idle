'use strict';
// Runs inside Electron (under Xvfb in CI): starts the real app against fake local servers and checks the security
// posture live. Exit 0 = pass. Not a unit test; run it with `npm run smoke`.
const http = require('node:http');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const assert = require('node:assert/strict');
const { app, BaseWindow } = require('electron');

const listen = (handler) => new Promise((res) => { const s = http.createServer(handler); s.listen(0, '127.0.0.1', () => res(s)); });
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
async function until(fn, what, ms = 15000) {
  const t = Date.now();
  while (Date.now() - t < ms) { if (await fn()) return; await sleep(100); }
  throw new Error('timeout: ' + what);
}

async function main() {
  const other = { hits: 0 };
  const B = await listen((_q, r) => { other.hits++; r.end('other'); });
  const portB = B.address().port;
  const page = (body) => `<!doctype html><title>t</title>${body}`;
  const A = await listen((q, r) => {
    if (q.url === '/watch') { r.writeHead(302, { location: '/login' }); return r.end(); } // unauthenticated visit: login redirect = up
    if (q.url === '/login') return r.end(page('login'));
    if (q.url === '/evil-nav') return r.end(page(`<script>setTimeout(()=>{location.href='http://127.0.0.1:${portB}/'},50)</script>`));
    if (q.url === '/redir') { r.writeHead(302, { location: `http://127.0.0.1:${portB}/` }); return r.end(); }
    r.end(page('hello ' + q.url));
  });
  const C = await listen((_q, r) => { r.writeHead(503); r.end('warming'); });
  const addrA = `127.0.0.1:${A.address().port}`;
  const originA = `http://${addrA}`;
  const addrC = `127.0.0.1:${C.address().port}`;

  const userData = fs.mkdtempSync(path.join(os.tmpdir(), 'd2e-smoke-'));
  const ctx = await require('../../src/app').start({ argv: ['--address', addrA, '--page', 'inventory'], userData, noSingleInstance: true });
  assert.ok(ctx, 'app started');
  const game = ctx.gameView.webContents;
  const helper = ctx.helperView.webContents;

  await until(() => ctx.gameShown && game.getURL().startsWith(originA) && !game.isLoading(), 'game page shown').catch((e) => {
    throw new Error(`${e.message} (state ${JSON.stringify(ctx.shell.state)}, url ${game.getURL()})`);
  });
  assert.equal(game.getURL(), originA + '/watch/inventory', 'opens the requested page on the exact origin');
  console.log('ok opens', game.getURL());

  assert.deepEqual(await game.executeJavaScript('[typeof window.d2idle, typeof require, typeof process, typeof module]'), ['undefined', 'undefined', 'undefined', 'undefined'], 'game page sees no bridge and no node');
  console.log('ok game view has no preload and no node');

  assert.equal(await game.executeJavaScript(`window.open('http://127.0.0.1:${portB}/') === null`), true, 'window.open returns null');
  await sleep(500);
  assert.equal(BaseWindow.getAllWindows().length, 1, 'no extra window');
  assert.equal(other.hits, 0, 'window.open went nowhere');
  console.log('ok window.open denied');

  await game.loadURL(originA + '/evil-nav');
  await sleep(800);
  assert.equal(other.hits, 0, 'page-driven navigation to another origin cancelled');
  assert.equal(game.getURL(), originA + '/evil-nav');
  console.log('ok will-navigate locked');

  await game.loadURL(originA + '/redir').catch(() => {});
  await sleep(500);
  assert.equal(other.hits, 0, 'redirect to another origin cancelled');
  console.log('ok will-redirect locked');

  const st = await helper.executeJavaScript('window.d2idle.getState()');
  assert.equal(st.address, originA);
  assert.deepEqual(await helper.executeJavaScript('[typeof require, typeof process, Object.keys(window.d2idle).sort().join()]'), ['undefined', 'undefined', 'getState,onState,openPage,retry,saveAddress']);
  console.log('ok helper bridge exposes the four calls');

  // Address change from the helper: a 503 server is "warming", a bad address is an error, the good one opens again.
  let r = await helper.executeJavaScript(`window.d2idle.saveAddress(${JSON.stringify(addrC)})`);
  assert.equal(r.status, 'warming');
  assert.equal(ctx.gameShown, false, 'helper is shown while the game is warming up');
  r = await helper.executeJavaScript(`window.d2idle.saveAddress('ftp://x:1')`);
  assert.match(r.error, /http/);
  console.log('ok warming and bad address');
  r = await helper.executeJavaScript(`window.d2idle.saveAddress(${JSON.stringify(addrA)})`);
  assert.equal(r.status, 'up');
  await until(() => ctx.gameShown, 'game shown again');
  console.log('ok back up');

  const cfgPath = path.join(userData, 'config.json');
  const mode = fs.statSync(cfgPath).mode & 0o777;
  assert.equal(JSON.parse(fs.readFileSync(cfgPath, 'utf8')).address, originA);
  if (process.platform !== 'win32') assert.equal(mode, 0o600);
  console.log('ok config 0600');
  assert.equal(other.hits, 0);
  console.log('SMOKE PASS');
}

main().then(() => app.exit(0), (e) => { console.error('SMOKE FAIL', e && e.stack || e); app.exit(1); });
