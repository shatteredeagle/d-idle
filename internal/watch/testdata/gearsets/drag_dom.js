// Behaviour checks of web/inventory-drag.js (docs/gearsets.md) in node with a tiny DOM: the verdict table (green, red
// with the server's reason, nothing) for a Paladin and a Sorceress view, the pointer flow of a mouse and of a finger
// (threshold, hold, scroll cancel), what a drop calls, the click swallowed after a drag, Escape, and the armed-item path
// for the keyboard. Usage: node drag_dom.js path/to/inventory-drag.js
const fs = require('fs'), vm = require('vm'), assert = require('assert'), path = require('path');
const views = JSON.parse(fs.readFileSync(path.join(__dirname, 'view.json'), 'utf8'));
const src = fs.readFileSync(process.argv[2], 'utf8');

function mk(tag, props) {
  const e = Object.assign({ tag, className: '', dataset: {}, children: [], textContent: '', attrs: {}, hidden: false, style: {}, id: '', rect: null, parent: null }, props || {});
  e.classList = { add(c) { e.className = (e.className + ' ' + c).trim(); }, contains(c) { return e.className.split(/\s+/).includes(c); } };
  e.appendChild = (k) => { k.parent = e; e.children.push(k); return k; };
  e.removeChild = (k) => { e.children = e.children.filter((x) => x !== k); k.parent = null; return k; };
  e.setAttribute = (k, v) => { e.attrs[k] = String(v); };
  e.removeAttribute = (k) => { delete e.attrs[k]; };
  e.hasAttribute = (k) => k in e.attrs;
  e.querySelector = (s) => { const c = s.slice(1); const f = (n) => { for (const k of n.children) { if (k.classList.contains(c)) return k; const r = f(k); if (r) return r; } return null; }; return f(e); };
  e.closest = (sel) => {
    const test = (n) => sel.split(',').some((part) => {
      part = part.trim();
      if (part === '.d-slot2[data-slot]') return n.classList.contains('d-slot2') && n.dataset.slot !== undefined;
      if (part === '[data-key^="item-"]') return (n.dataset.key || '').startsWith('item-');
      if (part === '#pack' || part === '#stash') return n.id === part.slice(1);
      return part[0] === '.' && n.classList.contains(part.slice(1));
    });
    for (let n = e; n; n = n.parent) if (test(n)) return n;
    return null;
  };
  return e;
}

function page(view) {
  const L = { doc: {}, timers: [] }, fetched = [], said = [];
  const body = mk('body'), pack = mk('div', { id: 'pack' }), stash = mk('div', { id: 'stash' }), card = mk('div', { className: 'd-card' });
  const doll = mk('div'), all = [];
  [card, pack, stash, doll].forEach((n) => body.appendChild(n));
  const reg = (n, r) => { n.rect = r; all.push(n); return n; };
  reg(pack, [0, 300, 400, 400]); reg(stash, [0, 500, 400, 600]);
  const tile = (id, x) => { const t = mk('button', { dataset: { key: 'item-' + id }, className: 'item' }); pack.appendChild(t); return reg(t, [x, 300, x + 20, 340]); };
  const slotEl = (slot, x, filled, owner) => {
    const s = mk('div', { className: 'd-slot2', dataset: { slot } }); if (filled) s.dataset.filled = ''; if (owner) s.dataset.owner = owner;
    doll.appendChild(s); return reg(s, [x, 100, x + 30, 130]);
  };
  const t3 = tile(3, 0), t4 = tile(4, 30), t5 = tile(5, 60), t7 = tile(7, 90), t9 = tile(9, 120);
  const st8 = mk('button', { dataset: { key: 'item-8' }, className: 'item' }); stash.appendChild(st8); reg(st8, [0, 500, 20, 540]);
  const slots = { head: slotEl('head', 0, false), rhand: slotEl('rhand', 40, true), lhand: slotEl('lhand', 80, true), torso: slotEl('torso', 120, true),
    rring: slotEl('rring', 160, false), lring: slotEl('lring', 200, false), mhead: slotEl('mhead', 240, true, 'merc'), mrhand: slotEl('mrhand', 280, false, 'merc') };
  reg(card, [0, 0, 999, 99]); // the header card; the page background below is "outside"
  const hit = (x, y) => { let best = null; for (const n of all) { const r = n.rect; if (x >= r[0] && x < r[2] && y >= r[1] && y < r[3]) best = n; } return best; };
  const document = {
    body, createElement: mk, getElementById: (i) => (i === 'pack' ? pack : i === 'stash' ? stash : null),
    elementFromPoint: (x, y) => hit(x, y),
    querySelectorAll: (s) => (s === '.d-slot2[data-slot]' ? Object.values(slots) : s === '[data-over]' ? all.filter((n) => n.dataset.over !== undefined) : []),
    querySelector: (s) => (s === '.item.sel' ? all.find((n) => n.classList.contains('sel') && n.classList.contains('item')) || null : null),
    addEventListener: (t, f, o) => { (L.doc[t] = L.doc[t] || []).push({ f, o }); },
    removeEventListener: (t, f) => { L.doc[t] = (L.doc[t] || []).filter((x) => x.f !== f); },
    dispatchEvent: (e) => { (L.doc[e.type] || []).forEach((x) => x.f(e)); },
  };
  body.removeAttribute = ((orig) => (k) => { orig(k); })(body.removeAttribute);
  const calls = [];
  const api = { equip: (...a) => calls.push(['equip', ...a]), unequip: (...a) => calls.push(['unequip', ...a]), stash: (...a) => calls.push(['stash', ...a]),
    take: (...a) => calls.push(['take', ...a]), trashConfirm: (...a) => calls.push(['trash', ...a]), say: (t) => said.push(t) };
  const window = { __dragTest: (h) => { L.h = h; }, d2InvActions: api };
  class CustomEvent { constructor(type, o) { this.type = type; this.detail = o && o.detail; } }
  const sandbox = { document, window, JSON, Math, Object, Array, String, Number, RegExp, Date, console, CustomEvent,
    setTimeout: (f, ms) => { L.timers.push({ f, ms }); return L.timers.length; }, clearTimeout: (id) => { if (L.timers[id - 1]) L.timers[id - 1].f = null; } };
  vm.runInNewContext(src, sandbox);
  L.h.init(); L.h.setView(view);
  const fire = (type, e) => { e = Object.assign({ target: null, stopPropagation() { this.stopped = true; }, preventDefault() { this.prevented = true; }, cancelable: true }, e); (L.doc[type] || []).forEach((x) => x.f(e)); return e; };
  return { L, h: L.h, body, tiles: { 3: t3, 4: t4, 5: t5, 7: t7, 9: t9, 8: st8 }, slots, pack, stash, calls, said, fire, fetched, document };
}

let bad = 0;
const J = (x) => JSON.stringify(x);
const same = (a, b) => assert.strictEqual(J(a), J(b));
const t = (name, fn) => { try { fn(); } catch (e) { bad++; console.error('FAIL ' + name + ': ' + (e && e.message)); } };

for (const cls of ['paladin', 'sorceress']) {
  const v = views[cls];
  t(cls + ' verdict table', () => {
    const p = page(v), pick = (id) => p.h.find(v, { id }), slotV = (src, slot, owner) => p.h.verdict(v, src, { type: 'slot', slot, owner: owner || 'hero' });
    same(slotV(pick(3), 'head'), { kind: 'ok', act: 'equip', id: 3, slot: 'head' });
    const nohead = slotV(pick(3), 'torso');
    assert.strictEqual(nohead.kind, 'no'); assert.strictEqual(nohead.reason, 'does not fit that slot');
    assert.strictEqual(slotV(pick(4), 'rring').kind, 'ok'); assert.strictEqual(slotV(pick(4), 'lring').kind, 'ok'); assert.strictEqual(slotV(pick(4), 'rhand').kind, 'no');
    const two = slotV(pick(5), 'rhand');
    assert.strictEqual(two.kind, 'no'); assert.ok(/two-handed weapon and a shield/.test(two.reason), two.reason);
    assert.strictEqual(slotV(pick(8), 'head').reason, 'move it to the backpack first'); // a stash item cannot be dropped on a slot
    const mercDrop = slotV(pick(3), 'mhead', 'merc');
    assert.strictEqual(mercDrop.kind, 'no'); assert.ok(/mercenary/.test(mercDrop.reason));
    const free = JSON.parse(JSON.stringify(v)); free.merc.actWhy = '';
    assert.strictEqual(p.h.verdict(free, pick(3), { type: 'slot', slot: 'mhead', owner: 'merc' }).target, 'merc'); // an executor would make it green
    // worn items: take off goes to the pack; the full pack refuses; trash outside is refused with the reason
    const worn = p.h.find(v, { slot: 'rhand' }), plate = p.h.find(v, { slot: 'torso' });
    same(p.h.verdict(v, worn, { type: 'pack' }), { kind: 'ok', act: 'unequip', slot: 'rhand' });
    assert.strictEqual(p.h.verdict(v, plate, { type: 'pack' }).kind, 'no');
    assert.strictEqual(p.h.verdict(v, worn, { type: 'outside' }).kind, 'no');
    assert.strictEqual(p.h.verdict(v, worn, { type: 'slot', slot: 'head' }).kind, 'none');
    const mw = p.h.find(v, { slot: 'mhead', owner: 'merc' });
    assert.strictEqual(p.h.verdict(v, mw, { type: 'pack' }).kind, 'no');
    // pack item outside: trash through its confirm; a quest item is refused with the server's reason
    same(p.h.verdict(v, pick(3), { type: 'outside' }), { kind: 'ok', act: 'trash', id: 3, key: 'item-3' });
    assert.strictEqual(p.h.verdict(v, pick(7), { type: 'outside' }).reason, 'this item is never trashed (quest item)');
    assert.strictEqual(p.h.verdict(v, pick(3), { type: 'other' }).kind, 'none');
    // pack <-> stash
    assert.strictEqual(p.h.verdict(v, pick(9), { type: 'stash' }).act, 'stash');
    assert.strictEqual(p.h.verdict(v, pick(8), { type: 'pack' }).act, 'take');
  });

  t(cls + ' mouse drag onto a slot queues the menu equip', () => {
    const p = page(v);
    p.fire('pointerdown', { target: p.tiles[3], button: 0, isPrimary: true, pointerId: 1, pointerType: 'mouse', clientX: 5, clientY: 310 });
    p.fire('pointermove', { pointerId: 1, clientX: 7, clientY: 311 });
    assert.strictEqual(p.h.state().drag, null, 'below the threshold nothing starts (a click stays a click)');
    p.fire('pointermove', { pointerId: 1, clientX: 30, clientY: 200 });
    assert.ok(p.h.state().drag, 'the drag started');
    assert.ok(p.body.hasAttribute('data-dragging'));
    assert.strictEqual(p.slots.head.dataset.drop, 'ok'); assert.strictEqual(p.slots.torso.dataset.drop, 'no'); assert.strictEqual(p.slots.rring.dataset.drop, 'no');
    p.fire('pointermove', { pointerId: 1, clientX: 10, clientY: 110 }); // over the (empty) head slot
    assert.strictEqual(p.slots.head.dataset.over, 'ok');
    assert.ok(/Equip/.test(p.h.state().ghost.children[1].textContent));
    p.fire('pointermove', { pointerId: 1, clientX: 130, clientY: 110 }); // over the plate: red with the reason
    assert.strictEqual(p.slots.torso.dataset.over, 'no'); assert.strictEqual(p.slots.head.dataset.over, undefined);
    assert.strictEqual(p.h.state().ghost.children[1].textContent, 'does not fit that slot');
    p.fire('pointermove', { pointerId: 1, clientX: 10, clientY: 110 });
    p.fire('pointerup', { pointerId: 1, clientX: 10, clientY: 110 });
    same(p.calls, [['equip', 3, 'head', undefined]]);
    assert.ok(!p.body.hasAttribute('data-dragging'));
    assert.strictEqual(p.slots.head.dataset.drop, undefined, 'marks are cleared');
    const click = p.fire('click', { target: p.tiles[3] }); assert.ok(click.stopped, 'the click after the drop must not open the menu');
  });

  t(cls + ' an invalid drop only says why; escape and cancel drop nothing', () => {
    const p = page(v);
    p.fire('pointerdown', { target: p.tiles[5], button: 0, isPrimary: true, pointerId: 1, pointerType: 'mouse', clientX: 65, clientY: 310 });
    p.fire('pointermove', { pointerId: 1, clientX: 90, clientY: 120 });
    p.fire('pointerup', { pointerId: 1, clientX: 50, clientY: 110 }); // over the weapon slot: the shield is worn, the two-hander refuses
    same(p.calls, []); assert.ok(/two-handed weapon and a shield/.test(p.said[0]));
    const q = page(v);
    q.fire('pointerdown', { target: q.tiles[3], button: 0, isPrimary: true, pointerId: 2, pointerType: 'mouse', clientX: 5, clientY: 310 });
    q.fire('pointermove', { pointerId: 2, clientX: 30, clientY: 200 });
    q.fire('keydown', { key: 'Escape' });
    assert.strictEqual(q.h.state().drag, null); same(q.calls, []);
    q.fire('pointerdown', { target: q.tiles[3], button: 0, isPrimary: true, pointerId: 3, pointerType: 'mouse', clientX: 5, clientY: 310 });
    q.fire('pointermove', { pointerId: 3, clientX: 30, clientY: 200 });
    q.fire('pointercancel', { pointerId: 3 });
    assert.strictEqual(q.h.state().drag, null); same(q.calls, []);
  });

  t(cls + ' worn to pack unequips, outside trashes through the confirm, stash and take', () => {
    const p = page(v);
    const go = (target, x0, y0, x1, y1) => { p.fire('pointerdown', { target, button: 0, isPrimary: true, pointerId: 1, pointerType: 'mouse', clientX: x0, clientY: y0 }); p.fire('pointermove', { pointerId: 1, clientX: x1, clientY: y1 }); p.fire('pointerup', { pointerId: 1, clientX: x1, clientY: y1 }); };
    go(p.slots.rhand, 45, 110, 200, 320); // the pack
    go(p.tiles[3], 5, 310, 600, 800); // outside every card
    go(p.tiles[9], 125, 310, 100, 520); // onto the stash
    go(p.tiles[8], 5, 510, 100, 320); // stash -> pack
    go(p.slots.torso, 125, 110, 200, 320); // a full pack: refused, says why
    same(p.calls, [['unequip', 'rhand', undefined], ['trash', 'item-3'], ['stash', 9], ['take', 8]]);
    assert.ok(/no free spot/.test(p.said[0]));
    // the merc's worn helm onto the pack: red with the executor reason, nothing is called
    p.calls.length = 0; p.said.length = 0;
    go(p.slots.mhead, 245, 110, 200, 320);
    same(p.calls, []); assert.ok(/mercenary/.test(p.said[0]));
  });

  t(cls + ' a finger needs a hold and scrolls the page when it moves first', () => {
    const p = page(v);
    p.fire('pointerdown', { target: p.tiles[3], button: 0, isPrimary: true, pointerId: 1, pointerType: 'touch', clientX: 5, clientY: 310 });
    assert.strictEqual(p.L.timers.length, 1);
    p.fire('pointermove', { pointerId: 1, clientX: 5, clientY: 360 }); // moved before the hold: a scroll
    assert.strictEqual(p.h.state().pend, null); assert.strictEqual(p.h.state().drag, null);
    assert.strictEqual(p.L.timers[0].f === null, true, 'the hold timer is cleared');
    p.fire('pointerdown', { target: p.tiles[3], button: 0, isPrimary: true, pointerId: 2, pointerType: 'touch', clientX: 5, clientY: 310 });
    p.L.timers[1].f(); // the hold elapsed
    assert.ok(p.h.state().drag, 'held: the drag started');
    assert.ok((p.L.doc.touchmove || []).some((x) => x.o && x.o.passive === false), 'touchmove is registered non-passive to block the scroll');
    const ctx = p.fire('contextmenu', {}); assert.ok(ctx.prevented, 'no context menu while held');
    p.fire('pointermove', { pointerId: 2, clientX: 10, clientY: 110 });
    p.fire('pointerup', { pointerId: 2, clientX: 10, clientY: 110 });
    same(p.calls, [['equip', 3, 'head', undefined]]);
    same((p.L.doc.touchmove || []), [], 'the scroll block is removed again');
  });

  t(cls + ' an armed item and an empty slot (keyboard and tap)', () => {
    const p = page(v);
    p.tiles[3].classList.add('sel');
    const c = p.fire('click', { target: p.slots.head });
    same(p.calls, [['equip', 3, 'head', undefined]]); assert.ok(c.stopped);
    p.calls.length = 0;
    const c2 = p.fire('click', { target: p.slots.torso }); // a worn slot: the menu keeps its click
    assert.ok(!c2.stopped); same(p.calls, []);
    p.tiles[3].className = 'item'; p.tiles[5].classList.add('sel');
    const c3 = p.fire('click', { target: p.slots.rring });
    same(p.calls, []); assert.ok(p.said.length === 1, 'a refusal says why'); // the two-hander does not fit a ring finger
  });
}

// re-render safety: the view the drag keeps is replaced by the d2:inventory events, and a locked sets block changes nothing here
t('view updates', () => {
  const p = page(views.paladin);
  const v2 = JSON.parse(JSON.stringify(views.paladin)); v2.backpack[0].canWear = [];
  p.document.dispatchEvent({ type: 'd2:inventory', detail: v2 });
  assert.strictEqual(p.h.verdict(v2, p.h.find(v2, { id: 3 }), { type: 'slot', slot: 'head', owner: 'hero' }).kind, 'no');
});

if (bad) { console.error(bad + ' failure(s)'); process.exit(1); }
console.log('drag ok');
