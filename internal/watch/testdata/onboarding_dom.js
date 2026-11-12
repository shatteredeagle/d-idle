// DOM golden of the onboarding panel (docs/onboarding.md): the welcome card, the checklist states, the Help entries and
// the reconnect banners against a stub DOM, with the real texts of data/onboarding/en.json. UPDATE=1 rewrites
// testdata/onboarding_dom.txt (also needed after a wording change). Usage: node onboarding_dom.js path/to/panel.js
const assert = require('assert'), fs = require('fs'), path = require('path');
const { load, shellCtx, mk } = require('./panel_harness.js');
const texts = JSON.parse(fs.readFileSync(path.join(__dirname, '../../../data/onboarding/en.json'), 'utf8'));

let now = 1_000_000, listeners = {}, reconnects = 0, videoState = 'live';
const win = {
  addEventListener(t, f) { (listeners[t] = listeners[t] || []).push(f); },
  d2video: { state: () => videoState, reconnect: () => { reconnects++; } },
  Date: { now: () => now },
};
const P = load(process.argv[2], { Date: win.Date, addEventListener: win.addEventListener, d2video: win.d2video });

function reply(over) {
  const v = {
    schema: 'watch.onboarding/1', boot: 'b1', hero: 'Alpha', showWelcome: true, showList: true, collapsed: false,
    steps: texts.steps.map((s) => Object.assign({ done: false }, s)),
    away: { absent: false, seconds: 0 },
    texts: { welcome: texts.welcome, help: texts.help, checklist: texts.checklist, steps: texts.steps, reconnect: texts.reconnect, art: texts.art },
  };
  Object.assign(v, over || {});
  v.done = v.steps.filter((s) => s.done).length;
  v.allDone = v.done === v.steps.length;
  if (v.allDone && !('showList' in (over || {}))) v.showList = false;
  return v;
}
const doneSteps = (...ids) => texts.steps.map((s) => Object.assign({ done: ids.includes(s.id) }, s));

function fresh() {
  const timers = [];
  const { root, c } = shellCtx({}, {}, timers);
  const mkEl = c.el;
  c.el = (tag, cls, text) => {
    const e = mkEl(tag, cls, text);
    e.replaceChildren = (...k) => { e.children = k; };
    e.classList = { add(n) { e.className = (e.className + ' ' + n).trim(); } };
    return e;
  };
  c.posts = [];
  c.post = (p, b) => { c.posts.push(p + ' ' + JSON.stringify(b)); return Promise.resolve({ ok: true }); };
  const inst = P.mount(root, c);
  return { c, inst, timers };
}

function dump(n, d, out) {
  if (n.hidden) { out.push('  '.repeat(d) + n.tag + '.' + (n.className || '-') + ' (hidden)'); return; }
  const key = n.dataset && n.dataset.key ? ' [' + n.dataset.key + ']' : '';
  const sev = n.dataset && n.dataset.sev ? ' sev=' + n.dataset.sev : '';
  const dn = n.dataset && n.dataset.done ? ' done=' + n.dataset.done : '';
  const href = n.href ? ' href=' + n.href : '';
  const txt = n.textContent ? ' ' + JSON.stringify(n.textContent) : '';
  out.push('  '.repeat(d) + n.tag + '.' + (n.className || '-') + key + sev + dn + href + txt);
  n.children.forEach((k) => dump(k, d + 1, out));
}
function shot(c, label) {
  const out = [`== ${label}: panel ${c.hidden ? 'hidden' : 'shown'}; meta ${JSON.stringify(c.meta.textContent)}`];
  if (!c.hidden) c.body.children.forEach((k) => dump(k, 1, out));
  return out.join('\n') + '\n';
}
const find = (n, key) => { if (n.dataset && n.dataset.key === key) return n; for (const k of n.children) { const r = find(k, key); if (r) return r; } return null; };

let out = '';
const checks = [];

// 1 fresh hero: welcome + checklist 0 of 5
let { c, inst } = fresh();
inst.update(reply());
out += shot(c, 'fresh hero');
// the welcome buttons: Journey ticks its step, dismiss hides the card and posts
find(c.body, 'welcome-journey').fire('click');
checks.push([c.posts.slice(), ['/api/onboarding {"action":"tick","step":"journey"}']]);
c.posts.length = 0;
find(c.body, 'welcome-dismiss').fire('click');
checks.push([c.posts.slice(), ['/api/onboarding {"action":"dismiss_welcome"}']]);
out += shot(c, 'after dismiss and the journey click');
c.posts.length = 0;
find(c.body, 'tick-junk').fire('click');
checks.push([c.posts.slice(), ['/api/onboarding {"action":"tick","step":"junk"}']]);
find(c.body, 'check-fold').fire('click');
out += shot(c, 'checklist folded');
find(c.body, 'check-fold').fire('click');

// 2 two steps ticked by the bot, welcome gone
({ c, inst } = fresh());
inst.update(reply({ showWelcome: false, steps: doneSteps('fight', 'equip') }));
out += shot(c, 'two steps done, welcome dismissed');

// 3 everything done: the panel hides itself; Help brings the entries back
({ c, inst } = fresh());
inst.update(reply({ showWelcome: false, steps: doneSteps('equip', 'policy', 'fight', 'journey', 'junk') }));
out += shot(c, 'all done');
listeners.d2help.forEach((f) => f());
out += shot(c, 'all done, Help open');
c.posts.length = 0;
find(c.body, 'help-list').fire('click');
checks.push([c.posts.slice(), ['/api/onboarding {"action":"show_list"}']]);
out += shot(c, 'all done, list brought back');

// 4 the route is off: nothing is shown
({ c, inst } = fresh());
inst.update(null, { status: 404, fails: 1 });
out += shot(c, 'route off (404)');

// 5 connection loss after a good reply: one blip is quiet, two are a banner, recovery clears it
({ c, inst } = fresh());
inst.update(reply({ showWelcome: false, steps: doneSteps('fight') }));
inst.update(null, { status: 0, fails: 1 });
out += shot(c, 'one failed poll');
inst.update(null, { status: 0, fails: 2 });
out += shot(c, 'two failed polls');
inst.update(null, { status: 503, fails: 3 });
out += shot(c, 'bot starting (503)');
inst.update(reply({ showWelcome: false, steps: doneSteps('fight') }));
out += shot(c, 'recovered');

// 5b the very first poll fails twice (bot down when the page opens): the built-in line, no empty banner
({ c, inst } = fresh());
inst.update(null, { status: 0, fails: 1 });
out += shot(c, 'first load, one failure');
inst.update(null, { status: 0, fails: 2 });
out += shot(c, 'first load, two failures');

// 6 the bot restarted (boot id changed), then the banner times out
({ c, inst } = fresh());
inst.update(reply({ showWelcome: false, steps: doneSteps('fight') }));
inst.update(reply({ showWelcome: false, boot: 'b2', steps: doneSteps('fight') }));
out += shot(c, 'bot restarted');
now += 21000;
inst.update(reply({ showWelcome: false, boot: 'b2', steps: doneSteps('fight') }));
out += shot(c, 'restart banner gone');

// 7 the player was away 2 h 14 m: banner with the Return screen link, dismissible
({ c, inst } = fresh());
inst.update(reply({ showWelcome: false, steps: doneSteps('fight'), away: { absent: true, seconds: 8040 } }));
out += shot(c, 'away 2h 14m');
inst.update(reply({ showWelcome: false, steps: doneSteps('fight'), away: { absent: false, seconds: 0 } }));
out += shot(c, 'away banner stays after the look is counted');
find(c.body, 'banner-away-x').fire('click');
out += shot(c, 'away dismissed');

// 8 the video stream is down: banner with a Retry that calls the page hook
({ c, inst } = fresh());
inst.update(reply({ showWelcome: false, steps: doneSteps('fight') }));
videoState = 'reconnecting';
inst.update(reply({ showWelcome: false, steps: doneSteps('fight') }));
out += shot(c, 'stream reconnecting');
find(c.body, 'banner-stream').fire('click');
checks.push([[reconnects], [1]]);
videoState = 'live';
inst.update(reply({ showWelcome: false, steps: doneSteps('fight') }));
out += shot(c, 'stream back');

// 9 art is being prepared: the note sits in the guide
({ c, inst } = fresh());
inst.update(reply({ showWelcome: false, art: { state: 'extracting', done: 120, total: 349 } }));
out += shot(c, 'preparing art');

// pure helpers
assert.strictEqual(P.fmtDuration(8040), '2h 14m');
assert.strictEqual(P.fmtDuration(30), 'under a minute');
assert.strictEqual(P.fmtDuration(600), '10m');
assert.strictEqual(P.fmtDuration(90000), '1d 1h');
assert.strictEqual(P.fill('a {x} {y}', { x: 1 }), 'a 1 {y}');

setImmediate(() => {
  checks.forEach(([got, want]) => assert.deepStrictEqual(got, want));
  const golden = path.join(__dirname, 'onboarding_dom.txt');
  if (process.env.UPDATE) fs.writeFileSync(golden, out);
  assert.strictEqual(out, fs.readFileSync(golden, 'utf8'));
  console.log('ok');
});
