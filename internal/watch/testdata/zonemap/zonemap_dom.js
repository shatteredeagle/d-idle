// DOM goldens of the zone-map slot panel (web/panels/zone-map) in a stub DOM: the grid of the fixture act (here, target, two
// pins, a failed mark) for both classes, and the zone menu for a field, here, the town, too hard, a locked Loop and queue, the
// charter, Loop open, the queue full and an area that is not orderable. Also the helpers (grid, band, route, arrow keys), the
// two-click rule (a zone, then Go now, is one accepted order) and the words-only rule (no margin number in any text).
// UPDATE=1 rewrites zonemap_dom.txt. Usage: node zonemap_dom.js path/to/web/panels
const assert = require('assert'), fs = require('fs'), path = require('path');
const { load, mk } = require('../panel_harness.js');
const dir = process.argv[2];
const calls = [];
let reply = { ok: true };
const sendCommand = (ctx, intent, args, mode) => { calls.push({ intent, args, mode }); return Promise.resolve(reply); };
const P = load(path.join(dir, 'zone-map', 'panel.js'), { sendCommand, replacing: () => 0, document: { addEventListener() {} }, ResizeObserver: undefined });
const fx = (n) => JSON.parse(fs.readFileSync(path.join(__dirname, n), 'utf8'));

function node(tag, cls, text) {
  const e = mk(tag);
  e.className = cls || '';
  if (text !== undefined) e.textContent = text;
  e.style = { setProperty(k, v) { e.dataset['css' + k] = v; } };
  e.focus = () => {};
  return e;
}
function make() {
  const body = node('div', 'd-slotbody'), meta = node('span', 'meta'), root = node('div', 'd-panel');
  const sheet = node('div', 'd-tilesheet');
  sheet.hidden = true;
  const c = { body, meta, el: node, store: () => ({ get: () => null, set() {} }), hide() {}, state() {}, toast() {}, later() {}, post: () => Promise.resolve({ ok: true }),
    tileSheet: () => ({ el: sheet, place() { sheet.dataset.host = 'area-map'; } }),
    ui: { btn: (t, o) => { const b = node('button', 'd-btn', t); Object.assign(b.dataset, o); return b; }, clear(n) { n.children.length = 0; } } };
  return { c, root, sheet, inst: P.mount(root, c), body };
}
function dump(e, d) {
  const data = Object.keys(e.dataset || {}).filter((k) => !k.startsWith('css--')).sort().map((k) => k + '=' + e.dataset[k]).join(',');
  const at = ['title', 'type', 'disabled'].map((k) => (e[k] ? ' ' + k + '=' + JSON.stringify(e[k]) : '')).join('');
  const lbl = e.attrs && e.attrs['aria-label'] ? ' aria=' + JSON.stringify(e.attrs['aria-label']) : '';
  let s = '  '.repeat(d) + (e.hidden ? '[' : '') + e.tag + (e.className ? '.' + e.className.split(' ').join('.') : '') + (data ? ' {' + data + '}' : '') +
    (e.textContent ? ' ' + JSON.stringify(e.textContent) : '') + at + lbl + (e.hidden ? ']' : '') + '\n';
  for (const k of e.children) s += dump(k, d + 1);
  return s;
}
const find = (e, f, out = []) => { if (f(e)) out.push(e); e.children.forEach((k) => find(k, f, out)); return out; };
const text = (e) => find(e, () => true).map((x) => x.textContent || '').join(' ');
const btn = (e, key) => find(e, (x) => x.tag === 'button' && x.dataset.key === key)[0];

let out = '';
const classes = { sorceress: 'Cold Plains', paladin: 'Dark Wood' }, OPEN = { sorceress: 'cp', paladin: 'bm' }; // a zone the hero is not in
for (const cls of Object.keys(classes)) {
  const v = fx('map-fixture-' + cls + '.json');
  const clone = () => JSON.parse(JSON.stringify(v));
  const byName = (m, n) => m.nodes.find((x) => x.name === n);
  const area = classes[cls];
  const states = {
    grid: [clone(), null],
    'menu-field': [clone(), OPEN[cls]],
    'menu-here': [clone(), v.nodes.find((n) => n.status.here).id],
    'menu-town': [clone(), 'camp'],
    'menu-hard': [(() => { const m = clone(); byName(m, 'Forgotten Tower').status.hard = true; byName(m, 'Forgotten Tower').status.hardWhy = 'no mercenary and clvl 7 is below 15'; return m; })(), 'ftw'],
    'menu-not-orderable': [clone(), 'cell'],
    'menu-failed': [clone(), v.nodes.find((n) => n.status.failed).id],
    'menu-locked': [(() => { const m = clone(); m.ladder = { loopOpen: false, loopTeaser: 'Send the hero out, then repair its gear yourself once.', queueOpen: false, queueTeaser: 'Finish three loop rounds.' }; return m; })(), OPEN[cls]],
    'menu-queue-full': [(() => { const m = clone(); m.ladder = { loopOpen: true, queueOpen: true, queueSlots: 2, queueWaiting: 2, queueTeaser: 'Clear the Den of Evil.' }; return m; })(), OPEN[cls]],
    'menu-no-ladder': [(() => { const m = clone(); delete m.ladder; return m; })(), OPEN[cls]],
  };
  for (const [name, [m, sel]] of Object.entries(states)) {
    const { inst, sheet, body } = make();
    inst.update(m);
    if (sel) body.children[0].children[0].children.find((b) => b.dataset && b.dataset.id === sel).fire('click');
    out += '== ' + name + ' ' + cls + '\n' + dump(body, 0) + (sel ? '-- menu\n' + dump(sheet, 0) : '');
    assert.ok(!/margin\D{0,12}\d/i.test(text(sheet) + text(body)), 'a margin number is shown');
  }
  // Loop open: the three controls and Start loop
  {
    const { inst, sheet, body } = make();
    inst.update(clone());
    body.children[0].children[0].children.find((b) => b.dataset && b.dataset.id === OPEN[cls]).fire('click');
    btn(sheet, 'loop').fire('click');
    out += '== menu-loop-open ' + cls + '\n' + dump(sheet, 0);
    for (const k of ['loop-min', 'loop-repair', 'loop-rounds', 'loop-start']) assert.ok(find(sheet, (x) => x.dataset && x.dataset.key === k).length === 1, k);
  }
}

// the charter flow, awaited: Go now answers 402, the menu shows the price and Pay and go sends the order again approved
const charterDone = (async () => {
  const v = fx('map-fixture-sorceress.json');
  const { inst, sheet, body } = make();
  inst.update(v);
  body.children[0].children[0].children.find((b) => b.dataset && b.dataset.id === 'cp').fire('click');
  calls.length = 0;
  reply = { ok: false, code: 'charter-needed', charter: { price: 20, text: 'Above range.' } };
  btn(sheet, 'go').fire('click');
  await new Promise((r) => setImmediate(r));
  assert.ok(/Charter: 20 tally/.test(text(sheet)), 'the charter price is shown inline');
  out += '== menu-charter\n' + dump(sheet, 0);
  reply = { ok: true };
  btn(sheet, 'charter').fire('click');
  await new Promise((r) => setImmediate(r));
  assert.strictEqual(JSON.stringify(calls.map((c) => [c.intent, c.mode, !!(c.args && c.args.approve)])), JSON.stringify([['go', 'now', false], ['go', 'now', true]]));
  assert.ok(sheet.hidden, 'the menu closes after the accepted order');
  // two clicks: a zone, then Go now, is one accepted order
  calls.length = 0;
  const m2 = make();
  m2.inst.update(v);
  let clicks = 0;
  const z = m2.body.children[0].children[0].children.find((b) => b.dataset && b.dataset.id === 'bg');
  z.fire('click'); clicks++;
  btn(m2.sheet, 'go').fire('click'); clicks++;
  await new Promise((r) => setImmediate(r));
  assert.strictEqual(clicks, 2);
  assert.strictEqual(JSON.stringify(calls), JSON.stringify([{ intent: 'go', args: { area: 'Burial Grounds', until: { minutes: 20 } }, mode: 'now' }]));
  // queue and loop go through the one dispatcher
  calls.length = 0;
  const m3 = make();
  m3.inst.update(v);
  m3.body.children[0].children[0].children.find((b) => b.dataset && b.dataset.id === 'bg').fire('click');
  btn(m3.sheet, 'queue').fire('click');
  await new Promise((r) => setImmediate(r));
  assert.strictEqual(JSON.stringify(calls.map((c) => [c.intent, c.mode])), JSON.stringify([['go', 'queue']]));
})();

// an act without map data lists the orderable areas
{
  const { inst, body } = make();
  inst.update(fx('map-act2-sorceress.json'));
  out += '== no-map-act\n' + dump(body, 0);
  assert.ok(find(body, (x) => x.className === 'zm-n zm-row').length > 5, 'the list shows the orderable areas');
}

// a failed first fetch (the server stalled, 500, a timeout) leaves a sentence in the tile, never a blank box; the first answer then
// replaces it with the grid (the user's "act screen is gone": only the legend title was there)
{
  const { inst, body } = make();
  inst.update(null, { status: 0, fails: 1 });
  const em = find(body, (x) => x.className === 'zm-empty')[0], gr = find(body, (x) => x.className === 'zm-grid')[0];
  assert.ok(!em.hidden && /Waiting for the bot/.test(em.textContent), 'a failed first fetch says so');
  assert.ok(gr.hidden, 'no empty grid next to it');
  inst.update(fx('map-fixture-sorceress.json'));
  assert.ok(em.hidden && !gr.hidden && find(body, (x) => x.className === 'zm-n').length > 5, 'the first answer shows the zones');
  inst.update(null, { status: 0, fails: 1 });
  assert.ok(em.hidden && !gr.hidden, 'a later failed poll keeps the zones it has');
}
// a zone that makes the draw throw leaves a sentence, and the panel still reports ready
{
  const { inst, body } = make();
  const m = JSON.parse(JSON.stringify(fx('map-fixture-sorceress.json')));
  m.nodes[0].name = { toString() { throw new Error('boom'); } };
  inst.update(m);
  const em = find(body, (x) => x.className === 'zm-empty')[0];
  assert.ok(!em.hidden && /could not be drawn/.test(em.textContent), 'a draw failure is said, not blank');
}

// the helpers
const v1 = fx('map-fixture-sorceress.json');
const g = P.gridOf(v1.nodes);
assert.strictEqual(g.cols, 3, 'three columns for Act I');
assert.strictEqual(g.rows, 10, 'ten rows for Act I');
assert.ok(v1.nodes.every((n) => g.at(n).c >= 0 && g.at(n).r >= 0));
assert.strictEqual(P.band({ orderable: false }, 5), 'no');
assert.strictEqual(P.band({ orderable: true, range: 'clvl 6-12', status: {} }, 5), 'above');
assert.strictEqual(P.band({ orderable: true, range: 'clvl 6-12', status: {} }, 8), 'in');
assert.strictEqual(P.band({ orderable: true, range: 'clvl 6-12', status: {} }, 20), 'below');
assert.strictEqual(P.band({ orderable: true, range: 'clvl 6-12', status: { hard: true } }, 8), 'hard');
assert.strictEqual(P.rightText({ status: { here: true }, range: 'clvl 1-8' }), 'here');
assert.strictEqual(P.rightText({ status: { queued: 2 }, range: 'clvl 1-8' }), '#2');
assert.strictEqual(P.rightText({ status: {}, range: 'clvl 1-8' }), '1-8');
assert.strictEqual(JSON.stringify(P.routeTo(v1.nodes, v1.edges, v1.town, 'cp').slice(0, 2)), JSON.stringify([v1.town, 'bm']));
const nd = (id) => v1.nodes.find((n) => n.id === id);
assert.strictEqual(P.walk(v1.nodes, 'bm', 'ArrowRight'), 'cp');
assert.strictEqual(P.walk(v1.nodes, 'bm', 'ArrowUp'), 'den');
assert.strictEqual(P.walk(v1.nodes, 'camp', 'ArrowLeft'), 'camp');
assert.ok(nd('bm'));

charterDone.then(() => {
  const f = path.join(__dirname, 'zonemap_dom.txt');
  if (process.env.UPDATE) fs.writeFileSync(f, out);
  else assert.strictEqual(out, fs.readFileSync(f, 'utf8'), 'zonemap_dom.txt differs (UPDATE=1 rewrites)');
}).catch((e) => { console.error(e); process.exit(1); });
