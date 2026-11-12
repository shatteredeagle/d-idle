// DOM goldens and behaviour of web/inventory-sets.js (docs/gearsets.md): the panel for a Paladin and a Sorceress view, locked,
// with a preview open (worn, pack, stash, missing, blocked and the steps), rename and delete confirm, save with and without
// a mercenary, what each button posts, and that a typed name survives the poll. UPDATE=1 rewrites the goldens.
// Usage: node sets_dom.js path/to/inventory-sets.js
const fs = require('fs'), vm = require('vm'), assert = require('assert'), path = require('path');
const dir = __dirname;
const views = JSON.parse(fs.readFileSync(path.join(dir, 'view.json'), 'utf8'));
const src = fs.readFileSync(process.argv[2], 'utf8');

function mk(tag) {
  const e = { tag, className: '', dataset: {}, children: [], textContent: '', attrs: {}, listeners: {}, hidden: false, id: '', value: '' };
  e.appendChild = (k) => { k.parent = e; e.children.push(k); return k; };
  Object.defineProperty(e, 'firstChild', { get() { return e.children[0] || null; } });
  e.removeChild = (k) => { e.children = e.children.filter((x) => x !== k); return k; };
  e.setAttribute = (k, v) => { e.attrs[k] = String(v); };
  e.addEventListener = (t, f) => { (e.listeners[t] = e.listeners[t] || []).push(f); };
  return e;
}
function ser(e, d) {
  const attrs = Object.keys(e.attrs).sort().map((k) => k + '=' + e.attrs[k]);
  const data = Object.keys(e.dataset).sort().map((k) => 'data-' + k + '=' + e.dataset[k]);
  const cls = e.className.split(/\s+/).filter(Boolean).sort().join('.');
  let out = '  '.repeat(d) + e.tag + (cls ? '.' + cls : '') + (e.id ? '#' + e.id : '') + ' [' + attrs.concat(data).join(' ') + ']' + (e.textContent ? ' ' + JSON.stringify(e.textContent) : '') + (e.value ? ' value=' + JSON.stringify(e.value) : '') + (e.hidden ? ' hidden' : '') + '\n';
  e.children.forEach((c) => { out += ser(c, d + 1); });
  return out;
}
function page() {
  const host = mk('div'), L = {}, posts = [], gets = [];
  host.id = 'gearsets';
  const find = (n, id) => { if (n.id === id) return n; for (const k of n.children) { const r = find(k, id); if (r) return r; } return null; };
  const document = { createElement: mk, activeElement: null, getElementById: (i) => (i === 'gearsets' ? host : find(host, i)),
    addEventListener: (t, f) => { (L[t] = L[t] || []).push(f); } };
  let nextPreview = null;
  const fetch = (u, o) => {
    if (/setpreview/.test(u)) { gets.push(u); return Promise.resolve({ ok: !!nextPreview, status: nextPreview ? 200 : 409, json: () => Promise.resolve(nextPreview || { code: 'set', error: 'that set does not exist any more' }) }); }
    posts.push({ u, body: JSON.parse(o.body) });
    return Promise.resolve({ ok: true, status: 200, json: () => Promise.resolve({ ok: true }) });
  };
  let refreshes = 0;
  const window = { __setsTest: (h) => { L.h = h; }, d2InvActions: { base: () => Promise.resolve('http://x'), refresh: () => { refreshes++; return Promise.resolve(); } } };
  vm.runInNewContext(src, { document, window, JSON, Math, Object, Array, String, Number, RegExp, Promise, console, fetch, encodeURIComponent });
  return { h: L.h, host, posts, gets, document, setPreview: (p) => { nextPreview = p; }, refreshes: () => refreshes, find: (id) => find(host, id) };
}
let bad = 0;
const check = (name, got) => {
  const file = path.join(dir, 'sets_dom_' + name + '.nopin.txt');
  if (process.env.UPDATE) { fs.writeFileSync(file, got); return; }
  if (!fs.existsSync(file)) { console.error('missing golden ' + file); bad++; return; }
  if (fs.readFileSync(file, 'utf8') !== got) { console.error('differs from ' + file + '\n' + got); bad++; }
};
const btn = (host, act) => { let hit = null; const f = (n) => { if (n.dataset && n.dataset.act === act) hit = n; n.children.forEach(f); }; f(host); return hit; };
const tick = () => new Promise((r) => setTimeout(r, 0));

const preview = {
  setId: 'gs-1', name: 'Sword and board', target: 'hero', missing: 1, ready: true,
  slots: [{ slot: 'rhand', name: 'Short Sword', now: 'Maul', status: 'pack', id: 1 }, { slot: 'lhand', name: 'Buckler', status: 'stash', id: 2 },
    { slot: 'head', name: 'Great Helm', status: 'missing', why: 'not found in the backpack, stash or worn: skipped, nothing is substituted' },
    { slot: 'rring', name: 'Ring A', status: 'blocked', why: 'restricted to another class' }, { slot: 'lring', name: 'Ring B', status: 'elsewhere', from: 'rring', id: 9 }],
  steps: [{ op: 'unequip', id: 9, slot: 'rring', name: 'Ring B', note: 'worn in another slot' }, { op: 'take', id: 2, name: 'Buckler', note: 'from the stash, in town' }, { op: 'equip', id: 1, slot: 'rhand', name: 'Short Sword' }],
};

(async () => {
  for (const cls of ['paladin', 'sorceress']) {
    const v = views[cls];
    { const p = page(); p.h.init(); p.h.onView(v); check(cls + '_open', ser(p.host, 0)); }
    { const p = page(); p.h.init(); const l = JSON.parse(JSON.stringify(v)); l.sets = { schema: 1, open: false, why: 'Reach level 15.', max: 8, sets: [] }; p.h.onView(l); check(cls + '_locked', ser(p.host, 0)); }
    { const p = page(); p.h.init(); p.h.onView(v); p.setPreview(preview); btn(p.host, 'apply').listeners.click[0]();
      assert.strictEqual(p.gets.length, 1); assert.ok(/id=gs-1/.test(p.gets[0]));
      await tick(); check(cls + '_preview', ser(p.host, 0));
      // Apply now posts the set id once; a not-ready preview posts nothing and says why
      btn(p.host, 'apply-go').listeners.click[0](); await tick(); await tick();
      assert.strictEqual(JSON.stringify(p.posts), JSON.stringify([{ u: 'http://x/api/inventory/sets/apply', body: { setId: 'gs-1' } }]));
      assert.strictEqual(p.refreshes(), 1);
    }
    { const p = page(); p.h.init(); p.h.onView(v); p.setPreview(Object.assign({}, preview, { ready: false, why: 'the backpack has no free spot for Buckler: make room first' }));
      btn(p.host, 'apply').listeners.click[0](); await tick();
      btn(p.host, 'apply-go').listeners.click[0](); await tick();
      assert.strictEqual(p.posts.length, 0, 'a refused preview posts nothing');
      assert.ok(/no free spot/.test(ser(p.host, 0)));
    }
    { const p = page(); p.h.init(); p.h.onView(v);
      // delete asks twice, rename posts the new name, save posts the name and the target; the merc button needs a merc
      btn(p.host, 'delete').listeners.click[0](); assert.strictEqual(p.posts.length, 0); assert.ok(btn(p.host, 'delete').dataset.confirm !== undefined);
      btn(p.host, 'delete').listeners.click[0](); await tick(); await tick();
      assert.strictEqual(p.posts[0].u, 'http://x/api/inventory/sets/delete');
      p.posts.length = 0;
      btn(p.host, 'rename').listeners.click[0](); const inputs = []; const f = (n) => { if (n.tag === 'input') inputs.push(n); n.children.forEach(f); }; f(p.host);
      inputs.find((i) => i.className.includes('sets-name') && i.id !== 'sets-new').value = 'Tank'; btn(p.host, 'rename-ok').listeners.click[0](); await tick(); await tick();
      assert.strictEqual(JSON.stringify(p.posts[0]), JSON.stringify({ u: 'http://x/api/inventory/sets/rename', body: { setId: 'gs-1', name: 'Tank' } }));
      p.posts.length = 0;
      p.find('sets-new').value = 'Fire'; btn(p.host, 'save-hero').listeners.click[0](); await tick(); await tick();
      assert.strictEqual(JSON.stringify(p.posts[0]), JSON.stringify({ u: 'http://x/api/inventory/sets/save', body: { name: 'Fire', target: 'hero' } }));
      assert.ok(btn(p.host, 'save-merc'), 'a mercenary is hired: his save button shows');
      const nomerc = JSON.parse(JSON.stringify(v)); nomerc.merc = { state: 'none', owned: false, equipment: {} }; p.h.onView(nomerc);
      assert.strictEqual(btn(p.host, 'save-merc'), null);
    }
    { // the poll redraws only on change and keeps a name that is being typed
      const p = page(); p.h.init(); p.h.onView(v);
      const inp = p.find('sets-new'); inp.value = 'Half typed'; p.document.activeElement = inp;
      const before = p.host.children[2]; p.h.onView(JSON.parse(JSON.stringify(v)));
      assert.strictEqual(p.host.children[2], before, 'unchanged view: the DOM is left alone');
      const v2 = JSON.parse(JSON.stringify(v)); v2.sets.sets.push({ id: 'gs-2', name: 'Other', target: 'hero', saved: 1, slots: [{ slot: 'head', name: 'Cap', code: 'helm' }] });
      p.h.onView(v2); assert.strictEqual(p.find('sets-new').value, 'Half typed');
    }
  }
  if (bad) { console.error(bad + ' failure(s)'); process.exit(1); }
  console.log('sets ok');
})().catch((e) => { console.error(e); process.exit(1); });
