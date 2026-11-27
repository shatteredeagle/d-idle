// dBusy (web/toast.js) in the stub DOM: a click changes the button before the request returns, a second tap is swallowed,
// success and failure end in done / failed and then clear, a redrawn twin of the button gets the state again.
'use strict';
const path = require('path');
const { makeDoc } = require('./nav_dom.cjs');
const T = require(path.join(process.argv[2], 'toast.cjs'));
const fails = [];
const ok = (c, m) => { if (!c) fails.push(m); };
const doc = makeDoc();
const timers = []; let clock = 0;
const later = (f, ms) => { const t = { f, at: clock + ms }; timers.push(t); return t; };
const clear = (t) => { const i = timers.indexOf(t); if (i >= 0) timers.splice(i, 1); };
const advance = (ms) => { clock += ms; for (const t of timers.filter((x) => x.at <= clock)) { clear(t); t.f(); } };
const B = T.createBusy(doc, { later, clear });
const flush = () => new Promise((r) => setImmediate(r));

(async () => {
  // explicit run
  const b = doc.createElement('button'); b.dataset.key = 'handin'; doc.body.appendChild(b);
  let release; const slow = new Promise((r) => { release = r; });
  const p = B.run(b, () => slow);
  ok(b.dataset.busy === 'working' && b.attrs['aria-busy'] === 'true', 'run: working state is set synchronously, before the request returns');
  let calls = 0; B.run(b, () => { calls++; return slow; });
  ok(calls === 0, 'run: a second tap while working does nothing');
  release({ ok: true }); await p;
  ok(b.dataset.busy === 'done', 'run: done after success');
  advance(2000);
  ok(b.dataset.busy === undefined && b.attrs['aria-busy'] === undefined, 'run: done clears by itself');
  await B.run(b, () => Promise.resolve({ ok: false, error: 'no' }));
  ok(b.dataset.busy === 'failed', 'run: ok:false is failed');
  advance(5000); ok(b.dataset.busy === undefined, 'run: failed clears');
  await B.run(b, () => Promise.reject(new Error('x'))).catch(() => {});
  ok(b.dataset.busy === 'failed', 'run: a rejected request is failed');
  advance(5000);

  // a redraw replaces the button while it works: the twin with the same key shows the state
  const win = { MutationObserver: null };
  doc.querySelector = (sel) => { const m = /data-key="([^"]+)"/.exec(sel); return m ? doc.all().find((n) => n.dataset && n.dataset.key === m[1]) || null : null; };
  let obs; win.MutationObserver = function (cb) { obs = cb; this.observe = () => {}; };
  let resolve; win.fetch = () => new Promise((r) => { resolve = r; });
  B.install(win);
  const a = doc.createElement('button'); a.dataset.key = 'hi'; doc.body.appendChild(a);
  a.click();
  ok(a.dataset.busy === undefined, 'auto: a click with no request does not mark the button');
  let sent = 0; a.addEventListener('click', () => { if (!sent++) win.fetch('/api/x', { method: 'POST' }); }); // the stub runs a button's own listeners before the document's capture guard
  a.click();
  ok(a.dataset.busy === 'working', 'auto: a click that starts a POST marks the button at once');
  const { Ev } = require('./nav_dom.cjs');
  const tap = new Ev('click', { cancelable: true }); a.dispatchEvent(tap);
  ok(tap.stopped && tap.defaultPrevented, 'auto: a tap on a working button is swallowed (stopped and prevented)');
  const twin = doc.createElement('button'); twin.dataset.key = 'hi';
  doc.body.replaceChildren(twin); obs();
  ok(twin.dataset.busy === 'working', 'auto: a redrawn twin shows the working state');
  resolve({ ok: false, headers: { get: () => '' } }); await flush(); await flush();
  ok(twin.dataset.busy === 'failed', 'auto: an HTTP error ends failed');
  const g = doc.createElement('button'); g.dataset.key = 'get'; doc.body.appendChild(g);
  g.addEventListener('click', () => { win.fetch('/api/x'); });
  g.click(); ok(g.dataset.busy === undefined, 'auto: a GET never marks a button');
  console.log(JSON.stringify({ fails }));
})();
