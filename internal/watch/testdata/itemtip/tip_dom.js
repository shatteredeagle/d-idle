// DOM goldens and behaviour checks of web/inventory-tip.js (docs/inventory-tooltip.md): the tooltip of a magic, rare,
// unique, set, unidentified, ethereal and low-quality item and a worn pinned slot, tap, tap-away, Escape, keyboard focus,
// poll updates that must not rebuild the tip, no scrolling calls. UPDATE=1 rewrites the goldens.
// Usage: node tip_dom.js path/to/inventory-tip.js
const fs = require('fs'), vm = require('vm'), assert = require('assert'), path = require('path');
const dir = __dirname;

function mk(tag) {
  const e = { tag, className: '', dataset: {}, children: [], textContent: '', attrs: {}, listeners: {}, hidden: false, style: {}, id: '' };
  e.classList = { add(c) { e.className = (e.className + ' ' + c).trim(); } };
  Object.defineProperty(e, 'firstChild', { get() { return e.children[0] || null; } });
  e.appendChild = (k) => { e.children.push(k); return k; };
  e.removeChild = (k) => { e.children = e.children.filter((x) => x !== k); return k; };
  e.setAttribute = (k, v) => { e.attrs[k] = String(v); };
  e.removeAttribute = (k) => { delete e.attrs[k]; };
  e.addEventListener = (t, f) => { (e.listeners[t] = e.listeners[t] || []).push(f); };
  e.contains = (x) => x === e || e.children.some((c) => c.contains && c.contains(x));
  e.offsetWidth = 200; e.offsetHeight = 120;
  e.scrollIntoView = () => { throw new Error('the tooltip must never scroll the page'); };
  return e;
}
function ser(e, d) {
  const attrs = Object.keys(e.attrs).sort().map((k) => k + '=' + e.attrs[k]);
  const data = Object.keys(e.dataset).sort().map((k) => 'data-' + k + '=' + e.dataset[k]);
  const cls = e.className.split(/\s+/).filter(Boolean).sort().join('.');
  let out = '  '.repeat(d) + e.tag + (cls ? '.' + cls : '') + ' [' + attrs.concat(data).join(' ') + ']' + (e.textContent ? ' ' + JSON.stringify(e.textContent) : '') + (e.hidden ? ' hidden' : '') + '\n';
  e.children.forEach((c) => { out += ser(c, d + 1); });
  return out;
}

function page(src) {
  const anchors = {}, docL = {}, winL = {}, boxes = {};
  const body = mk('body');
  const getA = (key) => {
    if (!anchors[key]) {
      const a = mk('button'); a.dataset.key = key; a.attrs.title = 'native title';
      a.closest = () => a; a.getBoundingClientRect = () => ({ left: 100, top: 100, right: 150, bottom: 140 });
      anchors[key] = a;
    }
    return anchors[key];
  };
  const document = {
    body, documentElement: { clientWidth: 1000, clientHeight: 700 },
    createElement: mk,
    getElementById: (i) => (boxes[i] = boxes[i] || mk('div')),
    querySelector: (s) => { const m = /data-key="([^"]+)"/.exec(s); return m ? (gone[m[1]] ? null : getA(m[1])) : null; },
    addEventListener: (t, f) => { (docL[t] = docL[t] || []).push(f); },
  };
  const gone = {};
  const window = { addEventListener: (t, f) => { (winL[t] = winL[t] || []).push(f); } };
  let h = null;
  window.__tipTest = (x) => { h = x; };
  const ctx = { document, window, JSON, Math, Object, Array, String, Number, RegExp, console };
  vm.runInNewContext(src, ctx);
  return { h, document, body, getA, docL, winL, boxes, gone };
}
const view = JSON.parse(fs.readFileSync(path.join(dir, 'tip_view.json'), 'utf8'));
const src = fs.readFileSync(process.argv[2], 'utf8');
let bad = 0;
const check = (name, got) => {
  const file = path.join(dir, 'tip_dom_' + name + '.nopin.txt');
  if (process.env.UPDATE) { fs.writeFileSync(file, got); return; }
  if (!fs.existsSync(file)) { console.error('missing golden ' + file); bad++; return; }
  if (fs.readFileSync(file, 'utf8') !== got) { console.error('tooltip differs from ' + file + '\n' + got); bad++; }
};

// ---- goldens: one tooltip per state
const states = { magic_game_text: 'item-1', rare_fallback: 'item-2', unique_game_text: 'item-3', set_game_text: 'item-4', unidentified: 'item-5',
  ethereal_socketed: 'item-6', low_quality: 'item-7', magic_plain: 'item-9', worn_pinned_unique: 'slot-rhand' };
for (const [name, key] of Object.entries(states)) {
  const p = page(src); p.h.init(); p.h.onView(view); p.h.show(key, 'hover');
  check(name, ser(p.h.state().node, 0));
}

// ---- name colour classes
{
  const p = page(src);
  assert.strictEqual(p.h.nameClass({ quality: 'normal', ethereal: true }), 'q-low');
  assert.strictEqual(p.h.nameClass({ quality: 'normal', sockets: 2 }), 'q-low');
  assert.strictEqual(p.h.nameClass({ quality: 'normal' }), 'q-normal');
  assert.strictEqual(p.h.nameClass({ quality: 'low' }), 'q-low');
  ['magic', 'rare', 'unique', 'set', 'crafted'].forEach((q) => assert.strictEqual(p.h.nameClass({ quality: q, ethereal: true }), 'q-' + q));
}

// ---- placement stays inside the viewport
{
  const p = page(src), pl = p.h.place;
  const inside = (r, w, h, vw, vh) => { const o = pl(r, w, h, vw, vh); assert.ok(o.x >= 8 && o.y >= 8 && o.x + w <= vw - 8 && o.y + h <= vh - 8, JSON.stringify([r, o])); return o; };
  assert.strictEqual(inside({ left: 100, top: 100, right: 150, bottom: 140 }, 200, 120, 1000, 700).x, 158); // right of the anchor
  assert.strictEqual(inside({ left: 800, top: 100, right: 880, bottom: 140 }, 200, 120, 1000, 700).x, 592); // no room right: left of it
  inside({ left: 10, top: 600, right: 60, bottom: 690 }, 300, 200, 390, 700); // phone: below/above, clamped
  inside({ left: 0, top: 0, right: 40, bottom: 40 }, 340, 280, 360, 300); // tip nearly as large as the screen (CSS caps the width at the viewport minus the margins)
}

// ---- behaviour
{
  const p = page(src); p.h.init(); p.h.onView(view);
  const fire = (l, t, e) => (l[t] || []).forEach((f) => f(e));
  const pack = p.boxes.pack, a1 = p.getA('item-1'), a2 = p.getA('item-2');
  const mouse = (t) => ({ target: t, pointerType: 'mouse', relatedTarget: null });
  // desktop hover shows, leaving hides, the native title is dropped so the browser does not draw a second tooltip
  fire(pack.listeners, 'pointerover', mouse(a1));
  assert.strictEqual(p.h.state().key, 'item-1'); assert.strictEqual(p.h.state().node.hidden, false);
  assert.strictEqual(a1.attrs.title, undefined); assert.strictEqual(a1.attrs['aria-describedby'], 'inv-tip');
  fire(pack.listeners, 'pointerout', mouse(a1));
  assert.strictEqual(p.h.state().key, ''); assert.strictEqual(p.h.state().node.hidden, true); assert.strictEqual(a1.attrs['aria-describedby'], undefined);
  // touch: a tap opens, a second tap on the same item closes, a tap elsewhere closes
  const touch = (t) => ({ target: t, pointerType: 'touch' });
  fire(p.docL, 'pointerdown', touch(a1)); fire(pack.listeners, 'click', touch(a1));
  assert.strictEqual(p.h.state().mode, 'tap');
  fire(p.docL, 'pointerdown', touch(a2)); // pointerdown on another item closes first, the click then opens that one
  assert.strictEqual(p.h.state().key, ''); fire(pack.listeners, 'click', touch(a2)); assert.strictEqual(p.h.state().key, 'item-2');
  fire(p.docL, 'pointerdown', touch(a2)); fire(pack.listeners, 'click', touch(a2)); assert.strictEqual(p.h.state().key, '', 'second tap closes');
  fire(p.docL, 'pointerdown', touch(a1)); fire(pack.listeners, 'click', touch(a1));
  fire(p.docL, 'pointerdown', touch(p.body)); assert.strictEqual(p.h.state().key, '', 'tap away closes');
  // hover does not interfere while a tap tip is open
  fire(p.docL, 'pointerdown', touch(a1)); fire(pack.listeners, 'click', touch(a1));
  fire(pack.listeners, 'pointerover', mouse(a2)); assert.strictEqual(p.h.state().key, 'item-1');
  p.h.hide();
  // keyboard: focus shows only after a key press, blur hides, Escape closes
  fire(pack.listeners, 'focusin', { target: a2 }); assert.strictEqual(p.h.state().key, '', 'focus after a mouse press does not open');
  fire(p.docL, 'keydown', { key: 'Tab' }); fire(pack.listeners, 'focusin', { target: a2 }); assert.strictEqual(p.h.state().key, 'item-2');
  fire(p.docL, 'keydown', { key: 'Escape' }); assert.strictEqual(p.h.state().key, '');
  fire(pack.listeners, 'focusin', { target: a2 }); fire(pack.listeners, 'focusout', { target: a2 }); assert.strictEqual(p.h.state().key, '');
  // a poll with the same data leaves the tooltip's DOM alone; changed text refills it; a vanished item closes it
  fire(pack.listeners, 'pointerover', mouse(a1));
  const first = p.h.state().node.children[0];
  p.h.onView(JSON.parse(JSON.stringify(view))); assert.strictEqual(p.h.state().node.children[0], first, 'unchanged poll rebuilt the tip');
  const v2 = JSON.parse(JSON.stringify(view)); v2.backpack[0].tip[1] = 'Defense: 11';
  p.h.onView(v2); assert.notStrictEqual(p.h.state().node.children[0], first); assert.ok(ser(p.h.state().node, 0).includes('Defense: 11'));
  const v3 = JSON.parse(JSON.stringify(view)); v3.backpack.shift(); p.h.onView(v3); assert.strictEqual(p.h.state().key, '');
  // the anchor is replaced by a redraw (same key): the tip follows it
  fire(pack.listeners, 'pointerover', mouse(a2)); p.gone['item-2'] = true; p.h.onView(view); // anchor missing: closes, no throw
  assert.strictEqual(p.h.state().key, '');
  // scroll and resize reposition only while open
  fire(p.winL, 'scroll', {}); fire(p.winL, 'resize', {});
}
if (bad) process.exit(1);
console.log('ok');
