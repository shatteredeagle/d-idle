'use strict';
// Fleet strip, failsafe banner, safety/deaths view, tuning (knobs + upgrades) and the viewport slot.
// Everything here renders from server data. Knobs and upgrades are never listed in this file: they come from
// GET {base}/knobs (see "Tuning contract" below). Only this origin's /api/v1 is contacted; text goes in with textContent.
//
// Tuning contract (proposed by WP-DASH, the server side does not serve it yet; a 404 shows a notice, nothing breaks):
//   GET  {base}/knobs   -> {knobs:[env.Knob], values:{key:number}, profile:"name", profiles:[{name,label,desc}],
//                           upgrades:[{id,track,tier,label,desc,cost,requires:[id],effects:[{knob,value}],state:"locked|available|unlocked"}],
//                           telemetry:{name:number | {label,value,unit}}}
//                          A bare array of env.Knob is accepted too (read-only, values = defaults).
//   PUT  {base}/knobs   {profile?:string, overrides?:{key:number|null}} (null = back to the profile value) -> the GET body
//   POST {base}/upgrades/{id}/unlock -> the GET body
(() => {
const D = window.D2;
if (!D) return;
const { h, s, $, clear, nf, dur, cap, clock } = D;
const ROOT = '/api/v1';

async function getJSON(url, allowBody) {
  const res = await fetch(url, { credentials: 'same-origin', headers: { Accept: 'application/json' } });
  let data = null;
  try { data = await res.json(); } catch (e) { /* not JSON */ }
  if (!res.ok && !(allowBody && data)) { const err = new Error((data && data.error) || res.status + ' ' + res.statusText); err.status = res.status; throw err; }
  return data;
}
const ago = (t) => { const sec = Math.max(0, (Date.now() - new Date(t)) / 1000); return dur(sec) + ' ago'; };
const when = (t) => { const d = new Date(t); return d.toLocaleDateString([], { month: 'short', day: 'numeric' }) + ' ' + clock(t).slice(0, 5); };
const pathOf = (multiPath, flatPath) => (D.multi ? D.base + multiPath : ROOT + flatPath);

// ---- views --------------------------------------------------------------------------------------
const VIEWS = ['overview', 'safety', 'tuning'];
function setView(v) {
  if (!VIEWS.includes(v)) v = 'overview';
  for (const name of VIEWS) $('view-' + name).hidden = name !== v;
  document.querySelectorAll('#views button').forEach((b) => { if (b.dataset.view === v) b.setAttribute('aria-current', 'page'); else b.removeAttribute('aria-current'); });
  const p = new URLSearchParams(location.hash.slice(1)); p.set('v', v); history.replaceState(null, '', '#' + p);
  if (v === 'safety') { refreshDeaths(); refreshHealth(); }
  if (v === 'tuning') loadKnobs(false);
  vpTick(); // start or stop the game view
}
document.querySelectorAll('#views button').forEach((b) => { b.onclick = () => setView(b.dataset.view); });
$('failsafe-link').onclick = (e) => { e.preventDefault(); setView('safety'); };

// ---- fleet --------------------------------------------------------------------------------------
let fleet = null;
function renderPicker() {
  const sel = $('inst-select');
  clear(sel);
  for (const i of D.instances) sel.append(h('option', { value: i.id, text: i.title || i.id }));
  sel.value = D.instance || '';
  $('inst-pick').hidden = D.instances.length < 2;
}
$('inst-select').onchange = (e) => D.select(e.target.value);

function chipText(i) {
  const c = (i.runner && i.runner.character) || {};
  const who = [c.class, c.level ? 'level ' + c.level : ''].filter(Boolean).join(' ');
  return who || (i.runner && i.runner.state) || '';
}
function renderFleet() {
  const nav = $('fleet');
  const list = (fleet && fleet.instances) || D.instances;
  nav.hidden = !D.multi || list.length < 1;
  if (nav.hidden) return;
  clear(nav);
  for (const i of list) {
    const bad = i.ok === false || (i.runner && i.runner.state === 'error');
    const cls = bad ? 'error' : i.ready === false ? 'paused' : i.mode || '';
    const act = (i.runner && i.runner.activity) || '';
    const b = h('button', { type: 'button', class: 'chip' + (i.id === D.instance ? ' sel' : '') + (bad ? ' bad' : ''), 'aria-pressed': String(i.id === D.instance), onclick: () => D.select(i.id) },
      h('span', { class: 'dot ' + cls, 'aria-hidden': 'true' }),
      h('span', { class: 'chip-main' },
        h('b', { text: i.title || i.id }),
        h('span', { class: 'muted', text: chipText(i) }),
        h('span', { class: 'muted chip-act', text: bad ? 'needs attention' : (i.mode === 'stopped' ? 'stopped' : act) }),
        h('span', { class: 'muted', text: nf.format(Math.round(i.xpPerHour || 0)) + ' xp/h' })));
    nav.append(b);
  }
}
function connectFleet() {
  let es;
  try { es = new EventSource(ROOT + '/fleet/events/stream'); } catch (e) { return; }
  es.addEventListener('fleet', (e) => { fleet = JSON.parse(e.data); renderFleet(); updateFailsafe(); renderSentinel(); });
  es.addEventListener('error', () => { /* EventSource retries by itself; the next fleet event refreshes everything */ });
}
async function pollFleet() {
  try { fleet = await getJSON(ROOT + '/fleet'); renderFleet(); updateFailsafe(); renderSentinel(); } catch (e) { /* stream or next poll */ }
}

// ---- health and failsafe ------------------------------------------------------------------------
let health = null, ready = false; // ready: the instance is known, so instance-relative paths are right
async function refreshHealth() {
  if (!ready) return;
  const my = D.instance;
  try {
    const hb = await getJSON(pathOf('/health', '/bot/health'), true);
    if (my === D.instance) health = hb;
  } catch (e) { if (my === D.instance) health = null; }
  updateFailsafe(); renderSentinel();
}
function failsafeReasons() {
  const out = [];
  if (fleet) {
    for (const i of fleet.instances || []) if (i.ok === false) out.push((i.title || i.id) + ': unhealthy' + (i.runner && i.runner.message ? ' (' + i.runner.message + ')' : ''));
    const sen = fleet.sentinel;
    if (sen) for (const i of sen.instances || []) if (i.health === 'attention' || i.health === 'down') out.push((i.character || i.id) + ': ' + (i.line || i.health));
  }
  if (health && (health.fatal || health.ok === false || health.ladderStep > 0 || health.stuckReason)) {
    const why = health.fatal || health.stuckReason || health.lastError || (health.ladderStep > 0 ? 'failsafe ladder step ' + health.ladderStep : 'unhealthy');
    const line = (D.instance ? D.instance + ': ' : '') + why;
    if (!out.some((x) => x.startsWith((D.instance || '') + ':') && D.multi)) out.push(line);
  }
  return out;
}
function updateFailsafe() {
  const r = failsafeReasons();
  const el = $('failsafe');
  el.hidden = r.length === 0;
  if (r.length) { $('failsafe-title').textContent = r.length === 1 ? 'Failsafe active' : r.length + ' failsafes active'; $('failsafe-text').textContent = r.slice(0, 3).join('; ') + (r.length > 3 ? '; and more' : ''); }
  $('safety-count').textContent = r.length ? '(' + r.length + ')' : '';
}

// ---- sentinel panel -----------------------------------------------------------------------------
function kv(rows) {
  const dl = h('dl', { class: 'kv' });
  for (const [k, v] of rows) if (v !== '' && v != null && v !== false) dl.append(h('dt', { text: k }), h('dd', { text: String(v) }));
  return dl;
}
function renderSentinel() {
  const hb = $('sen-health'), list = $('sen-list'), recent = $('sen-recent');
  clear(hb); clear(list); clear(recent);
  hb.append(h('h3', { text: 'This instance' }));
  if (health) {
    hb.append(kv([['Bot state', health.state], ['Runner', health.status], ['Activity', health.activity], ['In state for', health.sinceSeconds != null ? dur(health.sinceSeconds) : ''],
      ['Since progress', health.lastProgressSeconds != null ? dur(health.lastProgressSeconds) : ''], ['Stuck reason', health.stuckReason],
      ['Failsafe ladder step', health.ladderStep || ''], ['Paused', health.paused ? (health.selfPaused ? 'by the bot itself' : 'yes') : ''], ['Last error', health.lastError], ['Fatal', health.fatal]]));
  } else hb.append(h('p', { class: 'muted', text: 'No health report from this instance.' }));
  const sen = fleet && fleet.sentinel;
  $('sen-mode').textContent = sen ? 'Sentinel: ' + sen.mode : '';
  if (!sen) { list.append(h('h3', { text: 'Sentinel' }), h('p', { class: 'muted', text: 'The sentinel is not running on this server.' })); return; }
  list.append(h('h3', { text: 'Game processes' }));
  for (const i of sen.instances || []) {
    const rules = (i.rules || []).filter((r) => r.active || r.attempts);
    const o = i.obs || {};
    const row = h('div', { class: 'sen-row ' + (i.health || '') },
      h('div', { class: 'goal-top' }, h('b', { text: (i.character || i.id) + (i.agent ? ' (' + i.agent + ')' : '') }), h('span', { class: 'badge ' + (i.health === 'ok' ? 'done' : i.health === 'unknown' ? '' : 'failed'), text: i.health })),
      h('div', { class: 'goal-meta', text: i.line || '' }));
    const bits = [];
    if (o.procAlive === false) bits.push('process gone'); else if (o.procAlive) bits.push('CPU ' + Math.round(o.cpuPct || 0) + '%');
    if (o.state) bits.push(o.state);
    if (o.hung) bits.push('window hung');
    if (o.life != null && o.maxLife) bits.push('life ' + o.life + '/' + o.maxLife);
    if ((o.dialogs || []).length) bits.push('dialog: ' + o.dialogs.join(', '));
    if (bits.length) row.append(h('div', { class: 'goal-meta', text: bits.join(', ') }));
    if (i.acting) row.append(h('div', { class: 'goal-meta', text: 'Acting: ' + i.acting }));
    for (const r of rules) row.append(h('div', { class: 'rule' + (r.active ? ' on' : ''), text: r.rule + (r.active ? ' active ' + dur(r.forSec) : '') + (r.attempts ? ', attempts ' + r.attempts : '') + (r.step ? ', step ' + r.step : '') }));
    if (i.lastAction) row.append(h('div', { class: 'goal-meta', text: 'Last action ' + ago(i.lastAction.time) + ': ' + [i.lastAction.rule, i.lastAction.action].filter(Boolean).join(' / ') }));
    list.append(row);
  }
  const rec = (sen.recent || []).slice(-30).reverse();
  if (!rec.length) recent.append(h('li', {}, h('span'), h('span', { class: 'muted', text: 'Nothing yet.' })));
  for (const r of rec) {
    recent.append(h('li', { class: r.kind === 'error' ? 'error' : r.kind === 'acted' ? 'level_up' : '' }, h('time', { datetime: r.time, text: clock(r.time) }),
      h('span', { text: [r.instance, r.kind, r.rule, r.action, r.evidence || r.note].filter(Boolean).join(' / ') })));
  }
}

// ---- deaths and recovery ------------------------------------------------------------------------
let deaths = null;
async function refreshDeaths() {
  if (!ready) return;
  const my = D.instance;
  try { const d = await getJSON(pathOf('/deaths', '/bot/deaths')); if (my === D.instance) deaths = d; } catch (e) { if (my === D.instance) deaths = null; }
  renderDeaths();
}
function tile(v, l) { return h('div', { class: 'tile' }, h('div', { class: 'v', text: v }), h('div', { class: 'l', text: l })); }
function sparkline(curve) {
  const pts = (curve || []).slice().sort((a, b) => b.msBefore - a.msBefore);
  if (pts.length < 2) return null;
  const W = 160, H = 34, t0 = pts[0].msBefore || 1;
  const xy = pts.map((p) => ((1 - p.msBefore / t0) * (W - 4) + 2).toFixed(1) + ',' + ((1 - Math.min(100, Math.max(0, p.pct)) / 100) * (H - 4) + 2).toFixed(1));
  return s('svg', { viewBox: '0 0 ' + W + ' ' + H, class: 'spark', role: 'img', 'aria-label': 'Life percent over the last ' + (t0 / 1000).toFixed(1) + ' seconds before death' },
    s('polyline', { points: xy.join(' '), fill: 'none', stroke: 'var(--life)', 'stroke-width': 2 }));
}
function deathCard(r) {
  const state = r.retrieved ? 'Corpse retrieved' : r.abandoned ? 'Corpse abandoned' : 'Recovering';
  const li = h('li', { class: 'death' },
    h('div', { class: 'goal-top' }, h('b', { text: r.areaName || 'Area ' + r.area }), h('span', { class: 'badge ' + (r.retrieved ? 'done' : r.abandoned ? 'failed' : 'paused'), text: state })),
    h('div', { class: 'goal-meta', text: [when(r.time), r.level ? 'level ' + r.level : '', r.difficulty ? cap(r.difficulty) : '', r.killer ? 'killed by ' + r.killer + (r.killerBoss ? ' (boss)' : '') : 'killer unknown'].filter(Boolean).join(', ') }));
  const cost = [r.costGold ? nf.format(r.costGold) + ' gold' : '', r.costItems ? r.costItems + ' items' : '', r.costSeconds ? dur(r.costSeconds) : ''].filter(Boolean).join(', ');
  const more = h('div', { class: 'goal-meta' });
  if (r.corpseValue) more.append('Gear on corpse about ' + nf.format(r.corpseValue) + ' gold. ');
  if (cost) more.append('Cost: ' + cost + '. ');
  if (r.why) more.append(r.why);
  if (more.childNodes.length) li.append(more);
  const au = r.autopsy;
  if (au) {
    li.append(h('div', { class: 'goal-meta', text: au.summary }));
    const k = au.kit || {};
    li.append(h('div', { class: 'goal-meta', text: ['Kit: belt ' + (k.BeltHP || 0) + ' hp / ' + (k.BeltMP || 0) + ' mp / ' + (k.BeltRV || 0) + ' rv', (k.TownPortals || 0) + ' portal scrolls', nf.format(k.Gold || 0) + ' gold carried',
      au.killerType ? au.killerType + ' killer' + (au.killerLevel ? ', monster level ' + au.killerLevel : '') : ''].filter(Boolean).join(', ') }));
    if (au.damagePer5s && au.damagePer5s.length) li.append(h('div', { class: 'goal-meta', text: 'Life lost per 5 s before dying: ' + au.damagePer5s.map((d) => d + '%').join(', ') }));
    if (au.reaction) li.append(h('div', { class: 'goal-meta', text: 'The bot will: ' + au.reaction }));
  }
  const sp = sparkline(r.lifeCurve);
  if (sp) li.append(sp);
  return li;
}
function renderHeat() {
  const box = $('heat'), sel = $('heat-area');
  clear(box);
  const heat = (deaths && deaths.heat) || [];
  const areas = [...new Set(heat.map((p) => p.area))];
  const names = {};
  for (const a of (deaths && deaths.byArea) || []) names[a.area] = a.name;
  const keep = sel.value;
  clear(sel);
  for (const a of areas) sel.append(h('option', { value: String(a), text: names[a] || 'Area ' + a }));
  if (areas.map(String).includes(keep)) sel.value = keep;
  if (!areas.length) { box.append(h('p', { class: 'muted', text: 'No deaths recorded yet.' })); return; }
  const pts = heat.filter((p) => String(p.area) === sel.value);
  const cell = 8;
  const xs = pts.map((p) => p.x), ys = pts.map((p) => p.y);
  const x0 = Math.min(...xs) - 3 * cell, x1 = Math.max(...xs) + 3 * cell, y0 = Math.min(...ys) - 3 * cell, y1 = Math.max(...ys) + 3 * cell;
  const max = Math.max(...pts.map((p) => p.count));
  const svg = s('svg', { viewBox: [x0, y0, Math.max(x1 - x0, 12 * cell), Math.max(y1 - y0, 8 * cell)].join(' '), class: 'heatmap', role: 'img', 'aria-label': 'Death heat map of ' + (names[+sel.value] || 'the area') + ', ' + pts.length + ' cells' });
  svg.append(s('rect', { x: x0, y: y0, width: 1e5, height: 1e5, fill: 'var(--surface2)' }));
  for (const p of pts) {
    const r = s('rect', { x: p.x - cell / 2, y: p.y - cell / 2, width: cell, height: cell, rx: 1.5, fill: 'var(--bad)', opacity: (0.25 + 0.75 * p.count / max).toFixed(2) });
    r.append(s('title', {}, p.count + (p.count === 1 ? ' death' : ' deaths') + ' near ' + p.x + ', ' + p.y));
    svg.append(r);
  }
  box.append(svg, h('p', { class: 'hint', text: 'Each square is one ' + cell + ' by ' + cell + ' subtile cell; darker means more deaths.' }));
}
$('heat-area').onchange = renderHeat;
function renderDeaths() {
  const d = deaths;
  const tiles = $('death-tiles'), list = $('death-list'), areas = $('death-areas');
  clear(tiles); clear(list); clear(areas);
  $('death-meta').textContent = d && d.pausedForDeaths ? 'Paused because of repeated deaths' : '';
  if (!d) { tiles.append(h('p', { class: 'muted', text: 'This server does not report deaths for this instance.' })); renderHeat(); return; }
  tiles.append(tile(nf.format(d.total || 0), 'Deaths'), tile(nf.format(d.lastHour || 0), 'Last hour'), tile(nf.format(d.totalGold || 0), 'Gold lost'), tile(nf.format(d.totalItems || 0), 'Items lost'));
  renderHeat();
  const by = (d.byArea || []).slice().sort((a, b) => b.deaths - a.deaths).slice(0, 8);
  const max = Math.max(1, ...by.map((a) => a.deaths));
  for (const a of by) {
    const fill = h('i'); fill.style.width = (100 * a.deaths / max) + '%'; fill.style.background = 'var(--bad)';
    areas.append(h('div', { class: 'qrow' }, h('span', { text: a.name || 'Area ' + a.area }), h('div', { class: 'qbar' }, fill), h('span', { class: 'n', text: a.deaths })));
  }
  if (!by.length) areas.append(h('p', { class: 'muted', text: 'Nothing yet.' }));
  const recs = (d.records || []).slice(-30).reverse();
  if (!recs.length) list.append(h('li', { class: 'muted', text: 'No deaths. Good.' }));
  for (const r of recs) list.append(deathCard(r));
}
let deathTimer = 0;
D.on('log', (ev) => { if (ev.type === 'death' && !deathTimer) deathTimer = setTimeout(() => { deathTimer = 0; refreshDeaths(); }, 1500); });

// ---- tuning: knobs and upgrades -----------------------------------------------------------------
const tune = { data: null, edits: {}, profile: null, loaded: '' };
function normalizeTune(d) {
  if (Array.isArray(d)) d = { knobs: d };
  const knobs = d.knobs || [];
  const values = Object.assign({}, ...knobs.map((k) => ({ [k.key]: k.default })), d.values || {});
  return { knobs, values, profile: d.profile || '', profiles: d.profiles || [], upgrades: d.upgrades || [], telemetry: d.telemetry || {}, readonly: !!d.readonly };
}
async function loadKnobs(force) {
  if (!ready) return;
  if (!force && tune.loaded === D.instance && tune.data) return;
  const my = D.instance;
  try {
    const d = normalizeTune(await getJSON(pathOf('/knobs', '/knobs')));
    if (my !== D.instance) return;
    Object.assign(tune, { data: d, edits: {}, profile: d.profile, loaded: my, error: '' });
  } catch (e) {
    if (my !== D.instance) return;
    Object.assign(tune, { data: null, edits: {}, loaded: my, error: e.status === 404 ? 'This server does not expose knobs yet (GET /knobs).' : e.message });
  }
  renderTune();
}
async function sendTune(url, method, body) {
  try {
    const res = await fetch(url, { method, credentials: 'same-origin', headers: { 'Content-Type': 'application/json' }, body: body ? JSON.stringify(body) : undefined });
    const data = await res.json().catch(() => null);
    if (!res.ok) throw new Error((data && data.error) || res.status + ' ' + res.statusText);
    D.banner('');
    await loadKnobs(true);
  } catch (e) { D.banner(e.message); }
}
const fmt = (k, v) => (v == null || Number.isNaN(v) ? '' : (Number.isInteger(v) ? nf.format(v) : String(Math.round(v * 1000) / 1000))) + (k.unit && k.unit !== 'count' ? ' ' + k.unit : '');
function stepFor(k) {
  if ([k.min, k.max, k.default].every(Number.isInteger)) return 1;
  const span = k.max - k.min;
  return span > 0 ? Number((span / 200).toPrecision(1)) : 0.01;
}
function knobRow(k, ro) {
  const val = k.key in tune.edits ? tune.edits[k.key] : tune.data.values[k.key];
  const dirty = k.key in tune.edits;
  const id = 'kb-' + k.key.replace(/[^a-z0-9]/gi, '-');
  const range = h('input', { type: 'range', id, min: k.min, max: k.max, step: stepFor(k), value: val, disabled: ro, 'aria-label': k.label });
  const num = h('input', { type: 'number', min: k.min, max: k.max, step: 'any', value: val, disabled: ro, 'aria-label': k.label + ' value', class: 'knob-num' });
  const set = (v, src) => {
    v = Math.min(k.max, Math.max(k.min, Number(v)));
    if (Number.isNaN(v)) return;
    if (v === tune.data.values[k.key]) delete tune.edits[k.key]; else tune.edits[k.key] = v;
    if (src !== range) range.value = v;
    if (src !== num) num.value = v;
    row.classList.toggle('dirty', k.key in tune.edits);
    updateApply();
  };
  range.oninput = () => set(range.value, range);
  num.onchange = () => set(num.value, num);
  const fromDefault = k.max > k.min ? (val - k.default) / (k.max - k.min) : 0;
  const better = k.higherIsBetter ? 'higher is better' : 'lower is better';
  const row = h('div', { class: 'knob' + (dirty ? ' dirty' : ''), 'data-key': k.key },
    h('div', { class: 'knob-head' }, h('label', { for: id, text: k.label || k.key }), h('span', { class: 'muted', text: k.key })),
    h('div', { class: 'knob-ctl' }, range, num, h('span', { class: 'muted', text: k.unit || '' })),
    h('div', { class: 'knob-meta muted', text: 'Default ' + fmt(k, k.default) + ', range ' + fmt({ unit: '' }, k.min) + ' to ' + fmt(k, k.max) + ', ' + better + (Math.abs(fromDefault) > 0.005 ? '. Currently ' + (((fromDefault > 0) === !!k.higherIsBetter) ? 'better' : 'worse') + ' than default.' : '.') }));
  if (k.desc) row.append(h('div', { class: 'knob-desc', text: k.desc }));
  return row;
}
function updateApply() {
  const n = Object.keys(tune.edits).length + (tune.profile !== tune.data.profile ? 1 : 0);
  $('knob-apply').disabled = !n; $('knob-revert').disabled = !n;
  $('knob-apply').textContent = n ? 'Apply ' + n + (n === 1 ? ' change' : ' changes') : 'Apply changes';
}
let filterText = '';
function renderTune() {
  const root = $('knobs'), up = $('upgrades');
  clear(root); clear(up);
  const d = tune.data;
  $('profile-wrap').hidden = !d || !d.profiles.length;
  if (!d) {
    $('knob-note').textContent = '';
    $('knob-apply').disabled = $('knob-revert').disabled = true;
    root.append(h('p', { class: 'muted', text: tune.error || 'Loading...' }));
    up.append(h('p', { class: 'muted', text: 'Upgrades come from the same data as the knobs.' }));
    return;
  }
  const ro = d.readonly;
  $('knob-note').textContent = ro ? 'Read only: this server sent the knob list without a way to change it.' : 'Every control comes from the knob metadata. Ranges are where the bot still works.';
  const ps = $('profile-select');
  clear(ps);
  for (const p of d.profiles) ps.append(h('option', { value: p.name, text: p.label || p.name }));
  ps.value = tune.profile;
  ps.disabled = ro;
  ps.onchange = () => { tune.profile = ps.value; updateApply(); };
  const prof = d.profiles.find((p) => p.name === tune.profile);
  if (prof && prof.desc) $('knob-note').textContent += ' Profile: ' + prof.desc;
  const tele = Object.entries(d.telemetry);
  if (tele.length) {
    const t = h('div', { class: 'tiles tele' });
    for (const [name, v] of tele) { const o = typeof v === 'object' ? v : { value: v }; t.append(tile((typeof o.value === 'number' ? nf.format(Math.round(o.value * 100) / 100) : String(o.value)) + (o.unit ? ' ' + o.unit : ''), o.label || name)); }
    root.append(t);
  }
  root.append(h('label', { class: 'knob-filter' }, 'Filter', h('input', { type: 'search', value: filterText, placeholder: 'name, key or description', oninput: (e) => { filterText = e.target.value; renderKnobGroups(groupsBox, ro); } })));
  const groupsBox = h('div', { class: 'knob-groups' });
  root.append(groupsBox);
  renderKnobGroups(groupsBox, ro);
  updateApply();
  renderUpgrades(up);
}
function renderKnobGroups(box, ro) {
  clear(box);
  const q = filterText.trim().toLowerCase();
  const groups = new Map();
  for (const k of tune.data.knobs) {
    if (q && !(k.key + ' ' + k.label + ' ' + (k.desc || '')).toLowerCase().includes(q)) continue;
    const g = k.key.includes('.') ? k.key.slice(0, k.key.indexOf('.')) : 'general';
    if (!groups.has(g)) groups.set(g, []);
    groups.get(g).push(k);
  }
  if (!groups.size) box.append(h('p', { class: 'muted', text: 'No knob matches.' }));
  for (const [g, ks] of groups) {
    const det = h('details', { class: 'knob-group', open: !!q || groups.size === 1 }, h('summary', {}, cap(g), ' ', h('span', { class: 'count', text: '(' + ks.length + ')' })));
    for (const k of ks) det.append(knobRow(k, ro));
    box.append(det);
  }
}
$('knob-revert').onclick = () => { tune.edits = {}; tune.profile = tune.data.profile; renderTune(); };
$('knob-apply').onclick = () => {
  const body = {};
  if (tune.profile !== tune.data.profile) body.profile = tune.profile;
  if (Object.keys(tune.edits).length) body.overrides = tune.edits;
  sendTune(pathOf('/knobs', '/knobs'), 'PUT', body);
};

function renderUpgrades(root) {
  const ups = tune.data.upgrades;
  if (!ups.length) { root.append(h('p', { class: 'muted', text: 'No upgrades are defined by this server.' })); return; }
  const label = (key) => { const k = tune.data.knobs.find((x) => x.key === key); return k ? k.label : key; };
  const tracks = new Map();
  for (const u of ups) { const t = u.track || 'general'; if (!tracks.has(t)) tracks.set(t, []); tracks.get(t).push(u); }
  for (const [track, list] of tracks) {
    list.sort((a, b) => (a.tier || 0) - (b.tier || 0));
    const ol = h('ol', { class: 'upgrade-track' });
    for (const u of list) {
      const st = u.state || (u.unlocked ? 'unlocked' : u.available ? 'available' : 'locked');
      const fx = (u.effects || []).map((e) => { const key = e.knob || e.Knob, v = e.value != null ? e.value : e.Value, k = tune.data.knobs.find((x) => x.key === key); return label(key) + ' to ' + (k ? fmt(k, v) : v); });
      const li = h('li', { class: 'upgrade ' + st },
        h('div', { class: 'goal-top' }, h('b', { text: (u.label || u.id) + (u.tier ? ' (tier ' + u.tier + ')' : '') }), h('span', { class: 'badge ' + (st === 'unlocked' ? 'done' : st === 'available' ? 'active' : ''), text: cap(st) })));
      if (u.desc) li.append(h('div', { class: 'goal-meta', text: u.desc }));
      if (fx.length) li.append(h('div', { class: 'goal-meta', text: 'Sets ' + fx.join(', ') }));
      const need = [];
      if (u.cost) need.push('Cost ' + (typeof u.cost === 'object' ? Object.entries(u.cost).map(([k2, v]) => v + ' ' + k2).join(', ') : nf.format(u.cost)));
      if ((u.requires || []).length) need.push('Needs ' + u.requires.map((r) => { const o = ups.find((x) => x.id === r); return o ? o.label || o.id : r; }).join(', '));
      if (need.length) li.append(h('div', { class: 'goal-meta', text: need.join('. ') }));
      if (st === 'available' && !tune.data.readonly) li.append(h('button', { class: 'btn small primary', type: 'button', onclick: () => sendTune(pathOf('/upgrades/' + encodeURIComponent(u.id) + '/unlock', '/upgrades/' + encodeURIComponent(u.id) + '/unlock'), 'POST') }, 'Unlock'));
      ol.append(li);
    }
    root.append(h('h3', { text: track }), ol);
  }
}

// ---- viewport slot ------------------------------------------------------------------------------
// The image shows only while /viewport/stream (MJPEG) delivers frames. Overlay API for other scripts:
//   D2.viewport.setOverlay([{type:'rect'|'circle'|'line'|'text', x, y, w, h, r, x2, y2, text, color}])  (pixels of the frame)
// (the server's /viewport/overlay JSON is no longer polled: the container has no scene source).
const vp = { timer: 0, shapes: [], gone: 0, live: false, since: 0 };
D.viewport = {
  setOverlay(shapes) { vp.shapes = Array.isArray(shapes) ? shapes : []; drawOverlay(); },
  clearOverlay() { vp.shapes = []; drawOverlay(); },
};
function drawOverlay() {
  const img = $('vp-img'), svg = $('vp-overlay');
  clear(svg);
  const w = img.naturalWidth, hgt = img.naturalHeight;
  if (!w) return;
  svg.setAttribute('viewBox', '0 0 ' + w + ' ' + hgt);
  for (const sh of vp.shapes) {
    const c = sh.color || 'var(--accent)', t = sh.type || sh.kind;
    const a = { stroke: c, fill: 'none', 'stroke-width': 2, 'vector-effect': 'non-scaling-stroke' };
    if (t === 'rect') svg.append(s('rect', { ...a, x: sh.x, y: sh.y, width: sh.w, height: sh.h }));
    else if (t === 'circle') svg.append(s('circle', { ...a, cx: sh.x, cy: sh.y, r: sh.r || 6 }));
    else if (t === 'line') svg.append(s('line', { ...a, x1: sh.x, y1: sh.y, x2: sh.x2, y2: sh.y2 }));
    else if (t === 'text') svg.append(s('text', { x: sh.x, y: sh.y, fill: c, 'font-size': sh.size || 14 }, sh.text || ''));
    if (sh.text && t !== 'text') svg.append(s('text', { x: sh.x, y: (sh.y || 0) - 4, fill: c, 'font-size': 12 }, sh.text));
  }
}
function vpUrl(name) { return D.base + '/viewport/' + name; }
// The game view is one MJPEG stream (/viewport/stream, view only). It runs only while the tab is visible and the
// overview is shown: a hidden tab stops draining the stream and the server drops it. A stream that ends (the
// server dropped a stalled client, the game restarted) fires load or error and is reopened with a backoff.
function vpWanted() { return !document.hidden && !$('view-overview').hidden && !(D.multi && !D.instance); }
function vpStop() {
  clearTimeout(vp.timer); vp.timer = 0; vp.live = false;
  const img = $('vp-img');
  if (img.getAttribute('src')) img.removeAttribute('src');
}
function vpTick() {
  clearTimeout(vp.timer); vp.timer = 0;
  if (!vpWanted()) { vpStop(); vp.timer = setTimeout(vpTick, 2000); return; }
  if (vp.live) return;
  vp.live = true; vp.since = performance.now();
  $('vp-card').hidden = false; $('vp-meta').textContent = 'live, view only';
  $('vp-img').src = vpUrl('stream') + '?t=' + Date.now();
}
function vpLost() {
  if (!vp.live) return;
  vp.live = false;
  // a stream that lived a while resets the backoff; quick failures back off 1 s .. 30 s
  vp.gone = performance.now() - vp.since > 10000 ? 0 : vp.gone + 1;
  $('vp-img').removeAttribute('src');
  vp.timer = setTimeout(vpTick, Math.min(30000, 1000 * 2 ** Math.min(vp.gone, 5)));
}
// Like cmd/d2watch/web/app.js: an ended multipart stream fires load (not error) and freezes on its last frame,
// a refused one (no game yet, too many viewers) fires error: both mean "stream gone".
$('vp-img').addEventListener('error', () => { $('vp-card').hidden = true; vpLost(); });
// Firefox-family browsers fire load on the FIRST part of a multipart stream, so load is not "stream gone": a stall
// watchdog reopens a stream that has not started or has not been reopened for 60 s (a dropped client freezes silently).
$('vp-img').addEventListener('load', () => { vp.loaded = performance.now(); });
setInterval(() => { if (vp.live && performance.now() - vp.since > 60000) { vpLost(); vp.gone = 0; } }, 5000);
document.addEventListener('visibilitychange', () => { if (document.hidden) { vpStop(); } else { vpTick(); } });
$('vp-img').addEventListener('load', drawOverlay);

// ---- wiring -------------------------------------------------------------------------------------
D.on('discovered', (list) => {
  if (!list) return;
  renderPicker(); renderFleet();
  pollFleet(); connectFleet();
  setInterval(pollFleet, 10000);
});
D.on('instance', () => {
  ready = true;
  health = deaths = null; tune.data = null; tune.loaded = ''; vp.shapes = []; vp.gone = 0;
  vpStop(); $('vp-card').hidden = true;
  if (D.multi) { $('inst-select').value = D.instance || ''; renderFleet(); }
  refreshHealth(); refreshDeaths();
  if (!$('view-tuning').hidden) loadKnobs(true);
  vpTick();
});
setInterval(refreshHealth, 5000);
setInterval(() => { if (!$('view-safety').hidden) refreshDeaths(); }, 15000);
const fromHash = () => {
  const p = new URLSearchParams(location.hash.slice(1));
  setView(p.get('v') || 'overview');
  const i = p.get('i');
  if (i && D.instances.some((x) => x.id === i)) D.select(i);
};
window.addEventListener('hashchange', fromHash);
fromHash();
})();
