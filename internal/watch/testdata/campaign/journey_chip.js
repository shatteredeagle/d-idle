// The journey chip line for the golden views (docs/campaign.md section 6). Usage: node journey_chip.js path/to/panel.js
const assert = require('assert'), fs = require('fs'), path = require('path');
const { load, shellCtx } = require('../panel_harness.js');
const P = load(process.argv[2]);
const got = {};
for (const f of ['view-paladin-clvl12.json', 'view-sorceress-clvl7.json']) {
  const v = JSON.parse(fs.readFileSync(path.join(__dirname, f), 'utf8'));
  const { root, c } = shellCtx({}, {}, []);
  P.mount(root, c).update(v);
  const a = root.children[0];
  got[f] = (c.hidden ? 'hidden' : a.textContent) + ' [' + (a.dataset.tone || '') + ']';
}
assert.deepStrictEqual(got, {
  'view-paladin-clvl12.json': 'Find Deckard Cain: comfortable, next: The Countess [good]',
  'view-sorceress-clvl7.json': 'Blood Raven: too early, next: Claim the Rogue [bad]',
});
const { c: c2, root: r2 } = shellCtx({}, {}, []);
P.mount(r2, c2).update(null);
assert.strictEqual(c2.hidden, true, 'no view: hidden');
console.log('ok');
