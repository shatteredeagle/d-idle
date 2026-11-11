// DOM golden of the journey panel (docs/campaign.md section 6): the path states, the Up next block and the buttons for
// a Paladin and a Sorceress view, a tap on a chapter, and the POSTs the buttons send. UPDATE=1 rewrites journey_dom.txt.
// Usage: node journey_dom.js path/to/panel.js (the Go test writes the view-<class>.tmp.json inputs).
const assert = require('assert'), fs = require('fs'), path = require('path');
const { load, shellCtx } = require('../panel_harness.js');
const P = load(process.argv[2]);
const posts = [];
let out = '';
const txt = (e) => (e.hidden ? '(hidden)' : JSON.stringify(e.textContent));
for (const cls of ['paladin', 'sorceress']) {
  const view = JSON.parse(fs.readFileSync(path.join(__dirname, 'view-' + cls + '.tmp.json'), 'utf8'));
  const { root, c } = shellCtx({}, {}, []);
  const mk = c.el;
  c.ui = {
    btn: (t, o) => { const b = mk('button', 'd-btn', t); b.dataset = Object.assign({}, o); return b; },
    meter: () => { const m = mk('div', 'd-meter'); return { root: m, set(p, t, l) { m.textContent = p + '% ' + t; m.dataset.level = l || ''; } }; },
    clear(n) { n.children.length = 0; },
  };
  c.post = (p, b) => { posts.push(p + ' ' + JSON.stringify(b)); return Promise.resolve({ ok: true }); };
  const inst = P.mount(root, c);
  inst.update(view);
  const [pathEl, up, det] = c.body.children;
  out += cls + ': meta ' + JSON.stringify(c.meta.textContent) + '\n';
  out += '  path ' + pathEl.children.map((li) => li.dataset.key + '=' + li.dataset.state + (li.dataset.band ? '/' + li.dataset.band : '')).join(' ') + '\n';
  const [, upTitle, upWhy, bar, upBtns] = up.children;
  out += '  up next ' + txt(upTitle) + ' why ' + txt(upWhy) + ' meter ' + txt(bar) + '\n';
  out += '  buttons ' + upBtns.children.map((b) => b.textContent).join(', ') + '\n';
  out += '  detail ' + (det.hidden ? '(hidden)' : 'shown') + '\n';
  pathEl.children[pathEl.children.length - 1].children[0].fire('click'); // tap Andariel
  const [, , det2] = c.body.children;
  const [dt, dx, db, dbtns] = det2.children;
  out += '  tap andariel: ' + txt(dt) + ' | ' + txt(db) + ' | ' + dbtns.children.map((b) => b.textContent).join(', ') + '\n';
  dbtns.children[0].fire('click');
  upBtns.children[0].fire('click');
}
setImmediate(() => {
  out += 'posts ' + posts.join(' ; ') + '\n';
  const golden = path.join(__dirname, 'journey_dom.txt');
  if (process.env.UPDATE) fs.writeFileSync(golden, out);
  assert.strictEqual(out, fs.readFileSync(golden, 'utf8'));
  console.log('ok');
});
