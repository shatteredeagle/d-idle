// DOM goldens of the cockpit band panels (statusline, needs-count, slim hero-meters) in a stub DOM: each state is rendered
// through the panel's own mount/update and compared with testdata/glance_dom_<panel>_<state>.txt (UPDATE=1 rewrites).
// Usage: node glance_dom.js path/to/web/panels
const assert = require('assert'), fs = require('fs'), path = require('path'), { load, mk } = require('./panel_harness.js');
const dir = process.argv[2];

function node(tag, cls, text) {
  const e = mk(tag);
  e.className = cls || '';
  // like a real element: setting the text drops the children
  let own = '';
  Object.defineProperty(e, 'textContent', { get: () => own, set: (v) => { own = v; e.children.length = 0; }, enumerable: true });
  if (text !== undefined) e.textContent = text;
  e.classList = { add(c) { e.className = (e.className + ' ' + c).trim(); } };
  const app = e.append;
  e.append = (...k) => app(...k.map((x) => (typeof x === 'string' ? { tag: '#text', textContent: x, children: [], dataset: {}, attrs: {} } : x)));
  return e;
}
const ui = {
  meter(kind, label) {
    const m = node('div', 'd-meter'), num = node('span', 'num');
    m.dataset.kind = kind; m.setAttribute('role', 'meter'); m.setAttribute('aria-label', label);
    m.append(node('span', 'lbl', label), node('span', 'track'), num);
    return { root: m, set(pct, text, level) { m.setAttribute('aria-valuenow', String(Math.round(Math.max(0, Math.min(100, pct || 0))))); num.textContent = text; if (level) m.dataset.level = level; else delete m.dataset.level; } };
  },
  chip(text, level) { const c = node('span', 'd-chip', text); if (level) c.dataset.level = level; return c; },
  badge(text, tone) { const b = node('span', 'd-badge', text); if (tone) b.dataset.tone = tone; return b; },
  stat(label) { const s = node('div', 'd-stat'), val = node('span', 'val'), delta = node('span', 'delta'); s.append(node('span', 'lbl', label), val, delta); return { root: s, set(v, d) { val.textContent = v; delta.textContent = d || ''; } }; },
};
function ctxFor(bare) {
  const root = node(bare ? 'div' : 'section'), body = bare ? root : node('div', 'd-card-body'), meta = node('span', 'meta');
  if (!bare) root.append(meta, body);
  const c = { body, meta, el: node, ui, hidden: false, hide(on) { c.hidden = !!on; root.hidden = !!on; }, state() {}, toast() {}, later() {} };
  return { root, c };
}
function dump(e, ind) {
  if (e.tag === '#text') return e.textContent.trim() ? ind + JSON.stringify(e.textContent) + '\n' : '';
  const a = [];
  if (e.className) a.push('class=' + JSON.stringify(e.className));
  for (const k of Object.keys(e.dataset).sort()) a.push('data-' + k + '=' + JSON.stringify(e.dataset[k]));
  for (const k of Object.keys(e.attrs).sort()) a.push(k + '=' + JSON.stringify(e.attrs[k]));
  for (const k of ['href', 'title']) if (e[k]) a.push(k + '=' + JSON.stringify(e[k]));
  if (e.hidden) a.push('hidden');
  let s = ind + '<' + e.tag + (a.length ? ' ' + a.join(' ') : '') + '>' + (e.textContent ? ' ' + JSON.stringify(e.textContent) : '') + '\n';
  for (const k of e.children) s += dump(k, ind + '  ');
  return s;
}


// the sentence as /api/glance gives it, per tone
const base = { tone: 'ok', word: 'Fighting', place: 'Cold Plains', reason: 'you ordered it, 12 min left', margin: 'safe', uptime: '94% of the time making progress', voice: 'Another day, another cow.',
  why: { code: 'fight', now: 'Fighting in Cold Plains, my life is 72%.', change: 'Life under my retreat line sends me to town.', next: 'Next: Farm Blood Moor.' } };
const sent = (o) => ({ schema: 'watch.glance/1', sentence: Object.assign({}, base, o), needs: [] });
const need = (sev, text) => ({ id: 'n', kind: 'k', sev, text, actions: [] });
const longReason = 'the quest path is blocked, so I farm my level plan meanwhile and try again later when something has changed ' + 'x'.repeat(10);
const cases = {
  statusline: { bare: true, states: {
    ok: sent({}),
    warn: sent({ tone: 'warn', word: 'Waiting', reason: 'town chores', margin: 'risky' }),
    danger: sent({ tone: 'danger', word: 'Dead', reason: 'died in Cold Plains', margin: 'too hard', voice: '' }),
    info: sent({ tone: 'info', word: 'Paused', place: '', reason: 'walking home first', margin: '', uptime: '', why: { code: 'activity', now: 'Walking home.', change: '', next: '' } }),
    loading: undefined,
    error: null,
    longreason: sent({ reason: longReason }),
  } },
  'needs-count': { bare: true, states: {
    zero: sent({}),
    one: Object.assign(sent({}), { needs: [need('danger', 'Weapon broken')] }),
    three: Object.assign(sent({}), { needs: [need('info', 'Idle'), need('warn', 'Go for Blood Raven?'), need('info', 'Belt empty')] }),
    error: null,
  } },
};
const heroes = { schema: 'watch.hero/1', known: true, name: 'Pally', class: 'Paladin', level: 9, area: 'Blood Moor', life: 30, maxLife: 150, mana: 41, maxMana: 60 };
let bad = 0;
function check(id, st, root, c, got, hides) {
  const f = path.join(__dirname, 'glance_dom_' + id + '_' + st + '.txt');
  if (process.env.UPDATE === '1' || !fs.existsSync(f)) { if (process.env.UPDATE !== '1') { console.error('missing golden ' + f); bad++; return; } fs.writeFileSync(f, got); return; }
  if (fs.readFileSync(f, 'utf8') !== got) { console.error('golden differs: ' + f + '\n' + got); bad++; }
  assert.strictEqual(c.hidden, hides, id + ' ' + st + ' hidden');
}
for (const id of Object.keys(cases)) {
  const P = load(path.join(dir, id, 'panel.js'));
  for (const st of Object.keys(cases[id].states)) {
    const { root, c } = ctxFor(true);
    const api = P.mount(root, c);
    const d = cases[id].states[st];
    if (d !== undefined) api.update(d, d ? undefined : { status: 404, fails: 1 });
    check(id, st, root, c, dump(root, ''), id === 'needs-count' && st === 'error');
  }
}
{ // hero-meters in the cockpit band: slim, two bars and one number each; the classic form is topband_dom_hero-meters_*
  const P = load(path.join(dir, 'hero-meters', 'panel.js'));
  const { root, c } = ctxFor(true);
  root.dataset.cockpitPanel = 'band';
  const api = P.mount(root, c);
  api.update(heroes);
  check('hero-meters', 'slim', root, c, dump(root, ''), false);
}
// rules behind the pictures
{
  const S = load(path.join(dir, 'statusline', 'panel.js')), N = load(path.join(dir, 'needs-count', 'panel.js'));
  assert.strictEqual(S.tone('danger'), 'danger'); assert.strictEqual(S.tone('weird'), 'info'); assert.strictEqual(S.tone(undefined), 'info');
  assert.strictEqual(JSON.stringify(S.rows({ why: {}, reason: '' })), '[]');
  assert.strictEqual(JSON.stringify(S.rows({ reason: 'r', why: { now: 'n' } }).map((r) => r[0])), '["Now"]');
  assert.strictEqual(S.rows(base).length, 6);
  assert.ok(!/\d/.test(S.rows({ margin: 'risky', why: {} }).map((r) => r[1]).join('')), 'the margin is a word');
  assert.strictEqual(N.worst([]), ''); assert.strictEqual(N.worst([need('info', 'a'), need('warn', 'b')]), 'warn');
  assert.strictEqual(N.worst([need('danger', 'a'), need('warn', 'b')]), 'danger'); assert.strictEqual(N.worst([need('odd', 'a')]), 'info');
  assert.strictEqual(N.label(0), 'all quiet'); assert.strictEqual(N.label(1), '1 needs you'); assert.strictEqual(N.label(3), '3 need you');
}
// only the sentence is live, and the long line keeps its full text in the title
{
  const P = load(path.join(dir, 'statusline', 'panel.js'));
  const { root, c } = ctxFor(true);
  P.mount(root, c).update(sent({ reason: longReason }));
  const out = dump(root, '');
  assert.strictEqual((out.match(/aria-live/g) || []).length, 1, 'one live region');
  assert.ok(out.includes('title=' + JSON.stringify('Cold Plains · ' + longReason)), 'full text in the title');
}
// a single failed poll after good data keeps the sentence; two in a row say No signal
{
  const P = load(path.join(dir, 'statusline', 'panel.js'));
  const { root, c } = ctxFor(true);
  const inst = P.mount(root, c);
  inst.update(sent({}));
  inst.update(null, { status: 0, fails: 1 });
  assert.ok(dump(root, '').includes('Fighting') && !dump(root, '').includes('No signal'), 'one failed poll is a blip');
  inst.update(null, { status: 0, fails: 2 });
  assert.ok(dump(root, '').includes('No signal'), 'two failed polls say No signal');
  inst.update(sent({}));
  assert.ok(dump(root, '').includes('Fighting'), 'it recovers');
}
if (bad) process.exit(1);
console.log('ok');
