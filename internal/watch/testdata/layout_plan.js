// Runs web/layout.js (the layout manifest logic of the shell) in node: breakpoints and fallback, editor ids mapped
// onto the real page, overlap push-down, packing of unnamed panels, the flow breakpoints, a foreign schema.
// Usage: node layout_plan.js path/to/web/layout.js fixture.json...
const assert = require('assert'), fs = require('fs'), { load } = require('./panel_harness.js');
const L = load(process.argv[2]);
// values from the vm context have their own Array prototype, so compare as JSON
const eq = (a, b, msg) => assert.strictEqual(JSON.stringify(a), JSON.stringify(b), msg);
const conf = { aliases: { 'hero-status': 'hero-vitals', 'hero-detail': 'hero-vitals', whyline: 'why', 'hero-now': 'now', voice: 'diary' }, compat: [], reserved: ['risk'] };
const page = ['video', 'activity', 'hero-vitals', 'map', 'gear', 'why', 'tally', 'currency', 'thoughts', 'now', 'orders-map', 'queue', 'rules', 'spend', 'chronicle',
  'diary', 'quests', 'cards', 'locked', 'bestiary', 'atlas', 'alerts', 'journal'].map((id) => ({ id, size: 's' }));

eq([0, 639, 640, 1023, 1024, 1439, 1440, 3000].map((w) => L.breakpointFor(w, '')), ['phone', 'phone', 'tablet', 'tablet', 'laptop', 'laptop', 'wide', 'wide']);
assert.strictEqual(L.breakpointFor(1920, 'overlay'), 'overlay');

function noOverlap(p) {
  const m = {};
  for (const it of p.items) {
    const h = it.collapsed ? 1 : it.rowSpan;
    assert.ok(it.col >= 1 && it.col + it.colSpan - 1 <= p.cols, it.id + ' inside the columns');
    for (let r = it.row; r < it.row + h; r++) for (let c = it.col; c < it.col + it.colSpan; c++) {
      assert.ok(!m[r + ':' + c], it.id + ' overlaps ' + m[r + ':' + c]);
      m[r + ':' + c] = it.id;
    }
  }
}

// every file given (the editor's raw exports and the shipped layouts) plans at every width with every page panel
// placed exactly once and nothing overlapping
for (const f of process.argv.slice(3)) {
  const layout = JSON.parse(fs.readFileSync(f, 'utf8'));
  for (const w of [390, 768, 1280, 1920]) {
    const p = L.plan(layout, w, '', conf, page);
    assert.ok(p, f + ' at ' + w);
    eq(p.items.map((i) => i.id).sort(), page.map((t) => t.id).sort(), f + ' places every panel once at ' + w);
    assert.strictEqual(p.flow, w < 1024);
    if (!p.flow) noOverlap(p);
    assert.strictEqual(new Set(p.items.map((i) => i.order)).size, p.items.length, 'orders are unique');
  }
}

// the mockup export: editor ids land on the real panels at the editor's positions; reserved ids are skipped silently
{
  const layout = JSON.parse(fs.readFileSync(process.argv[3], 'utf8'));
  const p = L.plan(layout, 1920, '', conf, page), at = (id) => p.items.find((i) => i.id === id);
  eq([at('hero-vitals').col, at('hero-vitals').row, at('hero-vitals').colSpan, at('hero-vitals').rowSpan], [1, 1, 4, 3]);
  eq([at('why').col, at('why').colSpan], [5, 5]);
  eq([at('now').col, at('now').row, at('now').colSpan], [1, 13, 6]);
  eq([at('video').col, at('video').row, at('video').colSpan, at('video').rowSpan], [1, 4, 6, 9]);
  assert.strictEqual(at('diary').placed, true);
  eq(p.unknown, [], 'risk is reserved, hero-detail is an alias of hero-vitals, nothing is unknown');
  assert.strictEqual(p.rowHeight, 48);
  // the editor sees the cells of the normal page
  const e = L.plan(layout, 1920, '', conf, page, true), key = (i) => [i.id, i.col, i.row, i.colSpan, i.rowSpan].join();
  eq(e.items.map(key), p.items.map(key));
}

// the merged Hero tile: the old Hero stats id resolves to hero-vitals; a layout naming both shows one tile at the first
// position, and a hidden flag on the alias alone switches the merged tile off
{
  const one = (id, extra) => Object.assign({ id, band: 'side', col: 1, row: 1, colSpan: 4, rowSpan: 4, collapsed: false, order: 0 }, extra);
  const mk = (...ps) => ({ schema: 'watch.layout/1', cols: 12, rowHeight: 48, breakpoints: { wide: { width: 1920, cols: 12, panels: ps } } });
  const at = (p, id) => p.items.filter((i) => i.id === id);
  let p = L.plan(mk(one('hero-vitals'), one('hero-detail', { col: 9, row: 9, order: 1 })), 1920, '', conf, page);
  eq(at(p, 'hero-vitals').map((i) => [i.col, i.row]), [[1, 1]]); eq(p.unknown, []);
  p = L.plan(mk(one('hero-detail', { col: 5, row: 2 }), one('hero-vitals', { order: 1 })), 1920, '', conf, page);
  eq(at(p, 'hero-vitals').map((i) => [i.col, i.row]), [[5, 1]], 'the first entry wins (it rises into the free room)');
  p = L.plan(mk(one('hero-detail', { hidden: true })), 1920, '', conf, page);
  eq(p.off, ['hero-vitals'], 'a hidden alias hides the merged tile');
  assert.strictEqual(at(p, 'hero-vitals').length, 0);
}

// unknown ids are reported once and ignored; a missing breakpoint falls back to the next wider; none: null
{
  const one = (id) => ({ id, band: 'grid', col: 1, row: 1, colSpan: 6, rowSpan: 2, collapsed: false, order: 0 });
  const layout = { schema: 'watch.layout/1', cols: 12, rowHeight: 48, breakpoints: { wide: { width: 1920, cols: 12, panels: [one('video'), Object.assign(one('zz-gone'), { order: 1 })] } } };
  const p = L.plan(layout, 1280, '', conf, page);
  assert.strictEqual(p.name, 'wide'); assert.strictEqual(p.bp, 'laptop');
  eq(p.unknown, ['zz-gone']);
  assert.strictEqual(L.plan({ schema: 'watch.layout/1', cols: 12, breakpoints: { phone: layout.breakpoints.wide } }, 1920, '', conf, page), null, 'a narrower breakpoint never stands in for a wider one');
  assert.strictEqual(L.plan({ schema: 'watch.layout/2', breakpoints: layout.breakpoints }, 1920, '', conf, page), null, 'another schema is refused');
  assert.strictEqual(L.plan(null, 1920, '', conf, page), null);
  assert.strictEqual(L.plan(layout, 1920, 'overlay', conf, page), null, 'overlay has its own breakpoint');
  // a collapsed panel takes one row
  layout.breakpoints.wide.panels = [Object.assign(one('video'), { collapsed: true, rowSpan: 9 }), Object.assign(one('map'), { row: 2, order: 1 })];
  const q = L.plan(layout, 1920, '', conf, page);
  assert.strictEqual(q.items.find((i) => i.id === 'map').row, 2);
  // toPanels round trip: what the editor saves plans back to the same cells
  const saved = { schema: 'watch.layout/1', cols: 12, rowHeight: 48, breakpoints: { wide: { width: 1920, cols: 12, panels: L.toPanels(q.items, {}) } } };
  const again = L.plan(saved, 1920, '', conf, page), key = (i) => [i.id, i.col, i.row, i.colSpan, i.rowSpan, i.collapsed].join();
  eq(again.items.map(key).sort(), q.items.map(key).sort());
}
// a hidden panel takes no cell and an empty band closes up; the editor keeps both
{
  const e = (id, col, row, w, h, order) => ({ id, band: 'grid', col, row, colSpan: w, rowSpan: h, collapsed: false, order });
  const layout = { schema: 'watch.layout/1', cols: 12, rowHeight: 48, breakpoints: { wide: { width: 1920, cols: 12, panels: [e('video', 1, 1, 12, 2, 0), e('map', 1, 3, 12, 3, 1), e('gear', 1, 6, 6, 2, 2)] } } };
  const pg = [{ id: 'video', size: 'm' }, { id: 'map', size: 's', hidden: true }, { id: 'gear', size: 's' }];
  const p = L.plan(layout, 1920, '', conf, pg);
  eq(p.items.map((i) => [i.id, i.row]), [['video', 1], ['gear', 3]]);
  eq(p.unknown, []);
  // the editor: the same cells; the hidden panel comes along at its file position (it is saved again) and reserves nothing
  const q = L.plan(layout, 1920, '', conf, pg, true);
  eq(q.items.map((i) => [i.id, i.row, !!i.hidden]), [['video', 1, false], ['gear', 3, false], ['map', 3, true]]);
  eq(L.toPanels(q.items, {}).map((x) => x.id).sort(), ['gear', 'map', 'video']);
}
console.log('ok');
