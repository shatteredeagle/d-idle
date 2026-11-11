// DOM golden of the gear panel's repair part (panel version 3, docs/repair.md): the button, its state and the line for
// every reply the repair controller can send, read from the Go goldens internal/repair/testdata/view-*.json (both
// classes), plus an old bot's reply without `trip`, plus the POST each click sends. UPDATE=1 rewrites
// testdata/gear_repairtrip_dom.txt. Usage: node gear_repairtrip_dom.js path/to/panel.js
const assert = require('assert'), fs = require('fs'), path = require('path');
const { load, shellCtx } = require('./panel_harness.js');
const P = load(process.argv[2]);
const dir = path.join(__dirname, '..', '..', 'repair', 'testdata');
const files = fs.readdirSync(dir).filter((f) => /^view-.*\.json$/.test(f)).sort();
assert.ok(files.length >= 26, 'the controller goldens are missing: ' + files.length);
const cases = files.map((f) => [f.replace(/^view-|\.json$/g, ''), JSON.parse(fs.readFileSync(path.join(dir, f), 'utf8'))]);
// an older bot: schema 1, no trip
const old = JSON.parse(JSON.stringify(cases.find(([n]) => n === 'queued-paladin')[1]));
delete old.trip; old.schema = 1;
cases.push(['old bot, requested', old]);
const oldIdle = JSON.parse(JSON.stringify(old)); delete oldIdle.request;
cases.push(['old bot, idle', oldIdle]);

const posts = [];
let out = '';
for (const [name, reply] of cases) {
  const { root, c } = shellCtx({}, {}, []);
  const mk = c.el;
  c.ui = { btn: (t, o) => { const b = mk('button', 'd-btn', t); b.dataset = Object.assign({}, o); return b; }, chip: (t) => mk('span', 'd-chip', t), clear(n) { n.children.length = 0; } };
  c.post = (p, b) => { posts.push(name + ' ' + p + ' ' + JSON.stringify(b)); return Promise.resolve({ ok: true }); };
  const inst = P.mount(root, c);
  inst.update(reply);
  const [farmBtn, act] = c.actions.children;
  const note = c.body.children[1];
  out += `${name}: button ${JSON.stringify(act.textContent)} ${act.disabled ? 'off' : 'on'} state=${act.dataset.state}${act.title ? ' why=' + JSON.stringify(act.title) : ''}; line ${note.hidden ? '(hidden)' : JSON.stringify(note.textContent)} state=${note.dataset.state}\n`;
  act.fire('click');
}
setImmediate(() => {
  const expect = [];
  for (const [name, r] of cases) {
    if (r.trip) { if (r.trip.button.enabled) expect.push(name + ' ' + (r.trip.button.action === 'cancel' ? '/api/repair/cancel' : '/api/repair/request') + ' {}'); }
    else expect.push(name + ' ' + (r.request && r.request.state === 'requested' ? '/api/repair/cancel' : '/api/repair/request') + ' {}');
  }
  assert.deepStrictEqual(posts, expect, 'a disabled button posts nothing; every other click posts the route its label names');
  const golden = path.join(__dirname, 'gear_repairtrip_dom.txt');
  if (process.env.UPDATE) fs.writeFileSync(golden, out);
  assert.strictEqual(out, fs.readFileSync(golden, 'utf8'));
  console.log('ok');
});
