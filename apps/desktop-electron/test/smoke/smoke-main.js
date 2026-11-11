'use strict';
// Runs inside Electron (under Xvfb in CI): starts the real app against fake local servers and checks the security
// posture live. Exit 0 = pass. Not a unit test; run it with `npm run smoke`.
const http = require('node:http');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const assert = require('node:assert/strict');
const crypto = require('node:crypto');
const { app, BaseWindow } = require('electron');

if (process.env.D2E_SMOKE_NO_SANDBOX) app.commandLine.appendSwitch('no-sandbox'); // CI container without userns only

const listen = (handler) => new Promise((res) => { const s = http.createServer(handler); s.listen(0, '127.0.0.1', () => res(s)); });
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
async function until(fn, what, ms = 15000) {
  const t = Date.now();
  while (Date.now() - t < ms) { if (await fn()) return; await sleep(100); }
  throw new Error('timeout: ' + what);
}

// A window that is "up" but white shows one colour only; real content (text on a page) has several.
async function assertNotBlank(wc, what) {
  await sleep(300);
  const bmp = (await wc.capturePage()).toBitmap();
  const colours = new Set();
  for (let i = 0; i + 3 < bmp.length && colours.size < 3; i += 4 * 7) colours.add(bmp.readUInt32LE(i));
  assert.ok(colours.size > 1, `${what} is blank (one uniform colour)`);
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
  // D: answers the probe, but the first two page loads (navigations) die like a LAN load blocked by the macOS Local
  // Network permission; later loads work. The window must show the helper with a message, never stay blank.
  let dDocs = 0;
  const D = await listen((q, r) => {
    if (q.headers['sec-fetch-dest'] === 'document' && ++dDocs <= 2) return q.socket.destroy();
    if (q.url === '/watch') { r.writeHead(302, { location: '/login' }); return r.end(); }
    r.end(page('recovered ' + q.url));
  });
  const E = await listen((q, r) => {
    if (q.headers['sec-fetch-dest'] === 'document') return; // never answers: a silent stall
    r.writeHead(302, { location: '/login' }); r.end();
  });
  const addrE = `127.0.0.1:${E.address().port}`;
  const addrD = `127.0.0.1:${D.address().port}`;
  const addrA = `127.0.0.1:${A.address().port}`;
  const originA = `http://${addrA}`;
  const addrC = `127.0.0.1:${C.address().port}`;

  // Fake signed update feed (test key; the app is told to trust it only through opts.updateKeys).
  const { publicKey, privateKey } = crypto.generateKeyPairSync('ed25519');
  const manifest = Buffer.from(JSON.stringify({ schema: 1, version: '99.0.0', seq: 1, notes: 'smoke' }));
  let feedPort;
  const F = await listen((q, r) => {
    const base = `http://127.0.0.1:${feedPort}/dl`;
    if (q.url.includes('/releases')) return r.end(JSON.stringify([{ tag_name: 'desktop-v99.0.0', assets: [{ name: 'MANIFEST.json', browser_download_url: base + '/MANIFEST.json' }, { name: 'MANIFEST.json.sig', browser_download_url: base + '/MANIFEST.json.sig' }] }]));
    if (q.url.endsWith('MANIFEST.json')) return r.end(manifest);
    if (q.url.endsWith('.sig')) return r.end(crypto.sign(null, manifest, privateKey).toString('base64') + '\n');
    r.writeHead(404); r.end();
  });
  feedPort = F.address().port;
  let autostartOn = false;
  const fakeAutostart = { supported: true, get: () => autostartOn, set: (v) => { autostartOn = v; } };

  const userData = fs.mkdtempSync(path.join(os.tmpdir(), 'd2e-smoke-'));
  const ctx = await require('../../src/app').start({ argv: ['--address', addrA, '--page', 'inventory'], userData, logFile: path.join(userData, 'main.log'), noSingleInstance: true, stallMs: 4000, autostart: fakeAutostart, updateFeed: `http://127.0.0.1:${feedPort}`, updateKeys: [publicKey.export({ format: 'der', type: 'spki' }).subarray(-32).toString('base64')], updateDelayMs: 100 });
  assert.ok(ctx, 'app started');
  const game = ctx.gameView.webContents;
  const helper = ctx.helperView.webContents;

  await until(() => ctx.gameShown && game.getURL().startsWith(originA) && !game.isLoading(), 'game page shown').catch((e) => {
    throw new Error(`${e.message} (state ${JSON.stringify(ctx.shell.state)}, url ${game.getURL()})`);
  });
  assert.equal(game.getURL(), originA + '/watch/inventory', 'opens the requested page on the exact origin');
  console.log('ok opens', game.getURL());
  await assertNotBlank(game, 'game view');
  console.log('ok game view is not blank');

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
  await assertNotBlank(helper, 'helper view');
  console.log('ok helper view is not blank');
  r = await helper.executeJavaScript(`window.d2idle.saveAddress('ftp://x:1')`);
  assert.match(r.error, /http/);
  console.log('ok warming and bad address');
  r = await helper.executeJavaScript(`window.d2idle.saveAddress(${JSON.stringify(addrA)})`);
  assert.equal(r.status, 'up');
  await until(() => ctx.gameShown, 'game shown again');
  console.log('ok back up');

  // Failed first load: message in the helper, helper visible, game view hidden, then an automatic retry opens the page.
  r = await helper.executeJavaScript(`window.d2idle.saveAddress(${JSON.stringify(addrD)})`);
  await until(() => ctx.retryPending, 'first load failure noticed');
  assert.equal(ctx.gameShown, false, 'blank game view is not shown after a failed load');
  assert.equal(ctx.helperView.getVisible(), true, 'helper is visible after a failed load');
  assert.equal(ctx.gameView.getVisible(), false);
  await until(async () => /Waiting for the server/.test(await helper.executeJavaScript(`document.getElementById('error').textContent`)), 'helper shows the waiting message');
  assert.equal(await helper.executeJavaScript(`document.getElementById('error').hidden`), false);
  await until(() => ctx.gameShown && ctx.gameView.webContents.getURL().includes('/watch') && !ctx.gameView.webContents.isLoading(), 'automatic retry opens the page', 30000);
  assert.equal(await ctx.gameView.webContents.executeJavaScript('document.body.innerText.length > 0'), true, 'page body is not blank');
  assert.equal(ctx.retryPending, false);
  assert.ok(dDocs >= 3, 'the page was retried without a relaunch');
  assert.match(fs.readFileSync(ctx.logFile, 'utf8'), /game load failed/, 'the failure is in the log');
  console.log('ok failed load shows helper, retries, recovers');
  // Silent stall: no event at all; the watchdog must show the helper with a message.
  await helper.executeJavaScript(`window.d2idle.saveAddress(${JSON.stringify(addrE)})`);
  await until(() => ctx.retryPending && !ctx.gameShown && ctx.helperView.getVisible(), 'stalled load shows the helper', 12000);
  assert.match(ctx.loadMessage, /no answer after 4 s/);
  await assertNotBlank(helper, 'helper view during a stall');
  console.log('ok silent stall shows the helper');
  await helper.executeJavaScript(`window.d2idle.saveAddress(${JSON.stringify(addrA)})`);
  await until(() => ctx.gameShown && ctx.gameView.webContents.getURL().startsWith(originA), 'back on server A');

  const cfgPath = path.join(userData, 'config.json');
  const mode = fs.statSync(cfgPath).mode & 0o777;
  assert.equal(JSON.parse(fs.readFileSync(cfgPath, 'utf8')).address, originA);
  if (process.platform !== 'win32') assert.equal(mode, 0o600);
  console.log('ok config 0600');
  assert.equal(other.hits, 0);

  // Tray: constructed (Xvfb has no tray host, so only the API object and the menu template are checked).
  assert.ok(ctx.tray && !ctx.tray.isDestroyed(), 'tray constructed');
  const labels = () => ctx.trayTemplate.filter((i) => i.label).map((i) => i.label);
  assert.ok(labels().includes(`Online (${addrA})`), 'status line: ' + labels());
  await until(() => ctx.update.available, 'update check');
  assert.equal(ctx.update.version, '99.0.0');
  assert.ok(labels().includes('Update available: 99.0.0'), 'tray shows the update');
  console.log('ok tray menu and signed update check');

  ctx.api.toggleAutostart();
  assert.equal(autostartOn, true);
  assert.ok(ctx.trayTemplate.find((i) => i.label === 'Start with the system').checked);
  console.log('ok autostart toggle');

  // Closing hides to the tray; the window is not destroyed, and Open brings it back.
  ctx.window.close();
  await sleep(300);
  assert.equal(ctx.window.isDestroyed(), false);
  assert.equal(ctx.window.isVisible(), false, 'hidden to tray');
  ctx.api.show();
  assert.equal(ctx.window.isVisible(), true);
  ctx.api.toggleHideToTray();
  assert.equal(JSON.parse(fs.readFileSync(cfgPath, 'utf8')).hideToTray, false);
  ctx.api.toggleHideToTray();
  assert.equal(JSON.parse(fs.readFileSync(cfgPath, 'utf8')).hideToTray, true);
  console.log('ok hide to tray');
  console.log('SMOKE PASS');
}

main().then(() => app.exit(0), (e) => { console.error('SMOKE FAIL', e && e.stack || e); app.exit(1); });
