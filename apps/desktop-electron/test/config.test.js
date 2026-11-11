'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const config = require('../src/config');
const { parseArgs, userArgs } = require('../src/args');
const { probe } = require('../src/probe');
const { Shell } = require('../src/state');
const { appMenuTemplate, pageItems } = require('../src/menu');

test('config: first run empty, round trip, 0600, unknown keys kept', () => {
  const dir = path.join(fs.mkdtempSync(path.join(os.tmpdir(), 'd2cfg-')), 'cfg');
  assert.deepEqual(config.load(dir), {});
  config.save(dir, { address: 'http://h:1', other: 7 });
  const c = config.load(dir);
  assert.equal(c.address, 'http://h:1');
  assert.equal(c.other, 7);
  if (process.platform !== 'win32') assert.equal(fs.statSync(path.join(dir, 'config.json')).mode & 0o777, 0o600);
  fs.writeFileSync(path.join(dir, 'config.json'), '[]');
  assert.throws(() => config.load(dir));
  fs.writeFileSync(path.join(dir, 'config.json'), '{bad');
  assert.throws(() => config.load(dir));
});

test('config: the Go launcher file is read as is', () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'd2cfg-'));
  fs.writeFileSync(path.join(dir, 'config.json'), '{\n  "address": "http://h:1"\n}\n');
  assert.equal(config.load(dir).address, 'http://h:1');
});

test('config: window size is bounds-checked', () => {
  assert.deepEqual(config.windowSize({}), { width: 1280, height: 800 });
  assert.deepEqual(config.windowSize({ window: { width: 1000, height: 700 } }), { width: 1000, height: 700 });
  assert.deepEqual(config.windowSize({ window: { width: 10, height: 700 } }), { width: 1280, height: 800 });
  assert.deepEqual(config.windowSize({ window: { width: '1000', height: 700 } }), { width: 1280, height: 800 });
});

test('args', () => {
  assert.deepEqual(parseArgs([]), { setup: false, print: false, address: '', page: '' });
  assert.deepEqual(parseArgs(['--setup', '--page', 'inventory', '--address', 'H:8891', '--print']), { setup: true, print: true, address: 'http://h:8891', page: 'inventory' });
  assert.equal(parseArgs(['--page=policy']).page, 'policy');
  for (const bad of [['--page', 'x'], ['--page'], ['--address'], ['--address', 'ftp://x:1'], ['--remote-debugging-port=9222'], ['--inspect'], ['--no-sandbox'], ['stray']]) {
    assert.throws(() => parseArgs(bad), undefined, bad.join(' '));
  }
  assert.deepEqual(userArgs(['electron', '.', '--setup'], false), ['--setup']);
  assert.deepEqual(userArgs(['app', '--setup'], true), ['--setup']);
});

test('probe: up, login redirect is up, 5xx warming, error down', async () => {
  const mk = (status) => async () => ({ status, body: null });
  assert.equal(await probe(mk(200), 'http://h:1'), 'up');
  assert.equal(await probe(mk(302), 'http://h:1'), 'up');
  assert.equal(await probe(mk(404), 'http://h:1'), 'up');
  assert.equal(await probe(mk(502), 'http://h:1'), 'warming');
  assert.equal(await probe(async () => { throw new Error('refused'); }, 'http://h:1'), 'down');
  assert.equal(await probe(async () => { throw new Error('Redirect was cancelled'); }, 'http://h:1'), 'up', 'Electron rejects manual redirects');
  let seen;
  await probe(async (u, init) => { seen = [u, init]; return { status: 200 }; }, 'http://h:1');
  assert.equal(seen[0], 'http://h:1/watch');
  assert.equal(seen[1].redirect, 'manual');
  assert.equal(seen[1].credentials, 'omit');
});

test('shell state machine', async () => {
  const seen = [];
  let status = 503;
  const sh = new Shell({ fetchFn: async () => ({ status }), onChange: (s) => seen.push(s.status) });
  assert.equal(sh.state.status, 'setup');
  await sh.tick();
  assert.equal(sh.state.status, 'setup', 'no address: nothing to probe');
  sh.setAddress('http://h:1');
  await sh.tick();
  assert.equal(sh.state.status, 'warming');
  status = 200;
  await sh.tick();
  assert.equal(sh.state.status, 'up');
  sh.setError('bad');
  assert.equal(sh.state.error, 'bad');
  assert.deepEqual(seen, ['checking', 'warming', 'up', 'up']);
  sh.state = { ...sh.state, status: 'setup' };
  status = 503;
  await sh.tick();
  assert.equal(sh.state.status, 'setup', 'setup holds while the user types');
});

test('menu has the four pages, change address, reload, quit', () => {
  const calls = [];
  const h = { openPage: (n) => calls.push(n), changeAddress: () => calls.push('addr'), reload: () => calls.push('reload'), quit: () => calls.push('quit') };
  const items = appMenuTemplate(h)[0].submenu.filter((i) => i.click);
  assert.deepEqual(items.map((i) => i.label), ['Watch', 'Inventory', 'Policy', 'Return screen', 'Change address', 'Reload', 'Quit']);
  items.forEach((i) => i.click());
  assert.deepEqual(calls, ['watch', 'inventory', 'policy', 'return', 'addr', 'reload', 'quit']);
  assert.equal(pageItems(h).length, 4);
});
