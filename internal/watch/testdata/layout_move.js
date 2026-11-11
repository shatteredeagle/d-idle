// The editor's move (web/layout.js moveItem): a panel dropped on free cells lands exactly there, a panel dropped on
// others pushes them down and the push cascades, the input is never changed (so Escape has the old layout to go back
// to), and a drop back on the start is a no-op. Usage: node layout_move.js path/to/web/layout.js path/to/web/layout.json
const assert = require('assert'), fs = require('fs'), { load } = require('./panel_harness.js');
const L = load(process.argv[2]);
const eq = (a, b, msg) => assert.strictEqual(JSON.stringify(a), JSON.stringify(b), msg);
const P = (id, col, row, colSpan, rowSpan, extra) => Object.assign({ id, col, row, colSpan, rowSpan, collapsed: false }, extra || {});
const at = (items, id) => { const i = items.find((x) => x.id === id); return [i.col, i.row, i.colSpan, i.rowSpan]; };
function noOverlap(items) {
  const m = {};
  for (const it of items.filter((i) => !i.hidden)) {
    for (let r = it.row; r < it.row + (it.collapsed ? 1 : it.rowSpan); r++) for (let c = it.col; c < it.col + it.colSpan; c++) {
      assert.ok(!m[r + ':' + c], it.id + ' overlaps ' + m[r + ':' + c]);
      m[r + ':' + c] = it.id;
    }
  }
}

// ---- the minimap fit (user report): a 3x9 panel goes into a free 3x9 hole exactly, with no spare room around it ----
{
  // row 1-9: video cols 1-6, a hole cols 7-9, stats cols 10-12; the map sits below everything
  const base = [P('video', 1, 1, 6, 9), P('stats', 10, 1, 3, 9), P('now', 1, 10, 12, 4), P('map', 1, 14, 3, 9)];
  const snap = JSON.stringify(base);
  const got = L.moveItem(base, 'map', { col: 7, row: 1 }, 12);
  eq(at(got, 'map'), [7, 1, 3, 9], 'the map fills the hole exactly');
  eq([at(got, 'video'), at(got, 'stats'), at(got, 'now')], [[1, 1, 6, 9], [10, 1, 3, 9], [1, 10, 12, 4]], 'nothing else moved');
  noOverlap(got);
  assert.strictEqual(JSON.stringify(base), snap, 'the layout before the drag is untouched');
  // the cause of the report: hidden panels (locked features, routes that are off) held cells nobody could see.
  // A hidden panel in the hole must not block the drop, must not move, and is still there to be saved.
  const withHidden = base.concat([P('chronicle', 7, 1, 3, 9, { hidden: true }), P('alerts', 7, 3, 6, 5, { hidden: true })]);
  const got2 = L.moveItem(withHidden, 'map', { col: 7, row: 1 }, 12);
  eq(at(got2, 'map'), [7, 1, 3, 9], 'a hidden panel does not block the hole');
  eq([at(got2, 'chronicle'), at(got2, 'alerts')], [[7, 1, 3, 9], [7, 3, 6, 5]], 'hidden panels stay where the file has them');
  eq(L.toPanels(got2, {}).map((p) => p.id).sort(), ['alerts', 'chronicle', 'map', 'now', 'stats', 'video'], 'and are saved again');
  // every edge of the hole: one column left or right, or one row lower, must also be accepted without extra room
  for (const [c, r] of [[7, 1], [7, 2], [7, 5]]) noOverlap(L.moveItem(base, 'map', { col: c, row: r }, 12));
  // occ/free agree: the hole is free for exactly 3x9 and not for 4x9
  const m = L.occ(base.filter((p) => p.id !== 'map'));
  assert.ok(L.free(m, 7, 1, 3, 9)); assert.ok(!L.free(m, 7, 1, 4, 9)); assert.ok(!L.free(m, 6, 1, 3, 9)); assert.ok(!L.free(m, 7, 1, 3, 10));
}

// ---- push down and cascade ----
{
  const base = [P('a', 1, 1, 6, 3), P('b', 7, 1, 6, 3), P('c', 1, 4, 6, 2), P('d', 1, 6, 6, 2), P('e', 7, 4, 6, 2)];
  // b dragged onto a: a goes below b, c below a, d below c; e (other columns) stays
  const got = L.moveItem(base, 'b', { col: 1, row: 1 }, 12);
  eq(at(got, 'b'), [1, 1, 6, 3], 'the moved panel takes its cells');
  eq([at(got, 'a')[1], at(got, 'c')[1], at(got, 'd')[1]], [4, 7, 9], 'overlap pushes down and cascades');
  eq(at(got, 'e'), [7, 1, 6, 2], 'a panel in other columns only rises into the room that opened');
  noOverlap(got);
  // a partial overlap (two columns of a) pushes just the same
  const part = L.moveItem(base, 'e', { col: 5, row: 2, colSpan: 4 }, 12);
  noOverlap(part);
  assert.ok(at(part, 'a')[1] > 1 || at(part, 'e')[1] >= 4, 'a and e no longer share cells');
  // resize: growing a over b's columns pushes b down
  const grown = L.moveItem(base, 'a', { colSpan: 9 }, 12);
  eq(at(grown, 'a'), [1, 1, 9, 3]); eq(at(grown, 'b')[1], 4); noOverlap(grown);
  // growing down pushes the column under it, in order
  const tall = L.moveItem(base, 'a', { rowSpan: 6 }, 12);
  eq([at(tall, 'c')[1], at(tall, 'd')[1]], [7, 9]); noOverlap(tall);
  // spans are clamped to the grid
  eq(at(L.moveItem(base, 'a', { col: 11, colSpan: 6 }, 12), 'a').slice(0, 3), [7, 1, 6]);
  eq(at(L.moveItem(base, 'a', { col: -3, row: -2 }, 12), 'a').slice(0, 2), [1, 1]);
}

// ---- revert, drop, no-op ----
{
  const base = [P('a', 1, 1, 6, 3), P('b', 7, 1, 6, 3), P('c', 1, 4, 12, 2)];
  const snap = JSON.stringify(base);
  // a preview over other panels, then back to the start: the preview from the same base is the base again
  const over = L.moveItem(base, 'b', { col: 3, row: 1 }, 12);
  assert.ok(!L.sameCells(over, base), 'the preview differs');
  const back = L.moveItem(base, 'b', { col: 7, row: 1 }, 12);
  assert.ok(L.sameCells(back, base), 'moving back where nothing overlaps is the layout from before');
  assert.strictEqual(JSON.stringify(base), snap, 'previews never change the base (Escape and a bad drop go back to it)');
  // drop: the preview becomes the base of the next move
  const next = L.moveItem(over, 'c', { row: 1 }, 12);
  noOverlap(next); eq(at(next, 'c').slice(0, 2), [1, 1]);
  // a move to the cells the panel already has is a no-op; an unknown or hidden id changes nothing
  assert.ok(L.sameCells(L.moveItem(base, 'a', { col: 1, row: 1 }, 12), base));
  assert.ok(L.sameCells(L.moveItem(base, 'zz', { col: 1, row: 1 }, 12), base));
  assert.ok(L.sameCells(L.moveItem(base.concat([P('h', 1, 1, 2, 2, { hidden: true })]), 'h', { col: 5 }, 12), base.concat([P('h', 1, 1, 2, 2, { hidden: true })])));
  // dropped far below everything the panel rises to the first room in its columns (the page shows no dead band)
  eq(at(L.moveItem(base, 'a', { row: 40 }, 12), 'a').slice(0, 2), [1, 6]); // under c, which b keeps at row 4
  // collapsed panels count one row
  const col = L.moveItem(base, 'a', { collapsed: true }, 12);
  eq(at(col, 'c')[1], 4); // b still spans rows 1-3
  eq(at(L.moveItem([P('a', 1, 1, 12, 3), P('c', 1, 4, 12, 2)], 'a', { collapsed: true }, 12), 'c')[1], 2);
}

// ---- the shipped default: every panel can be moved to every column without an overlap or a lost panel ----
{
  const layout = JSON.parse(fs.readFileSync(process.argv[3], 'utf8'));
  const base = layout.breakpoints.wide.panels.map((p) => Object.assign({}, p));
  for (const p of base) {
    for (let c = 1; c + p.colSpan - 1 <= 12; c += 2) {
      const got = L.moveItem(base, p.id, { col: c, row: 5 }, 12);
      noOverlap(got);
      assert.strictEqual(got.length, base.length);
      assert.strictEqual(at(got, p.id)[0], c, p.id + ' keeps the column it was dropped in');
    }
  }
}
console.log('ok');
