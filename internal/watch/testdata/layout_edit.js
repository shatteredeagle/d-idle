// Drives web/layoutedit.js in a stub DOM: a pointer drag previews live and pushes panels down, a drop commits, Escape
// and a release outside the grid put everything back, the keyboard previews until Enter, and only Save reaches the
// stored layout. Usage: node layout_edit.js path/to/web/layout.js path/to/web/layoutedit.js
const assert = require('assert'), fs = require('fs'), vm = require('vm'), { load } = require('./panel_harness.js');
const L = load(process.argv[2]);
const eq = (a, b, msg) => assert.strictEqual(JSON.stringify(a), JSON.stringify(b), msg);

function mk(tag, cls) {
  const ls = {}, attrs = {};
  const e = { tag, className: cls || '', dataset: {}, style: {}, children: [], parent: null, hidden: false, tabIndex: -1, textContent: '', focused: 0,
    setAttribute(k, v) { attrs[k] = String(v); }, getAttribute(k) { return k in attrs ? attrs[k] : null; }, removeAttribute(k) { delete attrs[k]; },
    addEventListener(t, f) { (ls[t] = ls[t] || []).push(f); }, removeEventListener(t, f) { ls[t] = (ls[t] || []).filter((x) => x !== f); },
    fire(t, ev) { const o = Object.assign({ button: 0, pointerId: 1, clientX: 0, clientY: 0, key: '', shiftKey: false, target: e, currentTarget: e, prevented: false, preventDefault() { o.prevented = true; }, stopPropagation() {} }, ev); (ls[t] || []).slice().forEach((f) => f(o)); return o; },
    listeners: (t) => (ls[t] || []).length,
    append(...k) { k.forEach((c) => { c.parent = e; e.children.push(c); }); }, appendChild(c) { e.append(c); return c; },
    remove() { if (e.parent) e.parent.children = e.parent.children.filter((c) => c !== e); },
    insertBefore(c, ref) { c.parent = e; e.children.splice(e.children.indexOf(ref), 0, c); },
    querySelector(sel) { const want = sel.replace(/^.*\./, ''); const walk = (n) => { for (const c of n.children) { if (c.className === want) return c; const r = walk(c); if (r) return r; } return null; }; return walk(e); },
    get firstChild() { return e.children[0] || null; }, removeChild(c) { c.remove(); },
    focus() { e.focused++; }, setPointerCapture() { e.captured = true; }, releasePointerCapture() { e.captured = false; },
    // one cell is 100 px wide and 60 px high with no gap, the grid starts at 0,0
    getBoundingClientRect() { const c = parseInt(e.style.gridColumn, 10) || 1, r = parseInt(e.style.gridRow, 10) || 1; return e.isMain ? { left: 0, top: 0, right: 1200, bottom: 6000, width: 1200 } : { left: (c - 1) * 100, top: (r - 1) * 60 }; },
  };
  Object.defineProperty(e, 'parentElement', { get: () => e.parent });
  return e;
}
const P = (id, col, row, colSpan, rowSpan) => ({ id, band: 'grid', col, row, colSpan, rowSpan, collapsed: false, order: 0 });

function setup(opts) {
  const shell = mk('div'), main = mk('div');
  main.isMain = true; shell.append(main);
  const panels = [P('video', 1, 1, 6, 4), P('map', 7, 1, 3, 4), P('stats', 10, 1, 3, 4), P('now', 1, 5, 6, 3), P('queue', 7, 5, 6, 3), P('locked', 1, 8, 6, 2)];
  panels.forEach((p, i) => { p.order = i; });
  const layout = { schema: 'watch.layout/1', cols: 12, rowHeight: 48, breakpoints: { wide: { width: 1920, cols: 12, panels } } };
  const targets = new Map(), page = [];
  for (const p of panels) { const n = mk('section'); n.id = p.id; if (opts && opts.aspect && opts.aspect[p.id]) n.dataset.aspect = opts.aspect[p.id]; main.append(n); targets.set(p.id, n); page.push({ id: p.id, size: 's', hidden: p.id === 'locked' }); }
  const plan = L.plan(layout, 1920, '', {}, page, true);
  const calls = { save: [], reset: 0, done: [], toast: [], raf: 0 };
  const api = { el: (t, c, text) => { const e = mk(t, c); if (text !== undefined) e.textContent = text; return e; }, toast: (s, t) => calls.toast.push(s + ': ' + t), main, targets, layout,
    demo: (opts && opts.demo) || '', plan: () => plan, save: async (l) => { calls.save.push(JSON.parse(JSON.stringify(l))); return { ok: true, layout: l }; },
    reset: async () => { calls.reset++; return { ok: true, layout }; }, done: (l) => calls.done.push(l) };
  const src = fs.readFileSync(process.argv[3], 'utf8').replace(/^import .*$/m, '').replace('export function start', 'function start');
  const ctx = { moveItem: L.moveItem, sameCells: L.sameCells, toPanels: L.toPanels, setOff: L.setOff, JSON, Math, Object, Map, parseFloat, parseInt, String,
    getComputedStyle: () => ({ columnGap: '0px', paddingLeft: '0px' }), requestAnimationFrame: (f) => { calls.raf++; f(); }, window: { innerWidth: 1920 },
    matchMedia: () => ({ matches: !!(opts && opts.still) }) };
  vm.runInNewContext(src + '\n;this.start = start;', ctx);
  const ed = ctx.start(api);
  const at = (id) => { const n = targets.get(id); return [parseInt(n.style.gridColumn, 10), parseInt(n.style.gridRow, 10), +n.style.gridColumn.split('span ')[1], +n.style.gridRow.split('span ')[1]]; };
  const frame = (id) => main.children.find((c) => c.className === 'd-editframe' && c.dataset.for === id);
  const handle = (id) => frame(id).children.find((c) => c.className === 'd-edithandle'), corner = (id) => frame(id).children.find((c) => c.className === 'd-editrs');
  const btn = (text) => shell.children[0].children.find((c) => c.textContent === text);
  return { ed, api, main, targets, layout, calls, at, handle, corner, btn, shell, frame };
}
const start = { video: [1, 1, 6, 4], map: [7, 1, 3, 4], stats: [10, 1, 3, 4], now: [1, 5, 6, 3], queue: [7, 5, 6, 3] };
const all = (t) => Object.fromEntries(Object.keys(start).map((id) => [id, t.at(id)]));

// ---- the page at the start: the cells of the plan, hidden panels have no handle and no cell ----
{
  const t = setup();
  eq(all(t), start);
  assert.strictEqual(t.targets.get('locked').dataset.editcell, undefined, 'a hidden panel is not editable');
  assert.strictEqual(t.main.dataset.editing, '1');
}

// ---- drag: live preview, push down with a cascade, commit on drop; the layout object changes only then ----
{
  const t = setup();
  const before = JSON.stringify(t.layout);
  // grab the map handle at its top-left cell (col 7,row 1) and move 6 columns left: onto the video
  t.handle('map').fire('pointerdown', { clientX: 650, clientY: 30 });
  t.handle('map').fire('pointermove', { clientX: 50, clientY: 30 });
  eq(t.at('map'), [1, 1, 3, 4], 'the dragged panel follows the pointer cell');
  eq([t.at('video')[1], t.at('now')[1]], [5, 9], 'the video is pushed under the map and pushes Hero now in turn');
  assert.strictEqual(t.main.dataset.preview, 'map');
  assert.ok(t.calls.raf > 0 && t.targets.get('video').dataset.flip === 'go', 'pushed panels slide');
  assert.strictEqual(t.targets.get('map').dataset.flip, undefined, 'the dragged one does not');
  assert.strictEqual(JSON.stringify(t.layout), before, 'a preview does not touch the layout');
  // back to the start cell: everything is where it was
  t.handle('map').fire('pointermove', { clientX: 650, clientY: 30 });
  eq(all(t), start, 'moving back reverts the push');
  assert.strictEqual(t.main.dataset.preview, undefined);
  // over the video again and release on the grid: committed
  t.handle('map').fire('pointermove', { clientX: 50, clientY: 30 });
  t.handle('map').fire('pointerup', { clientX: 50, clientY: 30 });
  eq(t.at('map'), [1, 1, 3, 4]); eq(t.at('video')[1], 5);
  assert.strictEqual(t.main.dataset.preview, undefined);
  assert.strictEqual(t.handle('map').listeners('pointermove'), 0, 'the drag listeners are gone');
  assert.strictEqual(t.calls.save.length, 0, 'a drop is not a Save');
  assert.strictEqual(JSON.stringify(t.layout), before, 'the shell layout is untouched until Save');
}

// ---- the minimap into a free hole next to it: exact fit, nothing else moves ----
{
  const t = setup();
  // free the cells of stats first by dragging it to the bottom, then drop the map exactly there (cols 10-12, rows 1-4)
  t.handle('stats').fire('pointerdown', { clientX: 950, clientY: 30 });
  t.handle('stats').fire('pointermove', { clientX: 950, clientY: 30 + 60 * 9 });
  t.handle('stats').fire('pointerup', { clientX: 950, clientY: 30 + 60 * 9 });
  const others = [t.at('video'), t.at('now'), t.at('queue'), t.at('stats')];
  t.handle('map').fire('pointerdown', { clientX: 650, clientY: 30 });
  t.handle('map').fire('pointermove', { clientX: 950, clientY: 30 });
  t.handle('map').fire('pointerup', { clientX: 950, clientY: 30 });
  eq(t.at('map'), [10, 1, 3, 4], 'the map sits exactly in the free cells');
  eq([t.at('video'), t.at('now'), t.at('queue'), t.at('stats')], others, 'nothing else moved');
}

// ---- Escape, a release outside the grid and pointercancel put everything back ----
for (const how of ['escape', 'outside', 'cancel']) {
  const t = setup();
  t.handle('map').fire('pointerdown', { clientX: 650, clientY: 30 });
  t.handle('map').fire('pointermove', { clientX: 50, clientY: 30 });
  assert.notDeepStrictEqual(all(t), start);
  if (how === 'escape') assert.ok(t.targets.get('map').fire('keydown', { key: 'Escape' }).prevented);
  if (how === 'outside') t.handle('map').fire('pointerup', { clientX: -40, clientY: 30 });
  if (how === 'cancel') t.handle('map').fire('pointercancel', {});
  eq(all(t), start, how + ' reverts');
  assert.strictEqual(t.main.dataset.preview, undefined, how);
  assert.strictEqual(t.targets.get('map').dataset.drag, undefined, how);
  assert.strictEqual(t.handle('map').listeners('pointermove'), 0, how);
  // and the editor still works afterwards
  t.handle('map').fire('pointerdown', { clientX: 650, clientY: 30 });
  t.handle('map').fire('pointermove', { clientX: 50, clientY: 30 });
  t.handle('map').fire('pointerup', { clientX: 50, clientY: 30 });
  eq(t.at('map'), [1, 1, 3, 4], how + ': a later drag commits');
}

// ---- resize by the corner pushes too ----
{
  const t = setup();
  t.corner('map').fire('pointerdown', { clientX: 890, clientY: 230 });
  t.corner('map').fire('pointermove', { clientX: 1090, clientY: 230 }); // two columns wider: over stats
  eq(t.at('map'), [7, 1, 5, 4]); eq(t.at('stats')[1], 5, 'stats is pushed down');
  t.targets.get('map').fire('keydown', { key: 'Escape' });
  eq(all(t), start);
}

// ---- keyboard: arrows preview, Enter commits, Escape or leaving the panel drops; Save posts the committed layout ----
{
  const t = setup();
  const map = t.targets.get('map');
  for (let i = 0; i < 6; i++) assert.ok(map.fire('keydown', { key: 'ArrowLeft' }).prevented);
  eq(t.at('map'), [1, 1, 3, 4]); eq(t.at('video')[1], 5);
  assert.strictEqual(t.main.dataset.preview, 'map');
  map.fire('keydown', { key: 'Escape' });
  eq(all(t), start, 'Escape drops the keyboard preview');
  map.fire('keydown', { key: 'ArrowLeft' });
  map.fire('blur', {});
  eq(all(t), start, 'leaving the panel drops the preview');
  map.fire('keydown', { key: 'ArrowRight' }); map.fire('keydown', { key: 'ArrowRight', shiftKey: true });
  map.fire('keydown', { key: 'Enter' });
  eq(t.at('map'), [8, 1, 4, 4], 'Enter commits the move and the resize');
  assert.strictEqual(t.main.dataset.preview, undefined);
  map.fire('blur', {});
  eq(t.at('map'), [8, 1, 4, 4], 'a committed move survives leaving the panel');
  // typing inside a panel is not a move
  map.fire('keydown', { key: 'ArrowLeft', target: {} });
  eq(t.at('map'), [8, 1, 4, 4]);
  // C collapses at once
  map.fire('keydown', { key: 'c' });
  assert.strictEqual(map.dataset.collapsed, '1');
  (async () => {
    await t.btn('Save').fire('click', {});
    await null; await null;
    assert.strictEqual(t.calls.save.length, 1);
    const saved = t.calls.save[0];
    assert.strictEqual(saved.schema, 'watch.layout/1');
    const ps = saved.breakpoints.wide.panels, m = ps.find((p) => p.id === 'map');
    eq([m.col, m.row, m.colSpan, m.rowSpan, m.collapsed], [8, 1, 4, 4, true]);
    eq(Object.keys(m).sort(), ['band', 'col', 'colSpan', 'collapsed', 'id', 'order', 'row', 'rowSpan'], 'the export format is unchanged');
    assert.ok(ps.find((p) => p.id === 'locked'), 'the hidden panel is saved again');
    eq(ps.map((p) => p.order), ps.map((_, i) => i));
    // Done removes every trace of the mode
    t.btn('Done').fire('click', {});
    assert.strictEqual(t.main.dataset.editing, undefined);
    assert.strictEqual(map.dataset.editcell, undefined);
    assert.ok(!t.main.children.some((c) => c.className === 'd-editframe'), 'the frames are gone');
    assert.strictEqual(map.children.length, 0, 'nothing was ever put inside the panel');
    assert.strictEqual(t.calls.done.length, 1);
    console.log('ok');
  })().catch((e) => { console.error(e); process.exit(1); });
}

// ---- every card is editable, also one that redraws itself (Hero now and Up next wiped the old in-panel handle) ----
{
  const t = setup();
  for (const id of Object.keys(start)) {
    assert.ok(t.frame(id) && t.handle(id) && t.corner(id), id + ' has a frame with a handle bar and a corner');
    assert.strictEqual(t.targets.get(id).children.length, 0, id + ': nothing is injected into the panel');
    assert.strictEqual(t.targets.get(id).dataset.editcell, '1');
  }
  const now = t.targets.get('now');
  now.children = []; now.append({ className: 'card-head', children: [] }); // app.js renderNow: clear and rebuild
  t.handle('now').fire('pointerdown', { clientX: 50, clientY: 270 });
  t.handle('now').fire('pointermove', { clientX: 650, clientY: 270 });
  t.handle('now').fire('pointerup', { clientX: 650, clientY: 270 });
  eq(t.at('now').slice(0, 2), [7, 5], 'a card that redrew itself is still dragged');
  eq(t.at('queue')[1], 8, 'and pushes like any panel');
  t.corner('now').fire('pointerdown', { clientX: 1190, clientY: 470 });
  t.corner('now').fire('pointermove', { clientX: 1190, clientY: 530 });
  t.corner('now').fire('pointerup', { clientX: 1190, clientY: 530 });
  eq(t.at('now')[3], 4, 'and resized');
  // the frame sits on the same cells as its panel
  eq([t.frame('now').style.gridColumn, t.frame('now').style.gridRow], [now.style.gridColumn, now.style.gridRow]);
}

// ---- hide and show: a hidden panel leaves the grid, the others rise, the bar brings it back, Save writes hidden: true ----
{
  const t = setup();
  t.handle('video').children.find((c) => c.textContent === 'Hide').fire('click', {});
  assert.strictEqual(t.targets.get('video').dataset.layoutOff, '1');
  assert.strictEqual(t.frame('video').hidden, true);
  eq(t.at('now')[1], 1, 'Hero now rises into the room');
  const show = t.shell.children[0].querySelector('off').children.find((c) => c.dataset && c.dataset.show === 'video');
  assert.ok(show, 'the bar lists the hidden panel');
  t.targets.get('map').fire('keydown', { key: 'h' });
  assert.strictEqual(t.targets.get('map').dataset.layoutOff, '1', 'H hides the focused panel');
  (async () => {
    await t.btn('Save').fire('click', {});
    await null; await null;
    const ps = t.calls.save[0].breakpoints.wide.panels;
    eq(ps.filter((p) => p.hidden).map((p) => p.id).sort(), ['map', 'video']);
    assert.ok(ps.filter((p) => !p.hidden).every((p) => !('hidden' in p)), 'a shown panel has no hidden key: the export format is unchanged');
    show.fire('click', {});
    assert.strictEqual(t.targets.get('video').dataset.layoutOff, undefined);
    eq(t.at('video'), [1, 1, 6, 4], 'shown again it takes its cells back');
    eq(t.at('now')[1], 5, 'and pushes Hero now down again');
  })().catch((e) => { console.error(e); process.exit(1); });
}

// ---- aspect lock: the game picture's height follows its width (4/3, plus the card head) ----
{
  const t = setup();
  // without data-aspect the corner sets both spans freely
  t.corner('map').fire('pointerdown', { clientX: 890, clientY: 230 });
  t.corner('map').fire('pointermove', { clientX: 790, clientY: 410 });
  t.corner('map').fire('pointerup', { clientX: 790, clientY: 410 });
  eq(t.at('map').slice(2), [2, 7]);
}
{
  // the stub grid: 1200 px, 12 columns, the editor's fallback gap of 12 px, rows of 48 px, 60 px of card head and padding
  const colw = (1200 - 11 * 12) / 12, rowsFor = (span) => Math.max(2, Math.ceil((60 + (span * colw + (span - 1) * 12) * 3 / 4 + 12) / (48 + 12)));
  const shell = setup;
  const t = shell({ aspect: { video: '4/3' } });
  assert.ok(t.handle('video').children.some((c) => c.className === 'lock' && c.textContent === '4:3' && !c.hidden), 'the handle says the panel is aspect-locked');
  assert.ok(t.handle('map').children.some((c) => c.className === 'lock' && c.hidden), 'and says nothing on a free panel');
  t.corner('video').fire('pointerdown', { clientX: 590, clientY: 230 });
  t.corner('video').fire('pointermove', { clientX: 390, clientY: 230 }); // two columns narrower, the pointer does not move down
  eq(t.at('video').slice(2), [4, rowsFor(4)], 'narrower: the rows follow');
  t.corner('video').fire('pointermove', { clientX: 390, clientY: 900 }); // dragging down alone does not stretch it
  eq(t.at('video').slice(2), [4, rowsFor(4)], 'the height cannot be dragged away from the width');
  t.corner('video').fire('pointerup', { clientX: 390, clientY: 900 });
  const v = t.targets.get('video');
  v.fire('keydown', { key: 'ArrowRight', shiftKey: true }); v.fire('keydown', { key: 'Enter' });
  eq(t.at('video').slice(2), [5, rowsFor(5)], 'the keyboard resize is locked too');
  v.fire('keydown', { key: 'ArrowRight' }); v.fire('keydown', { key: 'Enter' });
  eq(t.at('video').slice(2), [5, rowsFor(5)], 'a move keeps the size');
}

// ---- reduced motion: no slide ----
{
  const t = setup({ still: true });
  t.handle('map').fire('pointerdown', { clientX: 650, clientY: 30 });
  t.handle('map').fire('pointermove', { clientX: 50, clientY: 30 });
  assert.strictEqual(t.calls.raf, 0);
  assert.strictEqual(t.targets.get('video').dataset.flip, undefined);
  assert.strictEqual(t.targets.get('video').style.transform, undefined);
}

// ---- ?drag=map:1:1 shows a drag in progress and commits nothing ----
{
  const t = setup({ demo: 'map:1:1' });
  eq(t.at('map'), [1, 1, 3, 4]); assert.strictEqual(t.main.dataset.preview, 'map');
  t.targets.get('map').fire('keydown', { key: 'Escape' });
  eq(all(t), start);
}
