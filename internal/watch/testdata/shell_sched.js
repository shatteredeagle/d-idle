// The shell's fetch scheduler (web/shell.js makeScheduler) in node: panels that read the same address share one
// request, every panel leaves the loading state after the first poll cycle, and a request that hangs holds one
// connection, not one per panel, so the other endpoints still load. Usage: node shell_sched.js path/to/web/shell.js
const assert = require('assert'), fs = require('fs'), vm = require('vm');
const src = fs.readFileSync(process.argv[2], 'utf8');
const a = src.indexOf('function makeScheduler()'), b = src.indexOf('// ---- layout manifest');
assert.ok(a > 0 && b > a, 'makeScheduler not found');
const calls = [], waiting = {};
let timers = [], now = 0;
const ctx = {
  SAFETY_POLL: 5000, FAIL_WAIT: 4000, FETCH_TIMEOUT: 8000, MAX_FAILS: 5, Map, Math, Error, Promise,
  document: { hidden: false }, window: {}, performance: { now: () => now },
  setTimeout: (f, ms) => { timers.push({ f, at: now + ms }); return timers.length; }, clearTimeout() {},
  AbortController: function () { this.signal = {}; this.abort = () => {}; },
  keepFocus: (root, fn) => fn(), syncBand() {},
  fetch: (url) => { calls.push(url); return new Promise((res, rej) => { (waiting[url] = waiting[url] || []).push({ res, rej }); }); },
};
vm.runInNewContext(src.slice(a, b) + '\n;this.makeScheduler = makeScheduler;', ctx);
const reply = (url, data) => (waiting[url] || []).splice(0).forEach((w) => w.res({ ok: true, json: async () => data }));
const settle = async () => { for (let i = 0; i < 8; i++) await null; };
const mkSlot = (id, endpoint, poll) => { const s = { m: { id, endpoint, poll, band: 'side' }, root: { dataset: { state: 'loading' }, parentElement: null }, got: [], api: {} }; s.api.update = (d, err) => s.got.push(err ? 'err' : d.v); return s; };
// run the pending timers once (the tick re-arms itself)
const tick = () => { const due = timers; timers = []; due.forEach((t) => t.f()); };

(async () => {
  const sched = ctx.makeScheduler();
  const hero = ['hero-vitals', 'activity', 'why', 'hero-meters', 'hero-detail'].map((id) => mkSlot(id, '/api/hero', 2));
  const gear = mkSlot('gear', '/api/repair', 5), tally = mkSlot('tally', '/api/tally', 0);
  [...hero, gear, tally].forEach((s) => sched.add(s));
  sched.start();
  assert.deepStrictEqual(calls.slice().sort(), ['/api/hero', '/api/repair', '/api/tally'], 'five panels on /api/hero are one request');
  // /api/hero hangs (a busy bot); the other endpoints answer and their panels leave the skeleton
  reply('/api/repair', { v: 'r1' }); reply('/api/tally', { v: 't1' });
  await settle();
  assert.strictEqual(gear.root.dataset.state, 'ready'); assert.strictEqual(tally.root.dataset.state, 'ready');
  now += 1000; tick(); await settle();
  assert.strictEqual(calls.filter((u) => u === '/api/hero').length, 1, 'a hanging request is not asked again while it is in flight');
  // the reply comes: all five panels get it and none stays in the loading state
  reply('/api/hero', { v: 'h1' });
  await settle();
  for (const s of hero) { assert.deepStrictEqual(s.got, ['h1'], s.m.id); assert.strictEqual(s.root.dataset.state, 'ready', s.m.id + ' left the skeleton'); }
  // the next cycle is again one request for the five
  now += 2100; tick(); await settle();
  assert.strictEqual(calls.filter((u) => u === '/api/hero').length, 2);
  reply('/api/hero', { v: 'h2' }); await settle();
  assert.ok(hero.every((s) => s.got.join() === 'h1,h2'));
  // a failed request reaches every sharer as a failure and none of them wedges
  now += 2100; tick(); await settle();
  (waiting['/api/hero'] || []).splice(0).forEach((w) => w.rej(new Error('boom')));
  await settle();
  assert.ok(hero.every((s) => s.got.join() === 'h1,h2,err' && !s.inflight), 'a failure releases every slot');
  // a panel whose update throws does not take the others down
  hero[0].api.update = () => { throw new Error('render'); };
  now += 9000; tick(); await settle();
  reply('/api/hero', { v: 'h3' }); await settle();
  assert.ok(hero.slice(1).every((s) => s.got[s.got.length - 1] === 'h3'), 'one throwing panel does not stop the rest');
  assert.strictEqual(hero[0].inflight, false);
  // the page has one event stream: the scheduler takes the one app.js opened
  let made = 0;
  ctx.EventSource = ctx.window.EventSource = function () { made++; this.addEventListener = () => {}; };
  ctx.window.d2events = { addEventListener() {} };
  ctx.makeScheduler();
  assert.strictEqual(made, 0, 'the shell does not open a second event stream');
  delete ctx.window.d2events;
  ctx.makeScheduler(); ctx.makeScheduler();
  assert.strictEqual(made, 1, 'without app.js it opens exactly one');
  console.log('ok');
})().catch((e) => { console.error(e); process.exit(1); });
