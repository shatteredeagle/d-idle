// DOM goldens of the top-band panels (hero-meters, alerts-count, hero-detail) in a stub DOM: each state is rendered
// through the panel's own mount/update and compared with testdata/topband_dom_<panel>_<state>.txt (UPDATE=1 rewrites).
// Usage: node topband_dom.js path/to/web/panels
const assert = require('assert'), fs = require('fs'), path = require('path'), { load, mk } = require('./panel_harness.js');
const dir = process.argv[2];

function node(tag, cls, text) {
  const e = mk(tag);
  e.className = cls || '';
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

const hero = { schema: 'watch.hero/1', known: true, name: 'Pally', class: 'Paladin', level: 9, difficulty: 'normal', area: 'Blood Moor', life: 30, maxLife: 150, mana: 41, maxMana: 60,
  xp: 15000, xpLevel: 12000, xpNext: 18000, xpPct: 50, xpPerHour: 12345, gold: 1240, stashGold: 5300, state: 'fight', stateLabel: 'fighting', activity: 'Fighting', goal: null, queue: [], why: {},
  stats: { resists: { fire: 12, cold: 30, light: -8, poison: 5 }, readiness: { light: 'amber', area: 'Blood Moor', reason: 'Lightning resistance is under the line.' } } };
const bare = Object.assign({}, hero, { name: 'Sorc', class: 'Sorceress', maxLife: 0, maxMana: 0, xpPct: null, xpNext: 0, gold: null, stashGold: null, stats: null });
const alerts = { alerts: [{ seq: 1, severity: 'warn', text: 'Short Staff is broken.' }, { seq: 2, severity: 'alarm', text: 'Life under 25% in Blood Moor.' }] };
const cases = {
  'hero-meters': { bare: true, states: { ready: hero, unreported: bare, unknown: { known: false }, error: null } },
  'alerts-count': { bare: true, states: { ready: alerts, warn: { alerts: [alerts.alerts[0]] }, empty: { alerts: [] }, error: null } },
  'hero-detail': { bare: false, states: { ready: hero, nostats: bare, error: null } },
};
let bad = 0;
for (const id of Object.keys(cases)) {
  const P = load(path.join(dir, id, 'panel.js'));
  for (const st of Object.keys(cases[id].states)) {
    const { root, c } = ctxFor(cases[id].bare);
    const api = P.mount(root, c);
    api.update(cases[id].states[st], cases[id].states[st] ? undefined : { status: 404, fails: 1 });
    const got = dump(root, ''), f = path.join(__dirname, 'topband_dom_' + id + '_' + st + '.txt');
    if (process.env.UPDATE === '1' || !fs.existsSync(f)) { if (process.env.UPDATE !== '1') { console.error('missing golden ' + f); bad++; continue; } fs.writeFileSync(f, got); continue; }
    if (fs.readFileSync(f, 'utf8') !== got) { console.error('golden differs: ' + f + '\n' + got); bad++; }
    // the contract on every state: an error or an unknown hero hides the panel instead of showing zeros
    if (st === 'error' || st === 'empty' || st === 'unknown') assert.strictEqual(c.hidden, true, id + ' ' + st + ' must hide');
    else assert.strictEqual(c.hidden, false, id + ' ' + st + ' must show');
  }
}
// rules behind the pictures
{
  const M = load(path.join(dir, 'hero-meters', 'panel.js')), D = load(path.join(dir, 'hero-detail', 'panel.js')), A = load(path.join(dir, 'alerts-count', 'panel.js'));
  assert.strictEqual(M.level(30, 150), 'bad'); assert.strictEqual(M.level(60, 150), 'warn'); assert.strictEqual(M.level(140, 150), ''); assert.strictEqual(M.level(0, 0), '');
  {
    const t = D.xpText({ xp: 79451, xpLevel: 72144, xpNext: 90180, xpPct: 40.4, xpPerHour: 12345.6 });
    assert.strictEqual(t, 'XP 79,451 / 90,180 (40%) · 12,346 xp/h');
    assert.strictEqual(D.xpText({ xp: 100, xpNext: 500, xpPct: 20 }), 'XP 100 / 500 (20%) · xp/h measuring');
    assert.strictEqual(D.xpText({ xp: 0, xpNext: 0 }), null); assert.strictEqual(D.xpText(null), null);
    for (const h of [{ xp: 5, xpNext: 9, xpPct: null, xpPerHour: null }, { xp: 79451, xpNext: 82000, xpPct: 97.1, xpPerHour: NaN }]) assert.ok(!/NaN|undefined|null| - /.test(D.xpText(h)), D.xpText(h));
  }
  assert.strictEqual(D.resistLevel(-8), 'bad'); assert.strictEqual(D.resistLevel(12), ''); assert.strictEqual(D.resistLevel(30), '');
  assert.strictEqual(A.tone(alerts.alerts), 'bad'); assert.strictEqual(A.tone([alerts.alerts[0]]), 'warn');
}
if (bad) process.exit(1);
console.log('ok');
