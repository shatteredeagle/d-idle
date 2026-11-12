// Fit content (web/layout.js fitRows, contentHeight, plan with measured rows): a panel's rows come from what it shows.
// The regression: a video panel 600 px wide with a stored span of 9 (or 14) rows must be as tall as its picture
// (width * 0.75) plus its head, not as tall as the file says. Usage: node layout_fit.js path/to/web/layout.js
const assert = require('assert'), { load } = require('./panel_harness.js');
const L = load(process.argv[2]);
const eq = (a, b, msg) => assert.strictEqual(JSON.stringify(a), JSON.stringify(b), msg);
const ROWH = 48, GAP = 12, PAD = 12;

// a card as the browser lays it out: head, then a body whose frame is 4:3 of the inner width
function card(width, top, kids) {
  let y = top + PAD;
  const children = kids.map((k) => { const r = { top: y, bottom: y + k.h }; y += k.h + 8; return { hidden: !!k.hidden, getClientRects: () => (k.gone ? [] : [r]), getBoundingClientRect: () => r }; });
  return { children, getBoundingClientRect: () => ({ top, width }) };
}
const cs = { paddingBottom: PAD + 'px', borderBottomWidth: '1px' };
const videoCard = (width) => card(width, 100, [{ h: 34 }, { h: (width - 2 * PAD) * 0.75 }]);

// ---- the height of a card is its content's ----
{
  const h = L.contentHeight(videoCard(600), cs);
  eq(h, PAD + 34 + 8 + 432 + PAD + 1);
  assert.ok(Math.abs(h - (600 * 0.75 + 34)) < 40, 'about width * 0.75 plus the head: ' + h);
  // hidden and unrendered children do not count; an empty card has no height to fit
  eq(L.contentHeight(card(600, 0, [{ h: 34 }, { h: 300, hidden: true }, { h: 300, gone: true }]), cs), PAD + 34 + PAD + 1);
  eq(L.contentHeight(card(600, 0, []), cs), 0);
}

// ---- rows from a height: the smallest span that holds it ----
{
  eq([1, 48, 49, 108, 109, 168].map((h) => L.fitRows(h, ROWH, GAP)), [1, 1, 2, 2, 3, 3]);
  eq(L.fitRows(48.4, ROWH, GAP), 1, 'a rounding half-pixel does not cost a row');
  const rows = L.fitRows(L.contentHeight(videoCard(600), cs), ROWH, GAP), cell = rows * ROWH + (rows - 1) * GAP;
  const slack = cell - L.contentHeight(videoCard(600), cs);
  assert.ok(slack >= 0 && slack < ROWH + GAP, 'the cell is taller than the picture and its head by less than one row: ' + slack);
}

// ---- the plan uses the measured rows, not the stored span; what was under the dead area rises ----
{
  const e = (id, col, row, w, h, order) => ({ id, band: 'grid', col, row, colSpan: w, rowSpan: h, collapsed: false, order });
  const layout = { schema: 'watch.layout/1', cols: 12, rowHeight: ROWH, breakpoints: { wide: { width: 1920, cols: 12, panels: [e('video', 1, 1, 6, 14, 0), e('map', 7, 1, 6, 14, 1), e('now', 1, 15, 6, 4, 2), e('journal', 7, 15, 6, 4, 3)] } } };
  const rows = L.fitRows(L.contentHeight(videoCard(600), cs), ROWH, GAP);
  assert.ok(rows < 14);
  const page = (r) => [{ id: 'video', size: 'm', rows: r }, { id: 'map', size: 's' }, { id: 'now', size: 'm' }, { id: 'journal', size: 'm' }];
  const at = (p, id) => { const i = p.items.find((x) => x.id === id); return [i.row, i.rowSpan]; };
  // before the first measure the stored span stands (an old shell, or the first pass)
  eq(at(L.plan(layout, 1920, '', {}, page(0)), 'video'), [1, 14]);
  const p = L.plan(layout, 1920, '', {}, page(rows));
  eq(at(p, 'video'), [1, rows], 'the stored span of 14 is not used');
  eq(at(p, 'now'), [rows + 1, 4], 'the card under the picture rises to it');
  eq(at(p, 'map'), [1, 14], 'a panel without a measure keeps its span');
  eq(at(p, 'journal'), [15, 4]);
  // a panel that needs more than the file gives pushes down instead of overlapping
  const q = L.plan(layout, 1920, '', {}, page(20));
  eq(at(q, 'video'), [1, 20]); eq(at(q, 'now')[0], 21);
  // the file is not changed and still names the stored span: old loaders and the editor export are unaffected
  eq(layout.breakpoints.wide.panels[0].rowSpan, 14);
  eq(Object.keys(L.toPanels(p.items, {})[0]).sort(), ['band', 'col', 'colSpan', 'collapsed', 'id', 'order', 'row', 'rowSpan']);
  // a panel the file does not name is packed with its measured rows
  const r = L.plan(layout, 1920, '', {}, page(rows).concat([{ id: 'gear', size: 's', rows: 3 }]));
  eq(at(r, 'gear')[1], 3);
  // the flow breakpoints have no fixed rows: a measure changes nothing there
  assert.strictEqual(L.plan({ schema: 'watch.layout/1', cols: 12, breakpoints: { phone: layout.breakpoints.wide } }, 390, '', {}, page(rows)).flow, true);
}
console.log('ok');
