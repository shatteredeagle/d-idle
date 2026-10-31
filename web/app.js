'use strict';
// D2 Idle dashboard. No framework, no build step, no network except this origin's /api/v1.
// All text is inserted with textContent (item and monster names come from the game).
(() => {
const ROOT = '/api/v1';
let API = ROOT; // instance base once an instance is selected: /api/v1/instances/{id}
let es = null;
const $ = (id) => document.getElementById(id);

function h(tag, props, ...kids) {
  const el = document.createElement(tag);
  for (const [k, v] of Object.entries(props || {})) {
    if (v == null || v === false) continue;
    if (k === 'class') el.className = v;
    else if (k === 'text') el.textContent = v;
    else if (k.startsWith('on')) el.addEventListener(k.slice(2), v);
    else if (v === true) el.setAttribute(k, '');
    else el.setAttribute(k, v);
  }
  for (const kid of kids.flat()) if (kid != null && kid !== false) el.append(kid.nodeType ? kid : document.createTextNode(String(kid)));
  return el;
}
const svgNS = 'http://www.w3.org/2000/svg';
function s(tag, attrs, ...kids) {
  const el = document.createElementNS(svgNS, tag);
  for (const [k, v] of Object.entries(attrs || {})) el.setAttribute(k, v);
  for (const kid of kids.flat()) if (kid) el.append(kid.nodeType ? kid : document.createTextNode(String(kid)));
  return el;
}
const clear = (el) => { while (el.firstChild) el.removeChild(el.firstChild); };

const state = {
  status: null, goals: [], stats: null, series: null, catalog: null, settings: null,
  tab: 'queue', log: [], lastSeq: 0, window: '6h', connected: false,
};

// ---- formatting ---------------------------------------------------------------------------------
const nf = new Intl.NumberFormat();
function compact(n) {
  n = Math.round(n || 0);
  if (Math.abs(n) >= 1e6) return (n / 1e6).toFixed(n >= 1e7 ? 0 : 1) + 'M';
  if (Math.abs(n) >= 1e4) return Math.round(n / 1e3) + 'k';
  if (Math.abs(n) >= 1e3) return (n / 1e3).toFixed(1) + 'k';
  return String(n);
}
function dur(sec) {
  sec = Math.max(0, Math.round(sec || 0));
  const hh = Math.floor(sec / 3600), mm = Math.floor(sec % 3600 / 60);
  if (hh >= 24) return Math.floor(hh / 24) + 'd ' + (hh % 24) + 'h';
  if (hh > 0) return hh + 'h ' + String(mm).padStart(2, '0') + 'm';
  if (mm > 0) return mm + 'm ' + String(sec % 60).padStart(2, '0') + 's';
  return sec + 's';
}
const clock = (t) => new Date(t).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false });
const ago = (t) => { const d = (Date.now() - new Date(t)) / 1000; return d < 5 ? 'now' : dur(d) + ' ago'; };
const pct = (a, b) => (b > 0 ? Math.max(0, Math.min(100, (a / b) * 100)) : 0);
const cap = (x) => x ? x[0].toUpperCase() + x.slice(1) : x;

// ---- API ----------------------------------------------------------------------------------------
async function api(method, path, body) {
  const opt = { method, headers: { 'Content-Type': 'application/json' }, credentials: 'same-origin' };
  if (body !== undefined) opt.body = JSON.stringify(body);
  let res;
  try { res = await fetch((path.startsWith('/api/') ? '' : API) + path, opt); } catch (e) { throw new Error('Cannot reach the bot. Is it running?'); }
  if (res.status === 204) return null;
  let data = null;
  try { data = await res.json(); } catch (e) { /* non-JSON */ }
  if (!res.ok) throw new Error((data && data.error) || res.status + ' ' + res.statusText);
  return data;
}
function banner(msg) { $('banner-text').textContent = msg; $('banner').hidden = !msg; }
async function act(p) { try { await p; banner(''); } catch (e) { banner(e.message); } await refreshGoals(); }
async function refreshGoals() { try { state.goals = await api('GET', '/goals'); renderGoals(); } catch (e) { /* stream will catch up */ } }

// ---- header -------------------------------------------------------------------------------------
function renderHeader() {
  const st = state.status;
  const mode = st ? st.mode : null;
  const err = st && st.runner && st.runner.state === 'error';
  $('mode-dot').className = 'dot ' + (err ? 'error' : mode || '');
  $('mode-text').textContent = !st ? (state.connected ? 'Loading' : 'Connecting') :
    err ? 'Needs attention' : { running: 'Running', paused: 'Paused', stopped: 'Stopped' }[mode];
  const c = $('controls');
  clear(c);
  if (!st) return;
  const b = (label, cls, fn, title) => h('button', { class: 'btn ' + cls, type: 'button', title, onclick: fn }, label);
  if (mode === 'stopped') c.append(b('Start', 'primary', () => act(api('POST', '/control/start')), 'Start playing the queue'));
  if (mode === 'paused') c.append(b('Resume', 'primary', () => act(api('POST', '/control/start')), 'Continue where it left off'));
  if (mode === 'running') c.append(b('Pause', '', () => act(api('POST', '/control/pause')), 'Freeze the game, keep the current goal'));
  if (mode !== 'stopped') c.append(b('Stop', 'danger', () => act(api('POST', '/control/stop')), 'Stop playing; the goal returns to the queue'));
  if (st.lastError) banner(st.lastError); else if (st.loadWarning && !state.warnShown) { state.warnShown = true; banner('Saved state: ' + st.loadWarning); }
}

// ---- now card -----------------------------------------------------------------------------------
function bar(kind, value, max, label) {
  const fill = h('i');
  fill.style.width = pct(value, max) + '%';
  return h('div', { class: 'bar-row' }, h('span', { class: 'muted', text: label }),
    h('div', { class: 'bar ' + kind, role: 'progressbar', 'aria-label': label, 'aria-valuemin': 0, 'aria-valuemax': max, 'aria-valuenow': value }, fill),
    h('span', { text: max ? nf.format(value) + ' / ' + nf.format(max) : nf.format(value) }));
}
function renderNow() {
  const root = $('now');
  clear(root);
  const st = state.status;
  if (!st) { root.append(h('p', { class: 'muted', text: 'Waiting for the bot...' })); return; }
  const r = st.runner, c = r.character;
  root.append(h('div', { class: 'char-name' }, h('strong', { text: c.name || 'No character' }),
    h('span', { class: 'muted', text: [c.class, c.level ? 'level ' + c.level : '', c.difficulty ? cap(c.difficulty) : ''].filter(Boolean).join(', ') })));
  root.append(h('p', { class: 'activity', text: st.mode === 'stopped' ? 'Idle, waiting in ' + (c.area || 'town') : (st.mode === 'paused' ? 'Paused: ' : '') + (r.activity || 'Idle') }));
  root.append(h('div', { class: 'bars' }, bar('life', c.life, c.maxLife, 'Life'), bar('mana', c.mana, c.maxMana, 'Mana'), bar('xp', c.xp, c.xpNext, 'XP')));
  root.append(h('dl', { class: 'kv' },
    h('dt', { text: 'Area' }), h('dd', { text: c.area || 'Unknown' }),
    h('dt', { text: 'Position' }), h('dd', { text: c.position ? c.position.x + ', ' + c.position.y : 'Unknown' }),
    h('dt', { text: 'XP per hour' }), h('dd', { text: nf.format(Math.round(st.xpPerHour || 0)) })));
  const g = st.activeGoal;
  const box = h('div', { class: 'cur-goal' });
  if (g) {
    box.append(h('div', { class: 'muted', text: 'Working on' }), h('div', { class: 'title', text: g.title }), goalProgress(g));
  } else {
    box.append(h('div', { class: 'muted', text: 'Working on' }), h('div', { text: st.mode === 'running' ? 'Nothing in the queue. Add a goal.' : 'Nothing, the bot is not running.' }));
  }
  root.append(box);
  if (r.message) root.append(h('p', { class: 'muted', text: r.message }));
}

function goalProgress(g) {
  const p = g.progress;
  const wrap = h('div');
  if (g.endless) {
    wrap.append(h('div', { class: 'goal-meta', text: 'No end condition, runs until you skip it. ' + progressText(g) }));
    return wrap;
  }
  const fill = h('i');
  fill.style.width = p.percent + '%';
  wrap.append(h('div', { class: 'bar', role: 'progressbar', 'aria-label': 'Progress of ' + g.title, 'aria-valuemin': 0, 'aria-valuemax': 100, 'aria-valuenow': Math.round(p.percent) }, fill));
  wrap.append(h('div', { class: 'goal-meta', text: Math.round(p.percent) + '%. ' + progressText(g) }));
  extras(wrap, g);
  return wrap;
}
// Optional fields from docs/goals/tracking.md: phase, ETA, verified badge and sub-steps. Absent = nothing shown.
function extras(wrap, g) {
  const p = g.progress, bits = [];
  if (p.phase) bits.push(p.phase);
  if (p.etaSeconds > 0) bits.push('about ' + dur(p.etaSeconds) + ' left');
  if (p.satisfied) bits.push('already true');
  if (bits.length || p.verified != null && p.source) {
    const row = h('div', { class: 'goal-meta', text: bits.join(', ') });
    if (p.source) row.append(h('span', { class: 'badge ' + (p.verified ? 'done' : ''), text: p.verified ? 'verified by game' : 'estimated' }));
    wrap.append(row);
  }
  if (p.steps && p.steps.length) {
    const ol = h('ol', { class: 'steps', 'aria-label': 'Steps of ' + g.title });
    for (const st of p.steps) {
      const frac = st.target ? ' ' + st.current + '/' + st.target : '';
      const mark = { done: 'done', active: 'active', failed: 'failed', skipped: 'skipped' }[st.state] || 'queued';
      ol.append(h('li', { class: 'step ' + mark }, h('span', { class: 'tick', 'aria-hidden': 'true', text: mark === 'done' ? '\u2713' : mark === 'failed' ? '\u00d7' : mark === 'active' ? '\u25b8' : '\u25cb' }),
        h('span', { text: (st.title || st.id) + frac }), h('span', { class: 'sr', text: ' (' + mark + ')' })));
    }
    wrap.append(ol);
  }
}
function progressText(g) {
  const p = g.progress, bits = [];
  switch (g.type) {
    case 'reach_level': bits.push('level ' + p.current + ' of ' + p.target); break;
    case 'kill_boss': bits.push(p.bossKills + ' of ' + p.target + ' kills'); break;
    case 'collect_item': bits.push(p.drops + ' of ' + p.target + ' found'); break;
    default: bits.push(nf.format(p.kills) + ' kills'); if (g.params.untilDrops || p.drops) bits.push(p.drops + ' drops');
  }
  if (p.deaths) bits.push(p.deaths + (p.deaths === 1 ? ' death' : ' deaths'));
  if (p.activeSeconds) bits.push(dur(p.activeSeconds) + ' played');
  return bits.join(', ');
}

// ---- tiles --------------------------------------------------------------------------------------
function renderTiles() {
  const t = (state.status && state.status.totals) || {};
  const items = [
    ['XP per hour', compact(state.status ? state.status.xpPerHour : 0)], ['Kills', compact(t.kills)], ['Drops', compact(t.drops)],
    ['Deaths', compact(t.deaths)], ['Potions used', compact(t.potions)], ['Time played', dur(t.activeSeconds)],
  ];
  const root = $('tiles');
  clear(root);
  for (const [l, v] of items) root.append(h('div', { class: 'tile' }, h('div', { class: 'v', text: v }), h('div', { class: 'l', text: l })));
}

// ---- goals --------------------------------------------------------------------------------------
const TYPE_NAME = { reach_level: 'Level up', farm_area: 'Farm', kill_boss: 'Boss', collect_item: 'Collect', idle: 'Rest' };
const DAYS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
function windowText(w) {
  const d = (w.days && w.days.length) ? w.days.map((x) => DAYS[x]).join(' ') + ' ' : 'Daily ';
  return d + w.start + ' to ' + w.end;
}
// Status strip: soak alerts (/api/v1/soak, 404 when the server has no -soak-dir) and goal notes.
let soak = null;
async function refreshSoak() {
  try { soak = await api('GET', ROOT + '/soak'); } catch (e) { soak = null; }
  renderStrip();
}
function renderStrip() {
  const noted = state.goals.filter((g) => g.note && !['done', 'skipped'].includes(g.state)).slice(-4);
  $('strip').hidden = !soak && !noted.length;
  $('strip-soak').hidden = !soak;
  const al = $('strip-alerts'), nl = $('strip-notes');
  clear(al); clear(nl);
  if (soak) {
    const lvl = ['ok', 'warn', 'crit'].includes(soak.level) ? soak.level : '';
    $('strip-level').className = 'badge ' + lvl;
    $('strip-level').textContent = 'Soak ' + (soak.level || 'unknown');
    $('strip-status').textContent = soak.error || soak.status || soak.run;
    $('strip-age').textContent = soak.ageSec > 300 ? 'stale, updated ' + dur(soak.ageSec) + ' ago' : '';
    (soak.alerts || []).forEach((a) => al.append(h('li', { class: a.level === 'crit' ? 'crit' : 'warn', text: (a.code ? a.code + ': ' : '') + a.msg })));
  }
  noted.forEach((g) => nl.append(h('li', { text: g.title + ': ' + g.note })));
}

function renderGoals() {
  renderStrip();
  const live = state.goals.filter((g) => !['done', 'failed', 'skipped'].includes(g.state));
  const fin = state.goals.filter((g) => ['done', 'failed', 'skipped'].includes(g.state));
  $('count-queue').textContent = live.length;
  $('count-done').textContent = fin.length;
  const list = $('goal-list');
  const keep = document.activeElement && list.contains(document.activeElement) ? document.activeElement.dataset.key : null;
  clear(list);
  const shown = state.tab === 'queue' ? live : fin;
  if (!shown.length) {
    list.append(h('li', { class: 'empty' }, state.tab === 'queue' ? 'Nothing queued. Choose "Add goal" and say what you want to achieve.' : 'Nothing finished yet.'));
    return;
  }
  shown.forEach((g, i) => list.append(goalItem(g, i, shown.length)));
  if (keep) { const el = list.querySelector('[data-key="' + CSS.escape(keep) + '"]'); if (el) el.focus(); }
}
function goalItem(g, i, n) {
  const A = (label, key, fn, cls, disabled) => h('button', { class: 'btn small ' + (cls || ''), type: 'button', 'data-key': g.id + key, disabled, 'aria-label': label + ': ' + g.title, onclick: fn }, label);
  const live = state.tab === 'queue';
  const meta = [TYPE_NAME[g.type]];
  if (g.priority > 0) meta.push('high priority'); else if (g.priority < 0) meta.push('low priority');
  if (g.params.difficulty) meta.push(cap(g.params.difficulty));
  (g.schedule || []).forEach((w) => meta.push(windowText(w)));
  const top = h('div', { class: 'goal-top' }, h('span', { class: 'goal-title', text: g.title }), h('span', { class: 'badge ' + g.state, text: g.state === 'queued' && live ? 'Waiting' : cap(g.state) }));
  const li = h('li', { class: 'goal ' + g.state }, top, h('div', { class: 'goal-meta', text: meta.join(' / ') }));
  if (g.note) li.append(h('div', { class: 'goal-meta', text: g.note }));
  li.append(goalProgress(g));
  const acts = h('div', { class: 'goal-actions' });
  if (live) {
    acts.append(A('Up', 'u', () => act(api('POST', '/goals/' + g.id + '/move', { position: i - 1 })), '', i === 0),
      A('Down', 'd', () => act(api('POST', '/goals/' + g.id + '/move', { position: i + 1 })), '', i === n - 1));
    if (g.state === 'paused') acts.append(A('Resume', 'p', () => act(api('POST', '/goals/' + g.id + '/resume'))));
    else acts.append(A('Hold', 'p', () => act(api('POST', '/goals/' + g.id + '/pause'))));
    acts.append(A('Skip', 's', () => act(api('POST', '/goals/' + g.id + '/skip'))), A('Edit', 'e', () => openGoalDialog(g)));
  } else {
    acts.append(A('Run again', 'r', () => act(api('POST', '/goals/' + g.id + '/reset'))));
  }
  acts.append(removeButton(g, A));
  li.append(acts);
  return li;
}
function removeButton(g, A) {
  const armed = state.armed && state.armed.id === g.id && Date.now() < state.armed.until;
  const b = A(armed ? 'Really remove?' : 'Remove', 'x', () => {
    if (armed) { state.armed = null; return act(api('DELETE', '/goals/' + g.id)); }
    state.armed = { id: g.id, until: Date.now() + 4000 };
    renderGoals();
    setTimeout(renderGoals, 4100);
  }, armed ? 'danger' : '');
  return b;
}
$('tab-queue').onclick = () => setTab('queue');
$('tab-done').onclick = () => setTab('done');
function setTab(t) {
  state.tab = t;
  $('tab-queue').setAttribute('aria-selected', t === 'queue');
  $('tab-done').setAttribute('aria-selected', t === 'done');
  renderGoals();
}

// ---- goal form ----------------------------------------------------------------------------------
const TYPES = [
  ['reach_level', 'Reach a level', 'Play until the character is level N.'],
  ['farm_area', 'Farm an area', 'Stay in one place or run, as long as you like.'],
  ['kill_boss', 'Defeat a boss', 'Kill a named boss one or more times.'],
  ['collect_item', 'Find an item', 'Keep going until enough matching items drop.'],
  ['idle', 'Take a break', 'Rest in town for a while, or in a time window.'],
];
let editing = null;
const val = (id) => $(id).value.trim();
const num = (id) => { const n = parseInt($(id).value, 10); return Number.isFinite(n) && n > 0 ? n : 0; };

function field(id, label, attrs, hint) {
  return h('label', {}, label, h('input', Object.assign({ id }, attrs)), hint ? h('span', { class: 'hint', text: hint }) : null);
}
function selectField(id, label, options) {
  return h('label', {}, label, h('select', { id }, options.map(([v, t]) => h('option', { value: v, text: t }))));
}
function typeFields(type) {
  const c = state.catalog || { areas: [], runs: [], bosses: [], itemTypes: [], qualities: [] };
  const q = [['any', 'Any quality'], ...c.qualities.filter((x) => x !== 'any').map((x) => [x, cap(x)])];
  const box = h('div', { class: 'fields' });
  if (type === 'reach_level') box.append(field('f-level', 'Target level', { type: 'number', min: 1, max: 99, inputmode: 'numeric' }, 'The bot picks where to level and moves on as you grow.'));
  if (type === 'farm_area') {
    box.append(field('f-area', 'Where', { type: 'text', list: 'dl-areas', placeholder: 'Pick an area or a run, or type a name', autocomplete: 'off' }));
    box.append(h('p', { class: 'hint', text: 'Stop after (leave all empty to keep farming until you skip it):' }));
    box.append(h('div', { class: 'grid2' }, field('f-kills', 'Kills', { type: 'number', min: 0, inputmode: 'numeric' }), field('f-mins', 'Minutes', { type: 'number', min: 0, inputmode: 'numeric' })));
    box.append(field('f-drops', 'Drops', { type: 'number', min: 0, inputmode: 'numeric' }, 'Counts any drop, or only the item filter below if you set one.'));
    box.append(h('div', { class: 'grid2' }, selectField('f-iquality', 'Only count quality', q), field('f-iname', 'Only count name', { type: 'text', placeholder: 'e.g. rune' })));
  }
  if (type === 'kill_boss') {
    box.append(field('f-boss', 'Which boss', { type: 'text', list: 'dl-bosses', placeholder: 'e.g. Mephisto', autocomplete: 'off' }));
    box.append(field('f-count', 'How many times', { type: 'number', min: 1, value: 1, inputmode: 'numeric' }));
  }
  if (type === 'collect_item') {
    box.append(h('div', { class: 'grid2' }, field('f-itype', 'Kind of item', { type: 'text', list: 'dl-itemtypes', placeholder: 'rune, charm, ring...', autocomplete: 'off' }), selectField('f-iquality', 'Quality', q)));
    box.append(field('f-iname', 'Name contains (optional)', { type: 'text', placeholder: 'e.g. Tir' }));
    box.append(field('f-count', 'How many', { type: 'number', min: 1, value: 1, inputmode: 'numeric' }));
  }
  if (type === 'idle') box.append(field('f-idle', 'Rest for how many minutes', { type: 'number', min: 0, inputmode: 'numeric' }, 'Leave empty and set a time window below to rest every day in that window.'));
  return box;
}
function currentType() { const r = document.querySelector('input[name=gtype]:checked'); return r ? r.value : 'reach_level'; }
function setType(type) {
  const holder = $('type-fields');
  clear(holder);
  holder.append(typeFields(type));
  if (type === 'idle') { $('f-priority').value = editing ? $('f-priority').value : '1'; }
  holder.querySelectorAll('input,select').forEach((e) => e.addEventListener('input', updatePreview));
  updatePreview();
}
function buildGoal() {
  const type = currentType();
  const p = {};
  const item = () => ({ quality: $('f-iquality').value === 'any' ? '' : $('f-iquality').value, name: val('f-iname'), type: $('f-itype') ? val('f-itype') : '' });
  switch (type) {
    case 'reach_level': p.level = num('f-level'); break;
    case 'farm_area': {
      const a = val('f-area'); const run = ((state.catalog || {}).runs || []).find((r) => r.label.toLowerCase() === a.toLowerCase());
      p.area = run ? run.key : a; p.untilKills = num('f-kills'); p.untilMinutes = num('f-mins'); p.untilDrops = num('f-drops');
      const it = item(); if (it.quality || it.name) p.item = it; break;
    }
    case 'kill_boss': p.boss = val('f-boss'); p.count = num('f-count') || 1; break;
    case 'collect_item': p.item = item(); p.count = num('f-count') || 1; break;
    case 'idle': p.minutes = num('f-idle'); break;
  }
  if (val('f-difficulty')) p.difficulty = val('f-difficulty');
  let schedule = [];
  if ($('f-win-on').checked) {
    const days = [...document.querySelectorAll('#f-days input:checked')].map((x) => +x.value);
    schedule.push({ days, start: $('f-win-start').value, end: $('f-win-end').value });
  }
  if (editing && editing.schedule && editing.schedule.length > 1) schedule = schedule.concat(editing.schedule.slice(1));
  return { type, title: val('f-title'), params: p, priority: +$('f-priority').value, schedule };
}
function updatePreview() {
  const g = buildGoal(), p = g.params;
  let t = '';
  switch (g.type) {
    case 'reach_level': t = p.level ? 'Play until the character reaches level ' + p.level + '.' : 'Choose a target level.'; break;
    case 'farm_area': {
      const lim = [p.untilKills && p.untilKills + ' kills', p.untilMinutes && p.untilMinutes + ' minutes', p.untilDrops && p.untilDrops + ' drops'].filter(Boolean);
      t = p.area ? 'Farm ' + areaLabel(p.area) + (lim.length ? ' until ' + lim.join(' or ') + '.' : ' until you skip it.') : 'Choose where to farm.'; break;
    }
    case 'kill_boss': t = p.boss ? 'Kill ' + p.boss + (p.count > 1 ? ' ' + p.count + ' times.' : ' once.') : 'Choose a boss.'; break;
    case 'collect_item': t = 'Find ' + p.count + ' x ' + ([p.item.quality, p.item.name || p.item.type].filter(Boolean).join(' ') || 'item') + '.'; break;
    case 'idle': t = p.minutes ? 'Rest for ' + p.minutes + ' minutes.' : (g.schedule.length ? 'Rest during the time window.' : 'Set minutes or a time window.'); break;
  }
  if (g.schedule.length) t += ' Only ' + windowText(g.schedule[0]).toLowerCase() + '.';
  $('goal-preview').textContent = t;
}
function areaLabel(a) { const r = ((state.catalog || {}).runs || []).find((x) => x.key === a); return r ? r.label : a; }

function openGoalDialog(g) {
  editing = g || null;
  $('goal-dialog-title').textContent = g ? 'Edit goal' : 'What do you want to achieve?';
  $('goal-save').textContent = g ? 'Save changes' : 'Add to queue';
  $('goal-error').hidden = true;
  const pick = $('type-pick');
  clear(pick);
  pick.append(h('legend', { class: 'sr', text: 'Goal type' }));
  for (const [v, name, desc] of TYPES) {
    if (g && g.type !== v) continue; // the type of an existing goal cannot change
    pick.append(h('label', { class: 'type-card' }, h('input', { type: 'radio', name: 'gtype', value: v, checked: g ? g.type === v : v === 'reach_level', onchange: () => setType(v) }),
      h('span', {}, h('b', { text: name }), h('small', { text: desc }))));
  }
  const days = $('f-days');
  clear(days);
  DAYS.forEach((d, i) => days.append(h('label', { class: 'check' }, h('input', { type: 'checkbox', value: i, onchange: updatePreview }), d)));
  $('f-priority').value = g ? String(g.priority > 0 ? 1 : g.priority < 0 ? -1 : 0) : '0';
  $('f-difficulty').value = g ? g.params.difficulty || '' : '';
  $('f-title').value = g && g.title !== autoTitleGuess(g) ? g.title : '';
  const w = g && g.schedule && g.schedule[0];
  $('f-win-on').checked = !!w;
  $('win-fields').hidden = !w;
  if (w) { $('f-win-start').value = w.start; $('f-win-end').value = w.end; (w.days || []).forEach((d) => { days.querySelector('input[value="' + d + '"]').checked = true; }); }
  $('advanced').open = !!(g && (w || g.priority || g.params.difficulty));
  setType(g ? g.type : 'reach_level');
  if (g) fillFields(g);
  updatePreview();
  $('goal-dialog').showModal();
  const first = $('type-fields').querySelector('input,select');
  if (first) first.focus();
}
function autoTitleGuess() { return null; }
function fillFields(g) {
  const p = g.params, set = (id, v) => { const e = $(id); if (e && v) e.value = v; };
  set('f-level', p.level); set('f-area', areaLabel(p.area || '')); set('f-kills', p.untilKills); set('f-mins', p.untilMinutes); set('f-drops', p.untilDrops);
  set('f-boss', p.boss); set('f-count', p.count); set('f-idle', p.minutes);
  if (p.item) { set('f-iquality', p.item.quality); set('f-iname', p.item.name); set('f-itype', p.item.type); }
}
$('btn-add').onclick = () => openGoalDialog(null);
$('goal-cancel').onclick = () => $('goal-dialog').close();
$('f-win-on').onchange = () => { $('win-fields').hidden = !$('f-win-on').checked; updatePreview(); };
['f-priority', 'f-difficulty', 'f-title', 'f-win-start', 'f-win-end'].forEach((id) => $(id).addEventListener('input', updatePreview));
$('goal-form').addEventListener('submit', async (ev) => {
  ev.preventDefault();
  const g = buildGoal();
  try {
    if (editing) await api('PATCH', '/goals/' + editing.id, { title: g.title, params: g.params, priority: g.priority, schedule: g.schedule });
    else await api('POST', '/goals', g);
    $('goal-dialog').close();
    banner('');
    await refreshGoals();
  } catch (e) { const el = $('goal-error'); el.textContent = e.message; el.hidden = false; }
});

// ---- settings -----------------------------------------------------------------------------------
$('btn-settings').onclick = async () => {
  try { state.settings = await api('GET', '/settings'); } catch (e) { banner(e.message); return; }
  const s = state.settings;
  $('s-auto').checked = s.autoResume; $('s-deaths').value = s.maxDeaths; $('s-chicken').value = s.chickenLifePct;
  $('s-potion').value = s.potionLifePct; $('s-diff').value = s.difficulty;
  $('settings-error').hidden = true;
  $('settings-dialog').showModal();
};
$('settings-cancel').onclick = () => $('settings-dialog').close();
$('settings-form').addEventListener('submit', async (ev) => {
  ev.preventDefault();
  try {
    await api('PUT', '/settings', { autoResume: $('s-auto').checked, maxDeaths: +$('s-deaths').value || 0, chickenLifePct: +$('s-chicken').value || 0, potionLifePct: +$('s-potion').value || 0, difficulty: $('s-diff').value });
    $('settings-dialog').close();
  } catch (e) { const el = $('settings-error'); el.textContent = e.message; el.hidden = false; }
});

// ---- log ----------------------------------------------------------------------------------------
const MAX_LOG = 300;
function addLog(ev) {
  if (ev.seq <= state.lastSeq) return;
  state.lastSeq = ev.seq;
  state.log.push(ev);
  if (state.log.length > MAX_LOG) state.log.splice(0, state.log.length - MAX_LOG);
}
function renderLog() {
  const list = $('log-list');
  const hide = $('log-hide-kills').checked;
  clear(list);
  let n = 0;
  for (let i = state.log.length - 1; i >= 0 && n < 120; i--) {
    const e = state.log[i];
    if (hide && e.type === 'kill' && !e.boss) continue;
    n++;
    list.append(h('li', { class: e.type }, h('time', { datetime: e.time, text: clock(e.time) }), h('span', { text: e.message })));
  }
  if (!n) list.append(h('li', {}, h('span', {}), h('span', { class: 'muted', text: 'Nothing yet.' })));
}
$('log-hide-kills').onchange = renderLog;

// ---- drops and areas ----------------------------------------------------------------------------
const QUALITIES = ['unique', 'set', 'rare', 'crafted', 'magic', 'normal'];
function renderDrops() {
  const st = state.stats;
  const bars = $('drops-bars'), list = $('drop-list');
  clear(bars); clear(list);
  if (!st) return;
  const q = st.qualities || {};
  const max = Math.max(1, ...QUALITIES.map((k) => q[k] || 0));
  const rows = h('div', { class: 'qbars', role: 'list', 'aria-label': 'Drops by quality' });
  for (const k of QUALITIES) {
    const fill = h('i'); fill.style.width = pct(q[k] || 0, max) + '%'; fill.style.background = 'var(--q-' + k + ')';
    rows.append(h('div', { class: 'qrow', role: 'listitem' }, h('span', { class: 'q-' + k, text: cap(k) }), h('div', { class: 'qbar' }, fill), h('span', { class: 'n', text: nf.format(q[k] || 0) })));
  }
  bars.append(rows);
  const drops = (st.drops || []).filter((d) => d.item.quality !== 'normal' || d.item.type === 'rune').slice(0, 8);
  if (!drops.length) list.append(h('li', { class: 'muted', text: 'No notable finds yet.' }));
  for (const d of drops) list.append(h('li', {}, h('span', { class: 'q-' + (d.item.quality || 'normal'), text: d.item.name }), h('span', { class: 'qtag', text: d.item.quality || 'normal' }), h('span', { class: 't', text: ago(d.time) })));
}
function renderAreas() {
  const list = $('area-list');
  clear(list);
  const a = (state.stats && state.stats.areas) || [];
  if (!a.length) list.append(h('li', { class: 'muted', text: 'No history yet.' }));
  a.slice(0, 12).forEach((v, i) => list.append(h('li', {}, h('span', { text: v.area }), h('span', { class: 'muted', text: (i === 0 ? 'now, ' : '') + dur(v.seconds) }))));
}

// ---- XP chart -----------------------------------------------------------------------------------
function niceMax(v) {
  if (v <= 0) return 10;
  const e = Math.pow(10, Math.floor(Math.log10(v))), f = v / e;
  return (f <= 1 ? 1 : f <= 2 ? 2 : f <= 5 ? 5 : 10) * e;
}
let chartSel = -1;
function renderChart() {
  const root = $('xp-chart');
  clear(root);
  const sr = state.series;
  if (!sr) { root.append(h('p', { class: 'muted', text: 'Loading...' })); return; }
  const pts = sr.points;
  const W = 640, H = 230, m = { l: 48, r: 12, t: 12, b: 26 };
  const iw = W - m.l - m.r, ih = H - m.t - m.b;
  const ymax = niceMax(Math.max(...pts.map((p) => p.xpPerHour)));
  const x = (i) => m.l + (pts.length === 1 ? iw / 2 : (i / (pts.length - 1)) * iw);
  const y = (v) => m.t + ih - (v / ymax) * ih;
  const svg = s('svg', { viewBox: '0 0 ' + W + ' ' + H, role: 'img', tabindex: 0, 'aria-label': 'Experience per hour over ' + $('xp-window').selectedOptions[0].text.toLowerCase() + '. Use left and right arrow keys to read values.' });
  const grid = s('g', { class: 'grid' }), axis = s('g', { class: 'axis' });
  for (let i = 0; i <= 4; i++) {
    const v = (ymax / 4) * i;
    grid.append(s('line', { x1: m.l, x2: W - m.r, y1: y(v), y2: y(v) }));
    axis.append(s('text', { x: m.l - 6, y: y(v) + 4, 'text-anchor': 'end' }, compact(v)));
  }
  const nt = Math.min(5, pts.length);
  for (let k = 0; k < nt; k++) {
    const i = Math.round((k / Math.max(1, nt - 1)) * (pts.length - 1));
    const d = new Date(pts[i].t * 1000);
    const lbl = sr.window > 86400 * 1.5 ? d.toLocaleDateString([], { weekday: 'short' }) : d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', hour12: false });
    axis.append(s('text', { x: x(i), y: H - 6, 'text-anchor': k === 0 ? 'start' : k === nt - 1 ? 'end' : 'middle' }, lbl));
  }
  const line = pts.map((p, i) => (i ? 'L' : 'M') + x(i).toFixed(1) + ' ' + y(p.xpPerHour).toFixed(1)).join('');
  svg.append(grid, axis, s('path', { class: 'series-area', d: line + 'L' + x(pts.length - 1) + ' ' + y(0) + 'L' + x(0) + ' ' + y(0) + 'Z' }), s('path', { class: 'series-line', d: line }));
  const cross = s('line', { class: 'cross', y1: m.t, y2: m.t + ih, visibility: 'hidden' }), mark = s('circle', { class: 'mark', r: 4.5, visibility: 'hidden' });
  svg.append(cross, mark);
  const tip = $('tip');
  const select = (i, cx, cy) => {
    chartSel = i;
    const p = pts[i];
    cross.setAttribute('x1', x(i)); cross.setAttribute('x2', x(i)); cross.setAttribute('visibility', 'visible');
    mark.setAttribute('cx', x(i)); mark.setAttribute('cy', y(p.xpPerHour)); mark.setAttribute('visibility', 'visible');
    const d = new Date(p.t * 1000);
    tip.textContent = d.toLocaleString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit', hour12: false }) + ': ' + nf.format(Math.round(p.xpPerHour)) + ' XP/h, ' + p.kills + ' kills, ' + p.drops + ' drops';
    tip.hidden = false;
    const r = svg.getBoundingClientRect();
    const px = cx != null ? cx : r.left + (x(i) / W) * r.width, py = cy != null ? cy : r.top + (y(p.xpPerHour) / H) * r.height;
    tip.style.left = Math.min(window.innerWidth - tip.offsetWidth - 8, Math.max(8, px + 12)) + 'px';
    tip.style.top = Math.max(8, py - tip.offsetHeight - 12) + 'px';
  };
  const hide = () => { cross.setAttribute('visibility', 'hidden'); mark.setAttribute('visibility', 'hidden'); tip.hidden = true; chartSel = -1; };
  const idxAt = (ev) => { const r = svg.getBoundingClientRect(); const px = ((ev.clientX - r.left) / r.width) * W; return Math.max(0, Math.min(pts.length - 1, Math.round(((px - m.l) / iw) * (pts.length - 1)))); };
  svg.addEventListener('pointermove', (ev) => select(idxAt(ev), ev.clientX, ev.clientY));
  svg.addEventListener('pointerleave', hide);
  svg.addEventListener('blur', hide);
  svg.addEventListener('keydown', (ev) => {
    const k = { ArrowLeft: -1, ArrowRight: 1, Home: -1e9, End: 1e9 }[ev.key];
    if (!k) { if (ev.key === 'Escape') hide(); return; }
    ev.preventDefault();
    select(Math.max(0, Math.min(pts.length - 1, (chartSel < 0 ? pts.length - 1 : chartSel) + k)));
  });
  root.append(svg);
  if (!pts.some((p) => p.xp > 0)) root.append(h('p', { class: 'muted', text: 'No experience in this range yet.' }));

  const tb = $('xp-table');
  clear(tb);
  const rows = pts.slice().reverse().map((p) => h('tr', {}, h('td', { text: new Date(p.t * 1000).toLocaleString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit', hour12: false }) }), h('td', { text: nf.format(Math.round(p.xpPerHour)) }), h('td', { text: p.kills }), h('td', { text: p.drops }), h('td', { text: p.deaths })));
  tb.append(h('div', { class: 'tbl' }, h('table', {}, h('thead', {}, h('tr', {}, ['Time', 'XP/h', 'Kills', 'Drops', 'Deaths'].map((t) => h('th', { text: t })))), h('tbody', {}, rows))));
}
$('xp-window').onchange = () => { state.window = $('xp-window').value; refreshSeries(); };

// ---- data loading -------------------------------------------------------------------------------
async function refreshSeries() { try { state.series = await api('GET', '/stats/series?window=' + state.window); renderChart(); } catch (e) { /* retry on next poll */ } }
async function refreshStats() { try { state.stats = await api('GET', '/stats'); renderDrops(); renderAreas(); } catch (e) { /* retry */ } }

function applyStatus(st) { state.status = st; renderHeader(); renderNow(); renderTiles(); emit('status', st); }

// Instance streams send envelopes {seq,time,kind,data}; the flat compat stream sends goals.Event. The UI uses the flat shape.
function fromEnvelope(e) {
  if (e.type) return e;
  const d = e.data || {}, kind = String(e.kind || '');
  const ev = { seq: e.seq, time: e.time, type: kind.replace('.', '_'), goalId: e.goal, boss: !!d.boss, message: d.message || '' };
  const where = d.areaName || (d.area ? 'area ' + d.area : '');
  switch (kind) {
    case 'kill': ev.message = 'Killed ' + (d.monster || 'a monster'); break;
    case 'drop': ev.message = 'Found ' + ((d.item && d.item.name) || 'an item'); break;
    case 'death': ev.message = 'Died' + (where ? ' in ' + where : '') + (d.killer ? ' to ' + d.killer : ''); break;
    case 'level_up': ev.message = 'Reached level ' + d.level; break;
    case 'area_change': ev.message = 'Entered ' + where; break;
    case 'potion': ev.message = 'Drank ' + (d.name || 'a potion'); break;
    case 'failsafe': ev.message = 'Failsafe ' + [d.layer, d.rule, d.action, d.result].filter(Boolean).join(' / '); break;
    case 'lifecycle': ev.message = 'Game ' + (d.state || '') + (d.reason ? ': ' + d.reason : ''); break;
    case 'activity': ev.message = d.detail || d.state || ''; break;
    default: if (!ev.message) ev.message = kind.replace(/[._]/g, ' ') + (d.quest ? ': ' + d.quest : '');
  }
  return ev;
}

function connect() {
  if (es) es.close();
  const my = es = new EventSource(API + '/events/stream');
  const live = (fn) => (e) => { if (es === my) fn(e); };
  my.addEventListener('open', live(() => { state.connected = true; if (!state.status) renderHeader(); }));
  my.addEventListener('status', live((e) => { applyStatus(JSON.parse(e.data)); }));
  my.addEventListener('goals', live((e) => { state.goals = JSON.parse(e.data); renderGoals(); emit('goals', state.goals); }));
  my.addEventListener('log', live((e) => { const ev = fromEnvelope(JSON.parse(e.data)); addLog(ev); scheduleLog(); emit('log', ev); }));
  my.addEventListener('error', live(() => {
    state.connected = false;
    $('mode-dot').className = 'dot error'; $('mode-text').textContent = 'Connection lost, retrying';
  }));
}
let logTimer = 0;
function scheduleLog() { if (!logTimer) logTimer = setTimeout(() => { logTimer = 0; renderLog(); }, 250); }

// ---- theme --------------------------------------------------------------------------------------
function applyTheme(t) {
  if (t === 'light' || t === 'dark') document.documentElement.dataset.theme = t; else delete document.documentElement.dataset.theme;
  $('btn-theme').textContent = 'Theme: ' + (t === 'light' || t === 'dark' ? t : 'auto');
  renderChart();
}
let theme = 'auto';
try { theme = localStorage.getItem('d2theme') || 'auto'; } catch (e) { /* storage blocked */ }
$('btn-theme').onclick = () => {
  theme = { auto: 'light', light: 'dark', dark: 'auto' }[theme] || 'auto';
  try { localStorage.setItem('d2theme', theme); } catch (e) { /* ignore */ }
  applyTheme(theme);
};
$('banner-close').onclick = () => banner('');

// ---- boot ---------------------------------------------------------------------------------------
// ---- instances ----------------------------------------------------------------------------------
// Panels (panels.js) register with D2.on('instance'|'status'|'goals'|'log', fn) and call D2.api for instance-relative paths.
const hooks = {};
function emit(name, arg) { for (const fn of hooks[name] || []) { try { fn(arg); } catch (e) { console.error(e); } } }
const D2 = window.D2 = {
  h, s, $, clear, api, nf, compact, dur, cap, clock, state, banner, act,
  instance: null, instances: [], multi: false,
  on(name, fn) { (hooks[name] = hooks[name] || []).push(fn); },
  get base() { return API; },
};

async function discover() {
  try {
    const r = await fetch(ROOT + '/instances', { credentials: 'same-origin' });
    if (!r.ok) return null;
    const a = await r.json();
    return Array.isArray(a) && a.length ? a : null;
  } catch (e) { return null; }
}
function wantedInstance(list) {
  let id = new URLSearchParams(location.hash.slice(1)).get('i');
  if (!id) { try { id = localStorage.getItem('d2inst'); } catch (e) { /* ignore */ } }
  return list.some((x) => x.id === id) ? id : list[0].id;
}
async function loadCatalog() {
  try {
    state.catalog = await api('GET', '/catalog');
    const fill = (id, items) => { const dl = $(id); clear(dl); items.forEach((t) => dl.append(h('option', { value: t }))); };
    fill('dl-areas', [...state.catalog.runs.map((r) => r.label), ...state.catalog.areas.map((a) => a.name)]);
    fill('dl-bosses', state.catalog.bosses.map((b) => b.name));
    fill('dl-itemtypes', state.catalog.itemTypes);
  } catch (e) { banner(e.message); }
}
async function selectInstance(id) {
  D2.instance = id;
  API = id ? ROOT + '/instances/' + encodeURIComponent(id) : ROOT;
  Object.assign(state, { status: null, goals: [], stats: null, series: null, log: [], lastSeq: 0, connected: false, warnShown: false });
  if (id) {
    try { localStorage.setItem('d2inst', id); } catch (e) { /* ignore */ }
    const p = new URLSearchParams(location.hash.slice(1)); p.set('i', id); history.replaceState(null, '', '#' + p);
  }
  renderHeader(); renderNow(); renderTiles(); renderGoals(); renderLog(); renderDrops(); renderAreas(); renderChart();
  emit('instance', id);
  await loadCatalog();
  connect();
  refreshGoals(); refreshStats(); refreshSeries();
}
D2.select = (id) => { if (id !== D2.instance) selectInstance(id); };

async function boot() {
  applyTheme(theme);
  renderNow(); renderTiles(); renderLog(); renderChart();
  const list = await discover();
  D2.multi = !!list; D2.instances = list || [];
  emit('discovered', list);
  await selectInstance(list ? wantedInstance(list) : null);
  refreshSoak();
  setInterval(refreshSoak, 30000);
  setInterval(refreshStats, 10000);
  setInterval(refreshSeries, 15000);
  setInterval(() => { if (state.stats) renderDrops(); }, 30000);
}
boot();
})();