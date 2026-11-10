// DOM golden of the gear panel's Farm gold part (panel version 2, docs/goldfarm.md): the button text, its hidden state
// and the status line for the reply shapes of GET /api/repair, plus the POST the click sends. UPDATE=1 rewrites
// testdata/gear_farm_dom.txt. Usage: node gear_farm_dom.js path/to/panel.js
const assert = require('assert'), fs = require('fs'), path = require('path');
const { load, shellCtx } = require('./panel_harness.js');
const P = load(process.argv[2]);
const gear = [{ slot: 'torso', name: 'Quilted Armor', dur: 3, max: 20, pct: 15, warn: true }];
const cases = {
  'no farm field (old bot)': { gear },
  'off': { gear, farm: { schema: 1, on: false, text: 'Not farming gold.' } },
  'on, short of the target': { gear, farm: { schema: 1, on: true, area: 'Blood Moor', have: 40, want: 190, text: 'Farming gold on your order: 40 of 190 in Blood Moor.' } },
  'on, target reached': { gear, farm: { schema: 1, on: true, area: 'Blood Moor', have: 200, want: 190, reached: true, text: 'Farming gold on your order: 200 of 190 reached, still farming in Blood Moor until you stop.' } },
  'ended by itself': { gear, farm: { schema: 1, on: false, note: 'level 1 is below the first safe zone', text: 'Farming gold stopped: level 1 is below the first safe zone' } },
  'no gear, farm on': { gear: [], farm: { schema: 1, on: true, text: 'Farming gold on your order: choosing a zone.' } },
};
const posts = [];
let out = '';
for (const [name, reply] of Object.entries(cases)) {
  const { root, c } = shellCtx({}, {}, []);
  const mk = c.el;
  c.ui = { btn: (t, o) => { const b = mk('button', 'd-btn', t); b.dataset = Object.assign({}, o); return b; }, chip: (t) => mk('span', 'd-chip', t), clear(n) { n.children.length = 0; } };
  c.post = (p, b) => { posts.push(p + ' ' + JSON.stringify(b)); return Promise.resolve({ ok: true }); };
  const inst = P.mount(root, c);
  inst.update(reply);
  const [farmBtn, repairBtn] = c.actions.children;
  const line = c.body.children.find((k) => k.dataset && k.dataset.role === 'farmgold');
  out += `${name}: panel ${c.hidden ? 'hidden' : 'shown'}; button ${farmBtn.hidden ? '(hidden)' : JSON.stringify(farmBtn.textContent)}; repair ${JSON.stringify(repairBtn.textContent)}; line ${line.hidden ? '(hidden)' : JSON.stringify(line.textContent)}\n`;
  if (name === 'on, short of the target') { farmBtn.fire('click'); }
  if (name === 'off') { farmBtn.fire('click'); }
}
setImmediate(() => {
  assert.deepStrictEqual(posts, ['/api/farmgold {"on":true}', '/api/farmgold {"on":false}'], 'a click posts the opposite of the shown state');
  const golden = path.join(__dirname, 'gear_farm_dom.txt');
  if (process.env.UPDATE) fs.writeFileSync(golden, out);
  assert.strictEqual(out, fs.readFileSync(golden, 'utf8'));
  console.log('ok');
});
