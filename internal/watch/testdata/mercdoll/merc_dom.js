// DOM goldens of web/inventory-merc.js (docs/research/merc-gear.md, TASK-287): paints the fixture views
// testdata/inventory_merc_<state>.json into a stub DOM and compares the serialised merc card with
// testdata/mercdoll/merc_dom_<state>.txt (UPDATE=1 rewrites them). States: none, desert, ironwolf, dead, unknown, absent
// (a schema 2 reply without the merc block: the card stays hidden), desert with game art on, and a second paint of an
// unchanged view (the DOM must not be rebuilt). Usage: node merc_dom.js path/to/inventory-merc.js
const fs = require('fs'), vm = require('vm'), assert = require('assert'), path = require('path');
const dir = __dirname, data = path.join(dir, '..');

function mk(tag) {
  const e = { tag, className: '', dataset: {}, children: [], textContent: '', attrs: {}, listeners: {}, style: {}, hidden: false, title: '', open: false };
  e.classList = { add(c) { e.className = (e.className + ' ' + c).trim(); } };
  Object.defineProperty(e, 'firstChild', { get() { return e.children[0] || null; } });
  e.appendChild = (k) => { e.children.push(k); return k; };
  e.removeChild = (k) => { e.children = e.children.filter((x) => x !== k); return k; };
  e.setAttribute = (k, v) => { e.attrs[k] = String(v); };
  e.addEventListener = (t, f) => { (e.listeners[t] = e.listeners[t] || []).push(f); };
  return e;
}
function ser(e, d) {
  const attrs = Object.keys(e.attrs).sort().map((k) => k + '=' + e.attrs[k]);
  const dat = Object.keys(e.dataset).sort().map((k) => 'data-' + k + '=' + e.dataset[k]);
  const cls = e.className.split(/\s+/).filter(Boolean).sort().join('.');
  let out = '  '.repeat(d) + e.tag + (cls ? '.' + cls : '') + ' [' + attrs.concat(dat).join(' ') + ']' + (e.textContent ? ' ' + JSON.stringify(e.textContent) : '') +
    (e.title ? ' title=' + JSON.stringify(e.title) : '') + (e.hidden ? ' hidden' : '') + (e.tag === 'details' ? (e.open ? ' open' : ' closed') : '') + '\n';
  e.children.forEach((c) => { out += ser(c, d + 1); });
  return out;
}
const src = fs.readFileSync(process.argv[2], 'utf8');
function page() {
  const host = mk('div'); host.hidden = true;
  const docL = {};
  const document = { getElementById: (i) => (i === 'merc-doll' ? host : null), createElement: mk, addEventListener: (t, f) => { (docL[t] = docL[t] || []).push(f); } };
  let h = null;
  const window = { __mercTest: (x) => { h = x; } };
  const ctx = { document, window, location: { search: '' }, JSON, Math, Object, Array, String, Number, RegExp, console, Image: undefined };
  vm.runInNewContext(src, ctx);
  return { h, host, docL };
}
let bad = 0;
const check = (name, got) => {
  const file = path.join(dir, 'merc_dom_' + name + '.txt');
  if (process.env.UPDATE) { fs.writeFileSync(file, got); return; }
  if (!fs.existsSync(file)) { console.error('missing golden ' + file); bad++; return; }
  if (fs.readFileSync(file, 'utf8') !== got) { console.error('merc card differs from ' + file + '\n' + got); bad++; }
};
const view = (n) => JSON.parse(fs.readFileSync(path.join(data, 'inventory_merc_' + n + '.json'), 'utf8'));

for (const n of ['none', 'desert', 'ironwolf', 'dead', 'unknown', 'absent']) {
  const p = page(); p.h.render(view(n)); check(n, ser(p.host, 0));
}
{ // game art on: the item icon and the ghosts, no image is fetched in the stub
  const p = page(); p.h.setArt(true); p.h.render(view('desert')); check('desert_art', ser(p.host, 0));
}
{ // an unchanged view does not rebuild the DOM (the fold and focus stay); a changed one does
  const p = page(); p.h.render(view('desert'));
  const first = p.host.children[0], body = first.children[1], kid = body.children[0];
  p.h.render(view('desert'));
  assert.strictEqual(p.host.children[0], first); assert.strictEqual(body.children[0], kid, 'unchanged view rebuilt the card');
  p.h.render(view('dead')); assert.strictEqual(p.host.children[0], first, 'the fold is kept across states');
  p.h.render(view('absent')); assert.ok(p.host.hidden, 'a reply without merc hides the card');
  p.h.render(view('desert')); assert.ok(!p.host.hidden);
}
{ // the event path: the page script listens for d2:inventory
  const d = {}; let seen = null;
  const document = { getElementById: () => { const h = mk('div'); seen = h; return h; }, createElement: mk, addEventListener: (t, f) => { d[t] = f; } };
  vm.runInNewContext(src, { document, window: {}, location: { search: '' }, JSON, Math, Object, Array, String, Number, RegExp, console });
  assert.strictEqual(typeof d['d2:inventory'], 'function');
  d['d2:inventory']({ detail: view('ironwolf') });
  assert.ok(seen && !seen.hidden && seen.children.length === 1);
}
if (bad) process.exit(1);
console.log('ok');
