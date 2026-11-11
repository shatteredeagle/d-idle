// DOM golden of the inventory page's repair card (docs/repair.md, the repair trip): the card for every reply of the
// repair controller (internal/repair/testdata/view-*.json, both classes) and an older bot's reply. UPDATE=1 rewrites
// testdata/inventory_repair_dom.txt. Usage: node inventory_repair_dom.js path/to/inventory.js
const fs = require('fs'), vm = require('vm'), assert = require('assert'), path = require('path');
function mk(tag) {
  const e = { tag, className: '', dataset: {}, children: [], textContent: '', attrs: {}, listeners: {}, style: {}, hidden: false, title: '', type: '', disabled: false };
  e.classList = { add(c) { e.className = (e.className + ' ' + c).trim(); }, contains(c) { return e.className.split(/\s+/).includes(c); }, remove() {} };
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
  const data = Object.keys(e.dataset).sort().map((k) => 'data-' + k + '=' + e.dataset[k]);
  const cls = e.className.split(/\s+/).filter(Boolean).sort().join('.');
  let out = '  '.repeat(d) + e.tag + (cls ? '.' + cls : '') + ' [' + data.join(' ') + ']' + (e.textContent ? ' ' + JSON.stringify(e.textContent) : '') + (e.title ? ' title=' + JSON.stringify(e.title) : '') + (e.disabled ? ' disabled' : '') + (e.hidden ? ' hidden' : '') + '\n';
  e.children.forEach((c) => { out += ser(c, d + 1); });
  return out;
}
const src = fs.readFileSync(process.argv[2], 'utf8');
const dir = path.join(__dirname, '..', '..', 'repair', 'testdata');
const files = fs.readdirSync(dir).filter((f) => /^view-.*\.json$/.test(f)).sort();
assert.ok(files.length >= 26);
const cases = files.map((f) => [f.replace(/^view-|\.json$/g, ''), JSON.parse(fs.readFileSync(path.join(dir, f), 'utf8'))]);
const old = JSON.parse(JSON.stringify(cases.find(([n]) => n === 'queued-sorceress')[1])); delete old.trip; old.schema = 1;
cases.push(['old bot, requested', old]);
let out = '';
for (const [name, view] of cases) {
  const els = { repair: mk('div') };
  let h = null;
  const document = { getElementById: (i) => els[i] || mk('div'), createElement: mk, activeElement: null, querySelectorAll: () => [], querySelector: () => null };
  const ctx = { document, console, JSON, Math, Object, Array, String, Promise, fetch: () => Promise.reject(new Error('no network')), setInterval() {}, location: { search: '' }, Image: function () {}, window: { __invTest: (x) => { h = x; } } };
  ctx.window.window = ctx.window;
  vm.runInNewContext(src.replace("typeof window !== 'undefined' && window.__invTest", 'window.__invTest'), ctx);
  assert.ok(h && h.drawRepair, 'test hook lacks drawRepair');
  h.drawRepair(view);
  out += '== ' + name + '\n' + ser(els.repair, 0);
}
const golden = path.join(__dirname, 'inventory_repair_dom.txt');
if (process.env.UPDATE) fs.writeFileSync(golden, out);
assert.strictEqual(out, fs.readFileSync(golden, 'utf8'));
console.log('ok');
