// DOM goldens of web/inventory.js (docs/paperdoll-art-build.md): paints the fixture view into a stub DOM and compares the
// serialised paper doll, backpack, belt and message with testdata/inventory_dom_<state>.txt (UPDATE=1 rewrites them).
// States: ready with art off (identical to the page before the art layer existed), ready with art on (fixture URLs only,
// no image is fetched), empty, error, and an image that fails to load. Usage: node inventory_dom.js path/to/inventory.js
const fs = require('fs'), vm = require('vm'), assert = require('assert'), path = require('path');
const dir = __dirname;
const ids = ['equip', 'pack', 'belt', 'stash', 'packmeta', 'stashmeta', 'equipmeta', 'note', 'detail', 'queue', 'msg'];

function mk(tag) {
  const e = { tag, className: '', dataset: {}, children: [], textContent: '', attrs: {}, listeners: {}, style: {}, hidden: false, title: '', type: '' };
  e.classList = { add(c) { e.className = (e.className + ' ' + c).trim(); }, contains(c) { return e.className.split(/\s+/).includes(c); }, remove(c) { e.className = e.className.split(/\s+/).filter((x) => x && x !== c).join(' '); } };
  Object.defineProperty(e, 'firstChild', { get() { return e.children[0] || null; } });
  e.appendChild = (k) => { e.children.push(k); return k; };
  e.removeChild = (k) => { e.children = e.children.filter((x) => x !== k); return k; };
  e.setAttribute = (k, v) => { e.attrs[k] = String(v); };
  e.addEventListener = (t, f) => { (e.listeners[t] = e.listeners[t] || []).push(f); };
  e.fire = (t) => (e.listeners[t] || []).forEach((f) => f({}));
  e.focus = () => {};
  return e;
}
function ser(e, d) {
  const attrs = Object.keys(e.attrs).sort().map((k) => k + '=' + e.attrs[k]);
  const data = Object.keys(e.dataset).sort().map((k) => 'data-' + k + '=' + e.dataset[k]);
  const sty = Object.keys(e.style).sort().map((k) => k + ':' + e.style[k]);
  const cls = e.className.split(/\s+/).filter(Boolean).sort().join('.');
  let out = '  '.repeat(d) + e.tag + (cls ? '.' + cls : '') + ' [' + attrs.concat(data, sty).join(' ') + ']' + (e.textContent ? ' ' + JSON.stringify(e.textContent) : '') + (e.title ? ' title=' + JSON.stringify(e.title) : '') + (e.hidden ? ' hidden' : '') + '\n';
  e.children.forEach((c) => { out += ser(c, d + 1); });
  return out;
}

function page(src) {
  const els = {};
  ids.forEach((i) => { els[i] = mk('div'); });
  let h = null;
  const document = { getElementById: (i) => els[i], createElement: mk, activeElement: null,
    // just enough of the selector grammar the page uses: "a.b.c, .d.e" and [data-key="k"]
    querySelectorAll(sel) {
      const out = [];
      const walk = (e) => { if (sel.split(',').some((part) => matches(e, part.trim()))) out.push(e); e.children.forEach(walk); };
      ids.forEach((i) => walk(els[i]));
      return out;
    },
    querySelector(sel) { return document.querySelectorAll(sel)[0] || null; } };
  const matches = (e, part) => {
    const k = /^\[data-key="(.*)"\]$/.exec(part);
    if (k) return e.dataset.key === k[1];
    const cls = part.split('.').slice(1);
    return cls.length > 0 && cls.every((c) => e.classList.contains(c));
  };
  const ctx = { document, console, JSON, Math, Object, Array, String, Promise, fetch: () => Promise.reject(new Error('no network in the stub')), setInterval() {}, location: { search: '' }, Image: function () {}, window: { __invTest: (x) => { h = x; } } };
  ctx.window.window = ctx.window;
  vm.runInNewContext(src.replace("typeof window !== 'undefined' && window.__invTest", 'window.__invTest'), ctx);
  assert.ok(h && h.paint, 'test hook lacks paint');
  return { els, h };
}
const view = JSON.parse(fs.readFileSync(path.join(dir, 'inventory-art-view.json'), 'utf8'));
const dump = (els) => ['equipmeta', 'equip', 'pack', 'belt', 'msg'].map((i) => '== ' + i + '\n' + ser(els[i], 0)).join('');

const src = fs.readFileSync(process.argv[2], 'utf8');
const states = {
  ready_off: (p) => { p.h.paint(view); },
  ready_on: (p) => { p.h.setArt(true); p.h.paint(view); },
  empty: (p) => { p.h.paint({ schema: 1, equipment: {}, backpack: [], stash: [], belt: [], pins: [], queue: [], recent: [] }); },
  empty_on: (p) => { p.h.setArt(true); p.h.paint({ schema: 1, equipment: {}, backpack: [], stash: [], belt: [], pins: [], queue: [], recent: [] }); },
  error: (p) => { p.h.say && p.h.say('inventory unavailable'); },
  image_fails: (p) => {
    p.h.setArt(true); p.h.paint(view);
    // the weapon icon and the first pack icon fail to load: those slots revert to their drawn look
    const slot = p.els.equip.children.find((c) => c.dataset.slot === 'rhand');
    slot.children.filter((c) => c.tag === 'img').forEach((c) => c.fire('error'));
    p.els.pack.children.filter((c) => c.dataset.key === 'item-10')[0].children.filter((c) => c.tag === 'img').forEach((c) => c.fire('error'));
  },
};
let bad = 0;
// a click only moves the selection: the detail card follows and the grids are NOT rebuilt (the page skips an unchanged view)
{
  const p = page(src);
  if (p.h.select) {
    p.h.paint(view);
    const firstCell = p.els.pack.children[0], items = p.els.pack.children.filter((c) => c.dataset.key && c.dataset.key.startsWith('item-'));
    items[0].fire('click');
    assert.ok(items[0].classList.contains('sel'), 'clicked item is highlighted');
    assert.ok(p.els.detail.children.length > 0 && p.els.detail.children[0].tag === 'h3', 'detail card shows the item');
    const first = p.els.detail.children[0].textContent;
    items[1].fire('click');
    assert.ok(!items[0].classList.contains('sel') && items[1].classList.contains('sel'), 'selection moved');
    assert.notStrictEqual(p.els.detail.children[0].textContent, first, 'detail card changed');
    assert.strictEqual(p.els.pack.children[0], firstCell, 'grid was rebuilt by a click');
  }
}
for (const [name, run] of Object.entries(states)) {
  if (name === 'error' && !page(src).h.say) continue; // old pages have no say() hook: the error state only exists once the hook does
  const p = page(src);
  run(p);
  const got = dump(p.els), file = path.join(dir, 'inventory_dom_' + name + '.nopin.txt');
  if (process.env.UPDATE) { fs.writeFileSync(file, got); continue; }
  if (!fs.existsSync(file)) { console.error('missing golden ' + file); bad++; continue; }
  if (fs.readFileSync(file, 'utf8') !== got) { console.error('DOM differs from ' + file + '\n' + got); bad++; }
}
if (bad) process.exit(1);
console.log('ok');
