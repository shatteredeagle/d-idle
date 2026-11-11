'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const plat = require('../src/platform');
const { appMenuTemplate } = require('../src/menu');

test('macOS keeps running with no window while a tray exists; other systems quit', () => {
  assert.equal(plat.quitOnAllClosed('darwin', true), false);
  assert.equal(plat.quitOnAllClosed('darwin', false), true);
  assert.equal(plat.quitOnAllClosed('linux', true), true);
  assert.equal(plat.quitOnAllClosed('win32', true), true);
});

test('close hides to the tray; on macOS even with the toggle off; never while quitting', () => {
  const base = { hasTray: true, hideToTray: false, quitting: false };
  assert.equal(plat.hideOnClose({ ...base, platform: 'darwin' }), true);
  assert.equal(plat.hideOnClose({ ...base, platform: 'linux' }), false);
  assert.equal(plat.hideOnClose({ ...base, platform: 'linux', hideToTray: true }), true);
  assert.equal(plat.hideOnClose({ ...base, platform: 'darwin', quitting: true }), false);
  assert.equal(plat.hideOnClose({ ...base, platform: 'darwin', hasTray: false }), false);
});

test('tray image is a Template image on macOS and both sizes exist', () => {
  assert.equal(plat.trayImageName('darwin'), 'trayTemplate.png');
  assert.equal(plat.trayImageName('linux'), 'tray.png');
  const dir = path.join(__dirname, '..', 'build');
  const dim = (f) => { const b = fs.readFileSync(path.join(dir, f)); return [b.readUInt32BE(16), b.readUInt32BE(20)]; };
  assert.deepEqual(dim('trayTemplate.png'), [22, 22]);
  assert.deepEqual(dim('trayTemplate' + String.fromCharCode(64) + '2x.png'), [44, 44]);
});

test('dock hides and shows only on macOS', () => {
  const calls = [];
  const app = { dock: { hide: () => calls.push('hide'), show: () => calls.push('show') } };
  const mac = plat.dockControl(app, 'darwin');
  mac.hide(); mac.show();
  const lin = plat.dockControl(app, 'linux');
  lin.hide(); lin.show();
  assert.deepEqual(calls, ['hide', 'show']);
});

test('macOS app menu uses the standard roles and a Reload item; others keep the single menu', () => {
  const h = { openPage() {}, changeAddress() {}, reload() {}, quit() {} };
  const mac = appMenuTemplate(h, 'darwin');
  assert.deepEqual(mac.filter((m) => m.role).map((m) => m.role), ['appMenu', 'editMenu', 'windowMenu']);
  assert.ok(mac.find((m) => m.label === 'Pages').submenu.some((i) => i.label === 'Reload' && i.accelerator === 'CmdOrCtrl+R'));
  assert.equal(appMenuTemplate(h, 'linux').length, 1);
});

test('icns file is well formed', () => {
  const b = fs.readFileSync(path.join(__dirname, '..', 'build', 'icon.icns'));
  assert.equal(b.toString('ascii', 0, 4), 'icns');
  assert.equal(b.readUInt32BE(4), b.length);
  const types = [];
  for (let o = 8; o < b.length; o += b.readUInt32BE(o + 4)) types.push(b.toString('ascii', o, o + 4));
  assert.deepEqual(types, ['ic07', 'ic08', 'ic09', 'ic10']);
});

test('retry backoff grows and is capped; macOS message names Local Network', () => {
  const r = require('../src/retry');
  assert.deepEqual([0, 1, 2, 3, 4, 9].map(r.backoffMs), [3000, 6000, 12000, 24000, 30000, 30000]);
  assert.equal(r.isRealFailure(-3), false);
  assert.equal(r.isRealFailure(-102), true);
  assert.match(r.loadFailureMessage('darwin', 'ERR_ADDRESS_UNREACHABLE'), /Local Network/);
  assert.match(r.loadFailureMessage('linux', 'x'), /retries by itself/);
  assert.doesNotMatch(r.loadFailureMessage('linux', 'x'), /Local Network/);
});

test('log folders per platform', () => {
  const l = require('../src/logs');
  assert.equal(l.logDir('darwin', {}, '/h'), '/h/Library/Logs/d2idle-watch');
  assert.equal(l.logDir('linux', {}, '/h'), '/h/.local/state/d2idle-watch');
  assert.equal(l.logDir('linux', { XDG_STATE_HOME: '/s' }, '/h'), '/s/d2idle-watch');
  assert.equal(l.logPath('darwin', {}, '/h'), '/h/Library/Logs/d2idle-watch/main.log');
  assert.match(l.logDir('win32', { APPDATA: 'C:\\A' }, 'C:\\h').replace(/\\/g, '/'), /d2idle-watch\/logs$/);
});

test('args: chromium switches pass, --debug is ours, typos still fail', () => {
  const { parseArgs } = require('../src/args');
  assert.equal(parseArgs(['--enable-logging=stderr', '--v=1', '--disable-gpu', '--remote-debugging-port=9222', '--debug']).debug, true);
  assert.throws(() => parseArgs(['--enable-loging']), /unknown argument/);
  assert.throws(() => parseArgs(['--no-sandbox']), /unknown argument/);
});
