// Runs web/senses.js against a stub DOM: the stitched world box, the nb= cache parameter and what the poll sends/keeps.
// Usage: node senses_logic.js path/to/senses.js
const fs = require('fs'), vm = require('vm'), assert = require('assert');
const els = {}, urls = [], calls = [];
const stubCtx = new Proxy({}, { get: (t, k) => (k in t ? t[k] : () => ({ width: 1 })), set: (t, k, v) => { t[k] = v; return true; } });
function mk(id) {
  return { id, hidden: true, textContent: '', width: 320, height: 320, classList: { toggle: () => false }, setAttribute() {}, addEventListener() {},
    getContext: () => stubCtx, getBoundingClientRect: () => ({ width: 320 }), setPointerCapture() {} };
}
const document = { getElementById: id => els[id] || (els[id] = mk(id)), documentElement: {} };
const rle = (w, h) => [1, w * h];
const cur = { areaId: 2, area: 'Blood Moor', version: 2, x0: 10, y0: 0, step: 1, w: 10, h: 3, rle: rle(10, 3), exits: [], waypoints: [] };
const nb = { areaId: 1, area: 'Rogue Encampment', version: 1, x0: 20, y0: 0, step: 1, w: 10, h: 3, rle: rle(10, 3), exits: [], waypoints: [] };
const replies = [
  { tiers_unlocked: [1], version: 2, map: cur, hero: { x: 18, y: 1 }, neighbours: [{ areaId: 1, version: 1, dx: 0, dy: 0, map: nb }] },
  { tiers_unlocked: [1], version: 2, hero: { x: 19, y: 1 }, neighbours: [{ areaId: 1, version: 1, dx: 0, dy: 0 }] },
  { tiers_unlocked: [1], version: 2, hero: { x: 19, y: 1 } },
  { tiers_unlocked: [1], version: 2, hero: { x: 19, y: 1 } },
];
let n = 0;
const ctx = {
  document, getComputedStyle: () => ({ getPropertyValue: () => '' }), console, encodeURIComponent, Uint8Array, Object, Math,
  fetch: u => { urls.push(u); return Promise.resolve({ status: 200, json: () => Promise.resolve(replies[Math.min(n++, replies.length - 1)]) }); },
  setTimeout: (f, ms) => { calls.push(ms); if (urls.length < 4) setImmediate(f); return 1; },
};
ctx.window = ctx;
vm.runInNewContext(fs.readFileSync(process.argv[2], 'utf8'), ctx);
setTimeout(() => {
  const P = ctx.sensesPanel;
  const u = P.union(cur, [{ map: nb, dx: 0, dy: 0 }]);
  assert.strictEqual(JSON.stringify(u), JSON.stringify({ x0: 10, y0: 0, x1: 30, y1: 3 }), 'stitched world box spans both areas');
  assert.strictEqual(P.union(cur, [{ map: nb, dx: -5, dy: 1 }]).x1, 25, 'offset moves the neighbour');
  assert.strictEqual(P.nbParam({ 1: { version: 1 }, 3: { version: 7 } }), '1:1,3:7');
  assert.strictEqual(urls[0], '/api/senses?v=0', 'first poll holds nothing');
  assert.strictEqual(urls[1], '/api/senses?v=2&nb=1%3A1', 'second poll reports the cached neighbour version');
  assert.ok(!/nb=/.test(urls[3]), 'a neighbour that disappeared is dropped from the cache: ' + urls[3]);
  assert.strictEqual(els.sensespanel.hidden, false);
  console.log('ok');
}, 200);
