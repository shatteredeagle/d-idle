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
console.log('ok');
