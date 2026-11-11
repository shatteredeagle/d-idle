// Runs web/panels/map/panel.js against a stub DOM: the stitched world box, the nb= query the shell sends, the
// neighbour cache, hidden until tier 1. Usage: node map_panel.js path/to/panel.js
const assert = require('assert'), { load, shellCtx } = require('./panel_harness.js');
const P = load(process.argv[2]);
const rle = (w, h) => [1, w * h];
const cur = { areaId: 2, area: 'Blood Moor', version: 2, x0: 10, y0: 0, step: 1, w: 10, h: 3, rle: rle(10, 3), exits: [], waypoints: [] };
const nb = { areaId: 1, area: 'Rogue Encampment', version: 1, x0: 20, y0: 0, step: 1, w: 10, h: 3, rle: rle(10, 3), exits: [], waypoints: [] };
assert.strictEqual(JSON.stringify(P.union(cur, [{ map: nb, dx: 0, dy: 0 }])), JSON.stringify({ x0: 10, y0: 0, x1: 30, y1: 3 }), 'stitched world box spans both areas');
assert.strictEqual(P.union(cur, [{ map: nb, dx: -5, dy: 1 }]).x1, 25, 'offset moves the neighbour');
assert.strictEqual(P.nbParam({ 1: { version: 1 }, 3: { version: 7 } }), '1:1,3:7');
const { root, c } = shellCtx({}, {}, []);
const api = P.mount(root, c);
assert.strictEqual(c.hidden, true, 'hidden before the first reply');
assert.strictEqual(api.query(), '?v=0', 'first poll holds nothing');
api.update({ tiers_unlocked: [], version: 0 });
assert.strictEqual(c.hidden, true, 'hidden while tier 1 is closed');
api.update({ tiers_unlocked: [1], version: 2, map: cur, hero: { x: 18, y: 1 }, neighbours: [{ areaId: 1, version: 1, dx: 0, dy: 0, map: nb }] });
assert.strictEqual(c.hidden, false);
assert.strictEqual(c.meta.textContent, 'Blood Moor');
assert.strictEqual(api.query(), '?v=2&nb=1%3A1', 'second poll reports the cached neighbour version');
api.update({ tiers_unlocked: [1], version: 2, hero: { x: 19, y: 1 }, neighbours: [{ areaId: 1, version: 1, dx: 0, dy: 0 }] });
assert.strictEqual(api.query(), '?v=2&nb=1%3A1');
api.update({ tiers_unlocked: [1], version: 2, hero: { x: 19, y: 1 } });
assert.ok(!/nb=/.test(api.query()), 'a neighbour that disappeared is dropped from the cache: ' + api.query());
api.update(null, { status: 404 });
assert.strictEqual(c.hidden, true, 'a missing route hides the map');
// ---- trail and coverage layers ----
assert.strictEqual(P.trailWindow('30000'), 30000); assert.strictEqual(P.trailWindow('7'), 60000, 'unknown window falls back to 1 min'); assert.strictEqual(P.trailWindow(null), 60000);
const tr = [{ ago: 90000, x: 1, y: 1 }, { ago: 50000, x: 2, y: 1 }, { ago: 40000, x: 9, y: 9, e: true }, { ago: 10000, x: 10, y: 9 }, { ago: 500, x: 11, y: 9 }];
const plan = P.trailPlan(tr, 60000);
assert.strictEqual(plan.pts.length, 4, 'points older than the window are cut');
assert.strictEqual(JSON.stringify(plan.stalls.map((s) => [s.x, s.y, s.ms])), JSON.stringify([[9, 9, 30000], [10, 9, 9500]]), 'a gap over 3 s is a stall; a gap into a zone entry is not');
assert.ok(P.trailAlpha(0, 60000) > P.trailAlpha(30000, 60000) && P.trailAlpha(60000, 60000) >= 0.15, 'newest is brightest, oldest still visible');
assert.notStrictEqual(P.trailSig(plan, 60000, 2), P.trailSig(P.trailPlan(tr, 30000), 30000, 2), 'window change redraws');
assert.strictEqual(P.trailSig(plan, 60000, 2), P.trailSig(P.trailPlan(tr.slice(), 60000), 60000, 2), 'same trail, same signature: no redraw');
assert.strictEqual(P.coverSummary({ state: 'roaming', total: 118, legs: 17, cells: [[0, 0, 1], [1, 1, 1], [2, 2, 2], [3, 3, 0]] }), 'roam: 2/118 points seen, 1 failed, leg 17');
assert.strictEqual(P.coverSummary({ state: 'roaming', total: 1593, drawn: 800, seen: 119, failed: 0, legs: 126, cells: [[0, 0, 0]] }), 'roam: 119/1593 points seen, 0 failed, leg 126, drawing 800 of 1593');
assert.strictEqual(P.coverSummary(null), '');
// ---- activity share, events, windows ----
assert.strictEqual(P.trailWindow('1800000'), 1800000, 'the 30 min window is remembered');
{
  const hist = [{ ago: 20000, x: 1, y: 1, a: 0 }, { ago: 10000, x: 2, y: 1, a: 1 }, { ago: 8000, x: 3, y: 1, a: 1 }, { ago: 6000, x: 4, y: 1, a: 2 }, { ago: 0, x: 5, y: 1, a: 2 }];
  const names = ['walk', 'combat', 'town', 'farm', 'flee', 'rest', 'loot'];
  assert.strictEqual(P.activityShare(P.trailPlan(hist, 60000), names), 'stall 80% \u00b7 combat 20%', 'gaps over 3 s count as stall, the rest by activity');
  assert.strictEqual(P.activityShare(P.trailPlan(hist.map((p, i) => Object.assign({}, p, { ago: 4000 - i * 1000 })), 60000), names), 'combat 50% \u00b7 walk 25% \u00b7 town 25%');
}
assert.strictEqual(P.activityShare({ pts: [] }, []), '', 'no points, no line');
assert.strictEqual(P.eventShown({ k: 'kill' }, 'key'), true);
assert.strictEqual(P.eventShown({ k: 'potion' }, 'key'), false, 'potions only in all');
assert.strictEqual(P.eventShown({ k: 'potion' }, 'all'), true);
assert.strictEqual(P.eventShown({ k: 'kill' }, 'off'), false);
assert.strictEqual(P.eventShown({ k: 'loot', big: true }, 'key'), true);
assert.strictEqual(P.eventText({ k: 'kill', t: 'Fallen Shaman killed', f: { sec: 2.1, life: 92, min: 88, end: 'clear', kills: 1 } }), 'Fallen Shaman killed, 2.1 s fight, life 92% (min 88%), end=clear, 1 kill');
assert.strictEqual(P.eventText({ k: 'potion' }), 'potion');
{
  const store = { trailwin: '30000' }, timers = [], calls = [];
  const { root: r2, c: c2 } = shellCtx({}, store, timers);
  c2.store = (k) => ({ get: () => (k in store ? store[k] : null), set: (v) => { store[k] = v; } });
  const a2 = P.mount(r2, c2);
  const btn = (key) => c2.actions.children.find((b) => b.dataset.key === key);
  assert.ok(btn('events') && btn('coverage') && btn('trail-30000') && btn('trail-60000') && btn('trail-300000'), 'selector buttons are there');
  a2.update({ tiers_unlocked: [1], version: 2, map: cur, hero: { x: 18, y: 1 }, trail_area: 2, ts: 1, trail: tr,
    coverage: { area: 2, cell: 8, cover: 5, total: 2, state: 'roaming', legs: 3, target: [12, 1], cells: [[12, 1, 0], [14, 1, 1]] } });
  assert.strictEqual(btn('coverage').attrs['aria-pressed'], 'false', 'coverage is off by default');
  btn('coverage').fire('click');
  assert.strictEqual(store.coverage, '1', 'the toggle is remembered');
  assert.ok(/roam: 1\/2 points seen/.test(c2.body.children[2].textContent), c2.body.children[2].textContent);
  btn('events').fire('click');
  assert.strictEqual(store.events, 'all', 'the events filter cycles key, all, off and is remembered');
  assert.strictEqual(btn('events').textContent, 'Events: all');
  a2.update({ tiers_unlocked: [1], version: 2, hero: { x: 18, y: 1 }, trail_area: 2, ts: 1, trail_legend: ['walk', 'combat'], trail: [{ ago: 5000, x: 9, y: 1 }, { ago: 3000, x: 10, y: 1, a: 1 }, { ago: 0, x: 11, y: 1, a: 1 }], events: [{ ago: 1000, x: 10, y: 1, k: 'kill', t: 'Fallen killed' }] });
  assert.ok(/combat/.test(c2.body.children[4].textContent), 'the share line names combat: ' + c2.body.children[4].textContent);
  btn('trail-300000').fire('click');
  assert.strictEqual(store.trailwin, '300000', 'the window is remembered');
  assert.strictEqual(btn('trail-300000').attrs['aria-pressed'], 'true');
  // the coverage of another area is not shown
  a2.update({ tiers_unlocked: [1], version: 2, hero: { x: 18, y: 1 }, coverage: { area: 9, cell: 8, total: 1, state: 'roaming', cells: [[1, 1, 0]] } });
  assert.ok(/not roaming/.test(c2.body.children[2].textContent));
  // storage that throws must not break the panel
  const bad = shellCtx({}, {}, []);
  bad.c.store = () => ({ get() { throw new Error('blocked'); }, set() { throw new Error('blocked'); } });
  const a3 = P.mount(bad.root, bad.c);
  a3.update({ tiers_unlocked: [1], version: 2, map: cur, hero: { x: 18, y: 1 }, trail: tr });
}
console.log('ok');
