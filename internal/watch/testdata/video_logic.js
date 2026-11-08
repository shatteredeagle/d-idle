// Runs web/app.js against a stub DOM and checks the video panel: reconnect backoff, collapse memory, fullscreen.
// Usage: node video_logic.js path/to/app.js
const fs = require('fs'), vm = require('vm'), assert = require('assert');
process.on('unhandledRejection', () => {});
const store = {}, timers = [], els = {};
function mk(id) {
  const listeners = {}, cls = new Set();
  const e = {
    id, hidden: false, dataset: {}, style: {}, children: [], textContent: '', value: '', checked: false, firstChild: null,
    classList: { add: c => cls.add(c), remove: c => cls.delete(c), contains: c => cls.has(c), toggle(c, on) { if (on === undefined) on = !cls.has(c); on ? cls.add(c) : cls.delete(c); return on; } },
    attrs: {}, setAttribute(k, v) { this.attrs[k] = String(v); }, removeAttribute(k) { delete this.attrs[k]; }, getAttribute(k) { return this.attrs[k]; },
    addEventListener(t, f) { (listeners[t] = listeners[t] || []).push(f); },
    fire(t) { (listeners[t] || []).forEach(f => f({ preventDefault() {}, key: '' })); },
    appendChild() {}, removeChild() {}, querySelector: () => null, querySelectorAll: () => [], focus() {}, requestFullscreen() { fs_.requested = true; },
  };
  return e;
}
const fs_ = { requested: false };
const document = {
  getElementById: id => els[id] || (els[id] = mk(id)),
  createElement: () => mk('x'), querySelector: () => null, querySelectorAll: () => [],
  addEventListener() {}, documentElement: { dataset: {} }, fullscreenElement: null, exitFullscreen() {},
};
const ctx = {
  document, console, URLSearchParams, Date, Math, JSON, Promise,
  location: { search: '?t=tok', hash: '', protocol: 'http:', hostname: 'h' },
  localStorage: { getItem: k => (k in store ? store[k] : null), setItem: (k, v) => { store[k] = v; } },
  fetch: u => Promise.resolve({ json: () => Promise.resolve(u.startsWith('/api/config') ? { frameUrl: '/stream' } : {}), ok: true }),
  setTimeout: (f, ms) => { timers.push({ f, ms }); return timers.length; }, setInterval: () => 0, clearTimeout() {}, clearInterval() {},
  EventSource: undefined,
};
ctx.window = ctx;
const run = async (preset) => {
  Object.keys(store).forEach(k => delete store[k]); Object.assign(store, preset); timers.length = 0;
  for (const k of Object.keys(els)) delete els[k];
  vm.runInNewContext(fs.readFileSync(process.argv[2], 'utf8'), ctx);
  await new Promise(r => setImmediate(r));
};
(async () => {
  await run({});
  const shot = els.shot, pane = els.gamepane;
  assert.strictEqual(pane.hidden, false, 'pane shown when the server offers a stream');
  assert.ok(/^\/stream\?t=tok&r=\d+$/.test(shot.src), 'src ' + shot.src);
  // errors back off 1 s, 2 s, ... capped at 15 s, and the page keeps retrying
  const waits = [];
  for (let i = 0; i < 20; i++) { timers.length = 0; shot.fire('error'); waits.push(timers.filter(t => t.ms >= 1000 && t.ms <= 15000).pop().ms); }
  assert.deepStrictEqual(waits.slice(0, 3), [1000, 2000, 3000]);
  assert.strictEqual(waits[19], 15000);
  shot.fire('load'); timers.length = 0; shot.fire('error');
  assert.strictEqual(timers.filter(t => t.ms <= 15000).pop().ms, 1000, 'a good frame resets the backoff');
  // collapse is remembered
  els['video-collapse'].fire('click');
  assert.ok(pane.classList.contains('collapsed')); assert.strictEqual(store.d2videocollapsed, '1');
  await run({ d2videocollapsed: '1' });
  assert.ok(els.gamepane.classList.contains('collapsed'), 'collapsed state restored');
  els['video-collapse'].fire('click');
  assert.ok(!els.gamepane.classList.contains('collapsed')); assert.strictEqual(store.d2videocollapsed, '0');
  els['video-full'].requestFullscreen = els.gamepane.requestFullscreen; // the click asks the pane, not the button
  els['video-full'].fire('click'); assert.ok(fs_.requested, 'fullscreen requested on the pane');
  console.log('ok');
})().catch(e => { console.error(e); process.exit(1); });
