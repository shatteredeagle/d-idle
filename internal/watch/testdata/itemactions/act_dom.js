// DOM goldens and behaviour checks of web/inventory-actions.js (docs/inventory-actions.md): the menu of a backpack item,
// a ring with two fingers, a quest item, a stash item, a pinned worn slot, the trash confirm step, queued and finished
// requests, for a Paladin and a Sorceress view; keyboard, Escape, sheet mode, no scrolling calls, no network from the
// menu until a choice. UPDATE=1 rewrites the goldens. Usage: node act_dom.js path/to/inventory-actions.js
const fs = require('fs'), vm = require('vm'), assert = require('assert'), path = require('path');
const dir = __dirname;

function mk(tag) {
  const e = { tag, className: '', dataset: {}, children: [], textContent: '', attrs: {}, listeners: {}, hidden: false, style: {}, id: '' };
  e.classList = { add(c) { e.className = (e.className + ' ' + c).trim(); } };
  Object.defineProperty(e, 'firstChild', { get() { return e.children[0] || null; } });
  e.appendChild = (k) => { k.parent = e; e.children.push(k); return k; };
  e.removeChild = (k) => { e.children = e.children.filter((x) => x !== k); return k; };
  e.setAttribute = (k, v) => { e.attrs[k] = String(v); };
  e.hasAttribute = (k) => k in e.attrs;
  e.removeAttribute = (k) => { delete e.attrs[k]; };
  e.addEventListener = (t, f) => { (e.listeners[t] = e.listeners[t] || []).push(f); };
  e.contains = (x) => x === e || e.children.some((c) => c.contains && c.contains(x));
  e.offsetWidth = 200; e.offsetHeight = 150;
  e.focus = (o) => { if (!o || o.preventScroll !== true) throw new Error('focus must pass preventScroll'); focused = e; };
  e.scrollIntoView = () => { throw new Error('the menu must never scroll the page'); };
  e.querySelector = (s) => { const c = s.slice(1); const f = (n) => { for (const k of n.children) { if (k.className.split(/\s+/).includes(c)) return k; const r = f(k); if (r) return r; } return null; }; return f(e); };
  e.querySelectorAll = (s) => { const out = []; const f = (n) => n.children.forEach((k) => { if (k.tag === 'button' && k.className.split(/\s+/).includes('act')) out.push(k); f(k); }); f(e); return out; };
  return e;
}
let focused = null, cur = 'paladin';
function ser(e, d) {
  const attrs = Object.keys(e.attrs).sort().map((k) => k + '=' + e.attrs[k]);
  const data = Object.keys(e.dataset).sort().map((k) => 'data-' + k + '=' + e.dataset[k]);
  const cls = e.className.split(/\s+/).filter(Boolean).sort().join('.');
  let out = '  '.repeat(d) + e.tag + (cls ? '.' + cls : '') + ' [' + attrs.concat(data).join(' ') + ']' + (e.textContent ? ' ' + JSON.stringify(e.textContent) : '') + (e.hidden ? ' hidden' : '') + '\n';
  e.children.forEach((c) => { out += ser(c, d + 1); });
  return out;
}
function page(src) {
  const anchors = {}, docL = {}, winL = {}, boxes = {}, fetched = [], body = mk('body');
  const getA = (key) => {
    if (!anchors[key]) {
      const a = mk('button'); a.dataset.key = key; a.closest = () => a;
      a.getBoundingClientRect = () => ({ left: 100, top: 100, right: 150, bottom: 140 });
      anchors[key] = a;
    }
    return anchors[key];
  };
  const document = {
    body, documentElement: { clientWidth: 1000, clientHeight: 700 }, createElement: mk,
    getElementById: (i) => (boxes[i] = boxes[i] || mk('div')),
    querySelector: (s) => { const m = /data-key="([^"]+)"/.exec(s); return m ? getA(m[1]) : null; },
    addEventListener: (t, f) => { (docL[t] = docL[t] || []).push(f); },
  };
  let h = null;
  const window = { addEventListener: (t, f) => { (winL[t] = winL[t] || []).push(f); }, __actTest: (x) => { h = x; } };
  const fetch = (u, o) => { fetched.push({ u, body: o && o.body }); return Promise.resolve({ ok: true, json: () => Promise.resolve(/inventory$/.test(u) ? views[cur] : { ok: true, answerBase: 'http://x' }) }); };
  vm.runInNewContext(src, { document, window, JSON, Math, Object, Array, String, Number, RegExp, Promise, console, fetch, location: { protocol: 'http:', hostname: 'h' }, setTimeout });
  return { h, document, body, getA, docL, winL, boxes, fetched, window };
}
const views = JSON.parse(fs.readFileSync(path.join(dir, 'act_view.json'), 'utf8'));
const src = fs.readFileSync(process.argv[2], 'utf8');
let bad = 0;
const check = (name, got) => {
  const file = path.join(dir, 'act_dom_' + name + '.nopin.txt');
  if (process.env.UPDATE) { fs.writeFileSync(file, got); return; }
  if (!fs.existsSync(file)) { console.error('missing golden ' + file); bad++; return; }
  if (fs.readFileSync(file, 'utf8') !== got) { console.error('menu differs from ' + file + '\n' + got); bad++; }
};

// ---- goldens: one menu per state and class
const states = { sword: 'item-1', wand: 'item-2', ring: 'item-3', quest: 'item-4', stash: 'item-5', worn: 'slot-torso', keep: 'item-8' };
for (const cls of ['paladin', 'sorceress']) {
  for (const [name, key] of Object.entries(states)) {
    const p = page(src); p.h.init(); p.h.onView(views[cls]); p.h.open(key);
    check(cls + '_' + name, ser(p.h.state().node, 0));
  }
}
{ // the trash confirm step: first click asks, the entry turns into the confirm and a way back appears
  const p = page(src); p.h.init(); p.h.onView(views.paladin); p.h.open('item-1');
  const m = p.h.state().node, trashBtn = () => m.querySelectorAll().find((b) => b.dataset.act === 'trash');
  trashBtn().listeners.click[0]();
  assert.strictEqual(p.fetched.filter((f) => /trash/.test(f.u)).length, 0, 'the first click must not queue');
  check('paladin_trash_confirm', ser(m, 0));
  assert.ok('confirm' in trashBtn().dataset);
}
{ // queue card with every state
  const p = page(src); p.h.init(); p.h.onView(views.sorceress); p.h.renderQueue();
  check('queue_card', ser(p.boxes.queue, 0) + 'title ' + p.boxes['t-queue'].textContent + '\n');
}

// ---- identify commands (schema 5): the menu entry, the errand strip and the queue card, own fixture act_view_identify.json
const idViews = JSON.parse(fs.readFileSync(path.join(dir, 'act_view_identify.json'), 'utf8'));
for (const cls of ['paladin', 'sorceress']) {
  for (const [name, key] of Object.entries({ ring: 'item-1', jewel: 'item-9', identified: 'item-2', stash: 'item-3', worn: 'slot-torso' })) {
    const p = page(src); p.h.init(); p.h.onView(idViews[cls]); p.h.open(key);
    check('identify_' + cls + '_' + name, ser(p.h.state().node, 0));
  }
  const p = page(src); p.h.init(); p.h.onView(idViews[cls]); p.h.renderErrands(); p.h.renderQueue();
  check('errands_' + cls, ser(p.boxes.errands, 0));
  check('queue_card_identify_' + cls, ser(p.boxes.queue, 0));
}

// ---- stash actions (schema 6): menu of a pack item, a refused pack item, a stash item, refused/identifiable stash items,
// the gold strip, the queue card and the pending badges; own fixture act_view_stash.json
const stViews = JSON.parse(fs.readFileSync(path.join(dir, 'act_view_stash.json'), 'utf8'));
for (const cls of ['paladin', 'sorceress']) {
  for (const [name, key] of Object.entries({ pack: 'item-1', packfull: 'item-7', take: 'item-3', takefull: 'item-5', notool: 'item-6' })) {
    const p = page(src); p.h.init(); p.h.onView(stViews[cls]); p.h.open(key);
    check('stash_' + cls + '_' + name, ser(p.h.state().node, 0));
  }
  const p = page(src); p.h.init(); p.h.onView(stViews[cls]); p.h.renderGold(); p.h.renderErrands(); p.h.renderQueue();
  check('stash_gold_' + cls, ser(p.boxes.stashgold, 0));
  check('stash_errands_' + cls, ser(p.boxes.errands, 0));
  check('stash_queue_card_' + cls, ser(p.boxes.queue, 0));
}

// ---- behaviour
const flush = () => new Promise((r) => setImmediate(r));
(async () => {
  for (const cls of ['paladin', 'sorceress']) {
    cur = cls; const p = page(src); p.h.init();
    const fire = (t, e) => (p.docL[t] || []).forEach((f) => f(e));
    p.h.onView(views[cls]);
    // a click on an item opens its menu and takes focus; a second click on it closes; Escape closes
    const a1 = p.getA('item-1');
    fire('click', { target: a1 }); assert.strictEqual(p.h.state().key, 'item-1'); assert.ok(focused && focused.tag === 'button');
    fire('click', { target: a1 }); assert.strictEqual(p.h.state().key, '');
    fire('click', { target: a1 }); fire('keydown', { key: 'Escape' }); assert.strictEqual(p.h.state().key, '');
    // an empty slot has no menu
    fire('click', { target: p.getA('slot-head') }); assert.strictEqual(p.h.state().key, '');
    // a refused entry says why and sends nothing
    fire('click', { target: p.getA('item-5') });
    const stashTrash = p.h.state().node.querySelectorAll().find((b) => b.dataset.act === 'trash');
    assert.strictEqual(stashTrash.attrs['aria-disabled'], 'true'); stashTrash.listeners.click[0]();
    assert.strictEqual(p.fetched.filter((f) => /\/api\/inventory\/(trash|equip|mark)/.test(f.u)).length, 0);
    fire('keydown', { key: 'Escape' });
    // arrow keys move focus among the entries and wrap
    fire('click', { target: p.getA('item-3') });
    const list = p.h.state().node.querySelectorAll(); const first = focused;
    fire('keydown', { key: 'End', preventDefault() {} }); assert.strictEqual(focused, list[list.length - 1]);
    fire('keydown', { key: 'ArrowDown', preventDefault() {} }); assert.strictEqual(focused, list[0]);
    assert.ok(first);
    fire('keydown', { key: 'Escape' });
    // sheet mode is a layout flag only
    p.h.setSheet(true); fire('click', { target: p.getA('item-3') }); assert.strictEqual(p.h.state().node.dataset.mode, 'sheet');
    fire('keydown', { key: 'Escape' });
    // an equip choice posts the item and the chosen slot, the menu closes
    p.h.setSheet(false); fire('click', { target: p.getA('item-3') });
    p.h.state().node.querySelectorAll().find((b) => b.dataset.act === 'equip-lring').listeners.click[0]();
    await flush(); await flush();
    const eq = p.fetched.find((f) => /\/api\/inventory\/equip/.test(f.u));
    assert.ok(eq && JSON.parse(eq.body).slot === 'lring' && JSON.parse(eq.body).id === 3, JSON.stringify(p.fetched));
    assert.strictEqual(p.h.state().key, '');
    // trash goes out only after the confirm click, with confirm true
    fire('click', { target: p.getA('item-1') });
    const m = p.h.state().node, tb = () => m.querySelectorAll().find((b) => b.dataset.act === 'trash');
    if (cls === 'paladin') { tb().listeners.click[0](); tb().listeners.click[0](); await flush(); await flush();
      const tr = p.fetched.find((f) => /\/api\/inventory\/trash/.test(f.u)); assert.ok(tr && JSON.parse(tr.body).confirm === true && JSON.parse(tr.body).id === 1, JSON.stringify(p.fetched)); }
    // an unchanged poll leaves the open menu's DOM alone; a vanished item closes it
    p.h.close(false); fire('click', { target: p.getA('item-3') });
    const node = p.h.state().node, firstChild = node.children[0];
    p.h.onView(JSON.parse(JSON.stringify(views[cls]))); assert.strictEqual(node.children[0], firstChild, 'unchanged poll rebuilt the menu');
    const gone = JSON.parse(JSON.stringify(views[cls])); gone.backpack = gone.backpack.filter((x) => x.id !== 3); p.h.onView(gone);
    assert.strictEqual(p.h.state().key, '');
  }
  // ---- identify commands: what a click sends, what stays local
  for (const cls of ['paladin', 'sorceress']) {
    cur = cls; const p = page(src); p.h.init();
    const fire = (t, e) => (p.docL[t] || []).forEach((f) => f(e));
    p.h.onView(idViews[cls]); p.h.renderErrands();
    const sent = (re) => p.fetched.filter((f) => re.test(f.u));
    const e = p.h.errands();
    // the Identify entry posts the item id; a refused one (no tool, stash) sends nothing and says why
    fire('click', { target: p.getA('item-9') });
    const idBtn = () => p.h.state().node.querySelectorAll().find((b) => b.dataset.act === 'identify');
    if (cls === 'paladin') {
      assert.ok(!idBtn().hasAttribute('aria-disabled'));
      idBtn().listeners.click[0](); await flush(); await flush();
      const r = sent(/\/api\/inventory\/identify/); assert.ok(r.length === 1 && JSON.parse(r[0].body).id === 9, JSON.stringify(p.fetched));
    } else {
      assert.strictEqual(idBtn().attrs['aria-disabled'], 'true'); idBtn().listeners.click[0]();
      assert.strictEqual(sent(/\/api\/inventory\/identify/).length, 0, 'a refused identify must not be sent');
    }
    fire('keydown', { key: 'Escape' });
    // an identified item has no Identify entry at all
    fire('click', { target: p.getA('item-2') });
    assert.ok(!p.h.state().node.querySelectorAll().some((b) => b.dataset.act === 'identify'));
    fire('keydown', { key: 'Escape' });
    // buy: a typed count is sent, an empty box means 0 (fill the tome), nonsense is refused locally
    const before = p.fetched.length;
    e.count.value = '12'; e.buy.listeners.click[0](); await flush(); await flush();
    const b1 = sent(/\/api\/inventory\/buyid/);
    if (cls === 'paladin') {
      assert.ok(b1.length === 1 && JSON.parse(b1[0].body).count === 12, JSON.stringify(p.fetched.slice(before)));
      e.count.value = ''; e.buy.listeners.click[0](); await flush(); await flush();
      assert.strictEqual(JSON.parse(sent(/\/api\/inventory\/buyid/)[1].body).count, 0);
      e.count.value = '99'; e.buy.listeners.click[0]();
      assert.strictEqual(sent(/\/api\/inventory\/buyid/).length, 2, 'out of range must stay local');
      assert.ok(e.note.textContent.length > 0);
    }
    // Cain: open posts, locked sends nothing and shows the teaser
    e.cain.listeners.click[0](); await flush(); await flush();
    if (cls === 'paladin') { assert.strictEqual(sent(/\/api\/inventory\/cain/).length, 1); assert.strictEqual(e.wrap.dataset.state, 'open'); }
    else { assert.strictEqual(sent(/\/api\/inventory\/cain/).length, 0); assert.strictEqual(e.wrap.dataset.state, 'locked'); assert.ok(/opens when Cain is rescued/.test(e.note.textContent)); assert.ok(/\bd-locked\b/.test(e.wrap.className)); }
    // a poll with unchanged data keeps what the player typed and rebuilds nothing
    e.count.value = '7'; const kids = e.box.children.length;
    p.h.onView(JSON.parse(JSON.stringify(idViews[cls]))); p.h.renderErrands();
    assert.strictEqual(e.count.value, '7'); assert.strictEqual(e.box.children.length, kids);
    // no identify block (an older bot): the strip stays hidden
    const old = JSON.parse(JSON.stringify(idViews[cls])); delete old.identify; p.h.onView(old); p.h.renderErrands(); assert.strictEqual(e.box.hidden, true);
  }
  { // stash behaviour: one click queues the move; the gold boxes validate and send op and count; pending badges follow the queue
    cur = 'paladin';
    const p = page(src); p.h.init(); p.h.onView(stViews.paladin);
    assert.strictEqual(p.getA('item-3').dataset.pending, 'identify'); assert.strictEqual(p.getA('item-1').dataset.pending, 'stash');
    p.h.open('item-7'); const off = p.h.state().node.querySelectorAll().find((b) => b.dataset.act === 'stash');
    assert.strictEqual(off.attrs['aria-disabled'], 'true'); off.listeners.click[0]();
    assert.strictEqual(p.fetched.filter((f) => /inventory\/(stash|take)$/.test(f.u)).length, 0, 'a refused move sends nothing');
    p.h.close();
    const q = JSON.parse(JSON.stringify(stViews.paladin)); q.queue = [];
    p.h.onView(q); assert.ok(!('pending' in p.getA('item-3').dataset), 'badge cleared once settled');
    p.h.open('item-3'); const tk = p.h.state().node.querySelectorAll().find((b) => b.dataset.act === 'take'); tk.listeners.click[0]();
    await flush(); await flush();
    assert.ok(p.fetched.some((f) => /inventory\/take$/.test(f.u) && JSON.parse(f.body).id === 3), 'take-out queued');
    p.h.onView(q); p.h.open('item-3'); const idb = p.h.state().node.querySelectorAll().find((b) => b.dataset.act === 'identify'); assert.ok(idb && !idb.attrs['aria-disabled'], 'identify in place is offered on the stash item');
    p.h.onView(JSON.parse(JSON.stringify(stViews.paladin))); await flush(); p.h.renderGold(); const g = p.h.goldBox();
    g.amt.value = '0'; g.dep.listeners.click[0](); g.amt.value = '99999'; g.wd.listeners.click[0]();
    assert.strictEqual(p.fetched.filter((f) => /inventory\/gold$/.test(f.u)).length, 0, 'bad amounts send nothing');
    g.amt.value = '1500'; g.dep.listeners.click[0](); await flush(); await flush();
    const gf = p.fetched.filter((f) => /inventory\/gold$/.test(f.u)); assert.strictEqual(gf.length, 1);
    assert.deepStrictEqual(JSON.parse(gf[0].body), { op: 'deposit', count: 1500 });
    g.amt.value = '77'; g.amt.listeners.input[0](); p.h.onView(JSON.parse(JSON.stringify(stViews.paladin))); await flush(); p.h.renderGold();
    assert.strictEqual(g.amt.value, '77', 'a typed amount survives the poll');
    const noBackend = JSON.parse(JSON.stringify(stViews.paladin)); delete noBackend.stashOps; p.h.onView(noBackend); p.h.renderGold(); assert.strictEqual(g.box.hidden, true);
  }
  // placement stays inside the viewport
  const pl = page(src).h.place;
  [[{ left: 100, top: 100, right: 150, bottom: 140 }, 200, 150, 1000, 700], [{ left: 900, top: 650, right: 960, bottom: 690 }, 200, 150, 1000, 700], [{ left: 10, top: 600, right: 60, bottom: 690 }, 300, 200, 390, 700]].forEach(([r, w, h, vw, vh]) => {
    const o = pl(r, w, h, vw, vh); assert.ok(o.x >= 8 && o.y >= 8 && o.x + w <= vw - 8 && o.y + h <= vh - 8, JSON.stringify([r, o]));
  });
  if (bad) process.exit(1);
  console.log('ok');
})();
