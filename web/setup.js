'use strict';
// D2 Idle first-run setup wizard. No framework, no network except this origin's /api/setup (docs/setup.md).
// All text is inserted with textContent. There is deliberately no CD key field anywhere.
(() => {
const API = '/api/setup';
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
const clear = (el) => { while (el.firstChild) el.removeChild(el.firstChild); };

const FRIENDLY = {
  'data-dir': 'Pick a folder for D.idle',
  'sources': 'Find your game files',
  'prefix': 'Prepare the game environment',
  'install-d2': 'Install Diablo II',
  'install-lod': 'Install Lord of Destruction',
  'import-game': 'Copy your Diablo II install',
  'patch': 'Update to version 1.14d',
  'extract': 'Read the game data',
  'character': 'Create a test character',
};
const WORKING = {
  'data-dir': 'Setting up your folder',
  'sources': 'Looking through your game files',
  'prefix': 'Preparing the game environment. This is a one-time job and can take a few minutes',
  'install-d2': 'Starting the Diablo II installer',
  'install-lod': 'Starting the Lord of Destruction installer',
  'import-game': 'Copying your Diablo II folder. Large folders take a while',
  'patch': 'Starting the 1.14d patcher',
  'extract': 'Reading the game data tables',
  'character': 'Creating your test character in the game',
};
const MODES = [
  { id: 'installer-dirs', title: 'I have the Diablo II installer folders', sub: 'Folders you downloaded or copied from your discs, one for the base game and one for Lord of Destruction.' },
  { id: 'stubs', title: 'I have the Blizzard downloader programs', sub: 'The small setup programs from your Blizzard account. They run in the game environment and download from Blizzard.' },
  { id: 'game-dir', title: 'Diablo II is already installed on this computer', sub: 'D.idle copies your existing install into its own folder. Your original is not changed.' },
];
const CLASSES = ['sorceress', 'paladin', 'necromancer', 'amazon', 'barbarian', 'druid', 'assassin'];

let S = null;            // last /state
let sKey = '';
let manual = null;       // screen the user navigated to by hand: welcome | checks | sources
let checks = null;       // last /checks report
let checksBusy = false, checksErr = '', checksFull = false;
let form = { mode: '', dataDir: '', installerD2Dir: '', installerLoDDir: '', stubD2: '', stubLoD: '', gameDirSrc: '', patchFile: '', slots: 1, charName: '', charClass: 'sorceress', handicap: '', pKind: 'campaign:normal_act1', pLevel: '30', pFarm: '' };
// Goal presets, sent as "goals" (fleet.Spec.GoalsArg translates kind:value into the d2bot --goals syntax).
// Handicap names are bot.Profiles() and go in "profile" (fleet.Spec.Profile).
const HANDICAPS = [['', 'Optimal (default)'], ['relaxed', 'Relaxed'], ['casual', 'Casual'], ['sloppy', 'Sloppy'], ['rookie', 'Rookie']];
const GOAL_PRESETS = [['', 'Let D.idle decide (default)'], ['campaign:normal_act1', 'Campaign: Normal, Act 1'], ['campaign:normal_act2', 'Campaign: Normal, Act 2 only'], ['campaign:normal_all', 'Campaign: every Normal act'], ['levelto', 'Level up to a level'], ['farm', 'Farm an area']];
function goalsValue(f) {
  if (f.pKind === 'levelto') { const n = parseInt(f.pLevel, 10); return n >= 2 && n <= 99 ? 'levelto:' + n : null; }
  if (f.pKind === 'farm') { const a = (f.pFarm || '').trim(); return /^[A-Za-z0-9_.-]{1,40}$/.test(a) ? 'farm:' + a : null; }
  return f.pKind;
}
let formErr = {};        // field -> message
let showFullLog = false, confirmReset = false;
let offline = false, screen = '', renderKey = '', poller = null;
let announced = '';

async function api(method, path, body) {
  try {
    const r = await fetch(API + path, { method, headers: body ? { 'Content-Type': 'application/json' } : undefined, body: body ? JSON.stringify(body) : undefined, cache: 'no-store' });
    let data = null;
    try { data = await r.json(); } catch (e) { /* empty body */ }
    offline = false;
    return { ok: r.ok, status: r.status, data };
  } catch (e) {
    offline = true;
    return { ok: false, status: 0, data: { error: 'Cannot reach D.idle. Is it still running? This page will keep trying.' } };
  }
}

// ---- state helpers -------------------------------------------------------------------------------
const step = (id) => S && S.steps.find((x) => x.id === id);
const cur = () => (S && S.current ? step(S.current) : null);
const anyActive = () => S && S.steps.some((x) => ['running', 'waiting-user', 'failed'].includes(x.status));
const anyProgress = () => S && S.steps.some((x) => ['done', 'skipped'].includes(x.status));
const hasInputs = () => S && S.inputs && S.inputs.mode && S.inputs.dataDir;

function autoScreen() {
  if (!S) return 'loading';
  if (S.done) return 'done';
  if (S.running || anyActive()) return 'progress';
  const c = cur();
  if (c && c.id === 'character' && c.status === 'pending') return 'character';
  if (anyProgress() && hasInputs()) return 'progress';
  return null;
}
function pickScreen() {
  const a = autoScreen();
  if (a === 'loading' || a === 'done' || a === 'progress' || a === 'character') {
    if (a === 'progress' && manual === 'sources') return 'sources'; // editing answers after a failure
    return a;
  }
  if (manual) return manual;
  return hasInputs() ? 'sources' : 'welcome';
}
function go(name) {
  manual = ['welcome', 'checks', 'sources'].includes(name) ? name : null;
  if (name === 'checks' && !checks && !checksBusy) runChecks(false);
  try { history.pushState(null, '', '#' + name); } catch (e) { /* ignore */ }
  render(true);
}
window.addEventListener('popstate', () => { const n = location.hash.slice(1); manual = ['welcome', 'checks', 'sources'].includes(n) ? n : null; render(true); });

// ---- polling -------------------------------------------------------------------------------------
async function refresh() {
  const r = await api('GET', '/state');
  if (r.ok && r.data) {
    const k = JSON.stringify(r.data);
    if (k !== sKey) { const first = !S; sKey = k; S = r.data; syncForm(); if (first && autoScreen()) manual = null; } // a stale #hash must not hide a run in progress
  }
  render(false);
  schedule();
}
function schedule() {
  clearTimeout(poller);
  const fast = offline || (S && (S.running || anyActive()));
  poller = setTimeout(refresh, fast ? 1200 : 5000);
}
function syncForm() {
  const i = (S && S.inputs) || {};
  for (const k of Object.keys(form)) if (i[k] !== undefined && i[k] !== '' && i[k] !== 0 && !dirty[k]) form[k] = i[k];
  if (!form.mode) form.mode = 'installer-dirs';
}
const dirty = {};

// ---- actions -------------------------------------------------------------------------------------
async function runChecks(full) {
  checksBusy = true; checksErr = ''; checksFull = full; render(true);
  const pre = S && step('install-d2') && step('install-d2').status === 'done' ? '' : 'pre=1';
  const r = await api('GET', '/checks?' + [pre, full ? '' : 'quick=1'].filter(Boolean).join('&'));
  checksBusy = false;
  if (r.ok) checks = r.data; else checksErr = (r.data && r.data.error) || 'The check could not run.';
  render(true);
}
async function startRun() {
  const r = await api('POST', '/start');
  if (!r.ok && r.status !== 202) { toast((r.data && r.data.error) || 'Could not start.'); }
  manual = null; showFullLog = false;
  await refresh();
}
async function retryStep(id) {
  await api('POST', '/reset?step=' + encodeURIComponent(id));
  await startRun();
}
async function cancelRun() { await api('POST', '/cancel'); await refresh(); }
async function startOver() {
  confirmReset = false;
  const r = await api('POST', '/reset');
  if (!r.ok) toast((r.data && r.data.error) || 'Could not reset.');
  manual = 'welcome'; checks = null;
  await refresh();
}
function toast(msg) { $('wz-live').textContent = msg; formErr._top = msg; render(true); }

const ABS = /^(\/|[A-Za-z]:[\\/]|\\\\)/;
function pathProblem(v, what, required) {
  v = (v || '').trim();
  if (!v) return required ? 'Enter ' + what + '.' : '';
  if (/^[A-Za-z0-9]{4}(-[A-Za-z0-9]{4}){3,}$/.test(v) || /^[A-Za-z0-9]{26}$/.test(v)) return 'That looks like a CD key. Never enter it here. You type it only in the Diablo II installer window.';
  if (!ABS.test(v)) return 'Use the full path, starting with / (Linux) or a drive letter such as C:\\.';
  if (/[\r\n\t]/.test(v)) return 'The path contains a line break or tab.';
  return '';
}
async function submitSources(ev) {
  ev.preventDefault();
  const f = form, e = {};
  const m = f.mode;
  e.dataDir = pathProblem(f.dataDir, 'a folder for D.idle to keep its files', true);
  if (m === 'installer-dirs') { e.installerD2Dir = pathProblem(f.installerD2Dir, 'the Diablo II installer folder', true); e.installerLoDDir = pathProblem(f.installerLoDDir, 'the Lord of Destruction installer folder', true); }
  if (m === 'stubs') { e.stubD2 = pathProblem(f.stubD2, 'the Diablo II downloader program', true); e.stubLoD = pathProblem(f.stubLoD, 'the Lord of Destruction downloader program', true); }
  if (m === 'game-dir') e.gameDirSrc = pathProblem(f.gameDirSrc, 'your Diablo II folder', true);
  e.patchFile = pathProblem(f.patchFile, 'the 1.14d patch file', m !== 'game-dir');
  const slots = parseInt(f.slots, 10);
  e.slots = slots >= 1 && slots <= 16 ? '' : 'Enter a number from 1 to 16.';
  for (const k of Object.keys(e)) if (!e[k]) delete e[k];
  formErr = e;
  if (Object.keys(e).length) { render(true); const first = document.querySelector('[aria-invalid="true"]'); if (first) first.focus(); return; }
  const body = { mode: m, dataDir: f.dataDir.trim(), slots };
  const pick = { 'installer-dirs': ['installerD2Dir', 'installerLoDDir'], stubs: ['stubD2', 'stubLoD'], 'game-dir': ['gameDirSrc'] }[m];
  for (const k of pick) body[k] = f[k].trim();
  if (f.patchFile.trim()) body.patchFile = f.patchFile.trim();
  const r = await api('PUT', '/inputs', body);
  if (!r.ok) { formErr = { _top: (r.data && r.data.error) || 'The settings were not accepted.' }; render(true); return; }
  formErr = {};
  if (step('sources') && step('sources').status === 'failed') await api('POST', '/reset?step=sources');
  await startRun();
}
async function submitCharacter(ev) {
  ev.preventDefault();
  const n = form.charName.trim();
  formErr = {};
  if (!/^[A-Za-z]{2,15}$/.test(n)) { formErr.charName = 'Use 2 to 15 letters, with no spaces, numbers or symbols.'; render(true); const x = $('f-charName'); if (x) x.focus(); return; }
  const pv = goalsValue(form);
  if (pv === null) { formErr.profile = form.pKind === 'farm' ? 'Use an area key of letters, digits and _ . - only.' : 'Enter a level from 2 to 99.'; render(true); const x = $('f-pValue'); if (x) x.focus(); return; }
  const body = { charName: n, charClass: form.charClass };
  if (pv) body.goals = pv;
  if (form.handicap) body.profile = form.handicap;
  const r = await api('PUT', '/inputs', body);
  if (!r.ok) { formErr._top = (r.data && r.data.error) || 'The settings were not accepted.'; render(true); return; }
  await startRun();
}

// ---- pieces --------------------------------------------------------------------------------------
function copyBtn(text, label) {
  const b = h('button', { class: 'btn small', type: 'button', 'aria-label': 'Copy ' + (label || 'command') }, 'Copy');
  b.addEventListener('click', async () => {
    let ok = false;
    try { await navigator.clipboard.writeText(text); ok = true; } catch (e) { /* fall back below */ }
    if (!ok) {
      const code = b.parentNode && b.parentNode.querySelector('code');
      if (code) { const r = document.createRange(); r.selectNodeContents(code); const sel = getSelection(); sel.removeAllRanges(); sel.addRange(r); }
    }
    b.textContent = ok ? 'Copied' : 'Press Ctrl+C';
    setTimeout(() => { b.textContent = 'Copy'; }, 1800);
  });
  return b;
}
function fixView(fix) {
  const parts = fix.split('`');
  const box = h('div', { class: 'wz-fix' });
  let para = [];
  const flush = () => { if (para.length) { box.append(h('p', null, para)); para = []; } };
  parts.forEach((p, i) => {
    if (i % 2 === 0) { if (p) para.push(p); }
    else if (/\s/.test(p.trim()) || /^[\w.\/-]+$/.test(p)) { flush(); box.append(h('div', { class: 'wz-cmd' }, h('code', null, p), copyBtn(p, 'command'))); }
    else para.push(h('code', { class: 'wz-inline' }, p));
  });
  flush();
  return box;
}
const BADGE = { pass: ['OK', 'pass'], warn: ['Look', 'warn'], fail: ['Fix needed', 'fail'], skip: ['Skipped', 'skip'] };
const GLYPH = { pass: '\u2713', warn: '!', fail: '\u2715', skip: '\u2013' };
function badge(st) { const b = BADGE[st] || [st, 'skip']; return h('span', { class: 'badge ' + b[1] }, h('span', { 'aria-hidden': 'true' }, GLYPH[st] || ''), b[0]); }
function checkItem(r) {
  return h('li', { class: 'wz-check ' + r.status },
    h('div', { class: 'wz-check-head' }, h('strong', null, r.title), badge(r.status)),
    r.detail ? h('div', { class: 'muted' }, r.detail) : null,
    r.fix && r.status !== 'pass' ? fixView(r.fix) : null);
}
function fieldPath(id, label, hint, ph) {
  const err = formErr[id];
  return h('div', { class: 'wz-field' },
    h('label', { for: 'f-' + id }, label),
    hint ? h('div', { class: 'hint', id: 'h-' + id }, hint) : null,
    h('input', { type: 'text', id: 'f-' + id, class: 'mono', value: form[id] || '', placeholder: ph || '', autocomplete: 'off', spellcheck: 'false', autocapitalize: 'off',
      'aria-describedby': (hint ? 'h-' + id + ' ' : '') + 'e-' + id, 'aria-invalid': err ? 'true' : null,
      oninput: (e) => { form[id] = e.target.value; dirty[id] = true; } }),
    h('div', { class: 'wz-err', id: 'e-' + id, role: 'alert' }, err || ''));
}
const NOKEY = 'This page never asks for a CD key. You type it only inside the Diablo II installer window, which is part of the game.';
function elapsed(iso) {
  const t = Date.parse(iso); if (!t) return '';
  const s = Math.max(0, Math.round((Date.now() - t) / 1000));
  return s < 60 ? s + ' s' : Math.floor(s / 60) + ' min ' + (s % 60) + ' s';
}

// ---- stepper -------------------------------------------------------------------------------------
function renderStepper() {
  const el = $('wz-stepper');
  const items = [{ id: '_check', title: 'Check your computer', st: (checks || anyProgress() || anyActive() || screen === 'sources') && screen !== 'welcome' && screen !== 'checks' ? 'done' : 'pending' }];
  for (const x of S ? S.steps : []) items.push({ id: x.id, title: FRIENDLY[x.id] || x.title, st: x.status === 'skipped' ? 'done' : x.status });
  let curIdx = 0;
  if (screen === 'welcome' || screen === 'checks') curIdx = 0;
  else if (screen === 'sources') curIdx = 1;
  else if (screen === 'done') curIdx = items.length;
  else if (screen === 'character') curIdx = items.findIndex((x) => x.id === 'character');
  else { const c = cur(); curIdx = c ? items.findIndex((x) => x.id === c.id) : 0; }
  if (curIdx < 0) curIdx = 0;
  const total = items.length;
  clear(el);
  const ol = h('ol');
  items.forEach((it, i) => {
    const done = it.st === 'done' || i < curIdx;
    ol.append(h('li', { class: (done ? 'done ' : '') + (it.st === 'failed' ? 'failed' : ''), 'aria-current': i === curIdx ? 'step' : null },
      h('span', { class: 'mark', 'aria-hidden': 'true' }, it.st === 'failed' ? '!' : done ? '\u2713' : String(i + 1)),
      h('span', null, it.title, h('span', { class: 'sr' }, done ? ' (done)' : i === curIdx ? ' (current)' : ' (later)'))));
  });
  const name = curIdx >= total ? 'All done' : items[curIdx].title;
  el.append(ol, h('div', { class: 'wz-compact' },
    h('div', { class: 'label' }, h('strong', null, name), h('span', null, curIdx >= total ? total + ' of ' + total : 'Step ' + (curIdx + 1) + ' of ' + total)),
    h('div', { class: 'wz-bar', role: 'progressbar', 'aria-label': 'Setup progress', 'aria-valuemin': '0', 'aria-valuemax': String(total), 'aria-valuenow': String(Math.min(curIdx, total)) }, h('i', { style: 'width:' + Math.round(100 * Math.min(curIdx, total) / total) + '%' }))));
}

// ---- screens -------------------------------------------------------------------------------------
function topErr() { return formErr._top ? h('div', { class: 'wz-note bad', role: 'alert' }, h('strong', null, 'Something went wrong'), formErr._top) : null; }

function screenWelcome() {
  return h('section', { class: 'wz-card', 'aria-labelledby': 'wz-h' },
    h('h2', { id: 'wz-h' }, 'Let\u2019s get D.idle ready'),
    h('p', { class: 'wz-lead' }, 'D.idle plays your own copy of Diablo II while you do something else. Setup takes about 20 minutes, and most of that is waiting. You make a few choices and the rest happens by itself.'),
    h('div', { class: 'wz-note' }, h('strong', null, 'What you need'),
      'Your own Diablo II and Lord of Destruction files (an installer folder, or an existing install), the 1.14d patch file, and your CD keys on hand for the game\u2019s installer.'),
    h('div', { class: 'wz-note' }, h('strong', null, 'About your CD key'), NOKEY + ' D.idle does not download or include any game files.'),
    h('p', null, 'First, a quick look at this computer to catch problems early.'),
    h('div', { class: 'wz-actions' },
      h('button', { class: 'btn primary big', type: 'button', onclick: () => go('checks') }, 'Check my computer'),
      h('button', { class: 'btn ghost', type: 'button', onclick: () => go('sources') }, 'Skip the check')));
}

function screenChecks() {
  const kids = [h('h2', { id: 'wz-h' }, 'Check my computer')];
  if (checksBusy) {
    kids.push(h('div', { class: 'wz-run', role: 'status' }, h('span', { class: 'spin', 'aria-hidden': 'true' }), h('span', null, checksFull ? 'Running the full check. This can take a minute.' : 'Looking at your computer\u2026')));
  } else if (checksErr) {
    kids.push(h('div', { class: 'wz-note bad', role: 'alert' }, h('strong', null, 'The check did not run'), checksErr));
    kids.push(h('div', { class: 'wz-actions' }, h('button', { class: 'btn primary big', type: 'button', onclick: () => runChecks(checksFull) }, 'Try again'), h('button', { class: 'btn ghost', type: 'button', onclick: () => go('sources') }, 'Continue anyway')));
  } else if (checks) {
    const c = checks.counts, rs = checks.results;
    const fails = rs.filter((r) => r.status === 'fail'), warns = rs.filter((r) => r.status === 'warn'), ok = rs.filter((r) => r.status === 'pass' || r.status === 'skip');
    const lead = fails.length ? fails.length + (fails.length === 1 ? ' thing needs' : ' things need') + ' fixing before setup can finish.'
      : warns.length ? 'Your computer will work. ' + warns.length + (warns.length === 1 ? ' item is' : ' items are') + ' worth a look.' : 'Everything looks good.';
    kids.push(h('p', { class: 'wz-lead', role: 'status' }, lead));
    kids.push(h('div', { class: 'wz-sum' }, ['fail', 'warn', 'pass', 'skip'].filter((k) => c[k]).map((k) => h('span', { class: 'badge ' + k }, h('span', { 'aria-hidden': 'true' }, GLYPH[k]), c[k] + ' ' + BADGE[k][0].toLowerCase()))));
    if (fails.length) kids.push(h('h3', null, 'Needs fixing'), h('ul', { class: 'wz-checks' }, fails.map(checkItem)));
    if (warns.length) kids.push(h('h3', null, 'Worth a look'), h('ul', { class: 'wz-checks' }, warns.map(checkItem)));
    if (ok.length) kids.push(h('details', { class: 'wz-passed' }, h('summary', null, 'Passed or not needed yet (' + ok.length + ')'), h('ul', { class: 'wz-checks', style: 'margin-top:10px' }, ok.map(checkItem))));
    if (fails.length) kids.push(h('p', { class: 'wz-hint' }, 'Fix these on this computer, then run the check again. Skipped checks mean the game is not installed yet. That is expected.'));
    kids.push(h('div', { class: 'wz-actions' },
      h('button', { class: 'btn primary big', type: 'button', onclick: () => go('sources') }, fails.length ? 'Continue anyway' : 'Continue'),
      h('button', { class: 'btn', type: 'button', onclick: () => runChecks(false) }, 'Check again'),
      h('button', { class: 'btn ghost', type: 'button', onclick: () => runChecks(true) }, 'Run the full check (slower)')));
  }
  return h('section', { class: 'wz-card', 'aria-labelledby': 'wz-h' }, kids);
}

function screenSources() {
  const m = form.mode, failed = step('sources') && step('sources').status === 'failed' ? step('sources') : null;
  const modeBox = h('fieldset', { class: 'wz-modes' }, h('legend', { class: 'wz-legend' }, 'Where is your Diablo II?'),
    MODES.map((x) => h('label', { class: 'wz-mode' },
      h('input', { type: 'radio', name: 'mode', value: x.id, checked: m === x.id ? true : null, onchange: () => { form.mode = x.id; formErr = {}; render(true); const r = document.querySelector('input[name=mode]:checked'); if (r) r.focus(); } }),
      h('div', null, h('strong', null, x.title), h('span', null, x.sub)))));
  const paths = [];
  if (m === 'installer-dirs') paths.push(
    fieldPath('installerD2Dir', 'Diablo II installer folder', 'The folder with the base game\u2019s setup files.', '/home/you/Downloads/Diablo2'),
    fieldPath('installerLoDDir', 'Lord of Destruction installer folder', 'The folder with the expansion\u2019s setup files.', '/home/you/Downloads/Diablo2-LoD'));
  if (m === 'stubs') paths.push(
    fieldPath('stubD2', 'Diablo II downloader program', 'The small setup program you got from Blizzard for the base game.', '/home/you/Downloads/Diablo-II-Setup.exe'),
    fieldPath('stubLoD', 'Lord of Destruction downloader program', 'The one for the expansion.', '/home/you/Downloads/Diablo-II-LoD-Setup.exe'));
  if (m === 'game-dir') paths.push(fieldPath('gameDirSrc', 'Your Diablo II folder', 'The folder that contains Game.exe. D.idle copies it and leaves yours alone.', '/home/you/Games/Diablo II'));
  paths.push(fieldPath('patchFile', m === 'game-dir' ? 'The 1.14d patch file (optional)' : 'The 1.14d patch file',
    m === 'game-dir' ? 'Only needed if your install is not already version 1.14d. Usually named LODPatch_114d.exe.' : 'Usually named LODPatch_114d.exe. You supply it. D.idle never downloads it.', '/home/you/Downloads/LODPatch_114d.exe'));
  const slotErr = formErr.slots;
  return h('section', { class: 'wz-card', 'aria-labelledby': 'wz-h' },
    h('h2', { id: 'wz-h' }, 'Point D.idle at your game files'),
    h('p', null, 'Type the full path of each folder, or paste it from your file manager. D.idle checks everything before it changes anything.'),
    topErr(),
    failed ? h('div', { class: 'wz-note bad', role: 'alert' }, h('strong', null, 'D.idle could not use those files'), failed.error || 'Check the paths below and try again.') : null,
    h('form', { id: 'wz-form', novalidate: true, onsubmit: submitSources, style: 'display:grid;gap:18px' },
      fieldPath('dataDir', 'Where should D.idle keep its files?', 'A folder on a local disk with at least 10 GB free. It is created if it does not exist.', '/home/you/d2idle'),
      modeBox,
      h('div', { class: 'wz-paths' }, paths),
      h('div', { class: 'wz-note' }, h('strong', null, 'No CD key here'), NOKEY),
      h('details', { class: 'wz-more' }, h('summary', null, 'Advanced'),
        h('div', { class: 'wz-field', style: 'margin-top:10px;max-width:14rem' },
          h('label', { for: 'f-slots' }, 'Characters playing at once'),
          h('div', { class: 'hint', id: 'h-slots' }, 'Each one uses about one CPU core. Start with 1.'),
          h('input', { type: 'number', id: 'f-slots', min: '1', max: '16', value: String(form.slots || 1), 'aria-describedby': 'h-slots e-slots', 'aria-invalid': slotErr ? 'true' : null, oninput: (e) => { form.slots = e.target.value; } }),
          h('div', { class: 'wz-err', id: 'e-slots', role: 'alert' }, slotErr || ''))),
      h('div', { class: 'wz-actions' },
        h('button', { class: 'btn primary big', type: 'submit' }, 'Check these and start'),
        h('button', { class: 'btn ghost', type: 'button', onclick: () => go('checks') }, 'Back to the computer check'))));
}

function instructions(id) {
  const first = id === 'install-d2' ? 'Diablo II' : id === 'install-lod' ? 'Lord of Destruction' : '';
  if (id === 'patch') return ['Open the patcher window (see below).', 'Click the patch button and wait until it says it is finished.', 'Close the patcher if it stays open. D.idle notices on its own and moves on.'];
  return ['Open the installer window (see below).', 'Click through the ' + first + ' installer as you normally would.', 'When the installer asks for a CD key, type it there. It is the game\u2019s own screen, and D.idle never sees it.', 'When the installer finishes, come back here. This page continues by itself.'];
}
function displayHint() {
  const d = S && S.display;
  if (!d) return null;
  if (/^https?:\/\//.test(d)) return h('p', null, h('a', { class: 'btn primary big', href: d, target: '_blank', rel: 'noopener' }, 'Open the installer window'));
  return h('p', { class: 'wz-hint' }, 'The window opens on the virtual screen ', h('code', { class: 'wz-inline' }, d), '. Connect to it with your VNC or remote desktop viewer, or use the Viewport panel on the dashboard.');
}
function screenProgress() {
  const c = cur() || (S && S.steps[S.steps.length - 1]);
  if (!c) return h('section', { class: 'wz-card' }, h('p', null, 'Loading\u2026'));
  const kids = [];
  const title = FRIENDLY[c.id] || c.title;
  const st = c.status;
  kids.push(h('h2', { id: 'wz-h' }, st === 'failed' ? 'This step hit a problem' : title));
  if (st === 'failed') kids.push(h('p', { class: 'muted' }, 'Step: ' + title));
  if (st === 'running' || (S.running && st === 'pending')) {
    kids.push(h('div', { class: 'wz-run', role: 'status' }, h('span', { class: 'spin', 'aria-hidden': 'true' }), h('span', null, (WORKING[c.id] || 'Working') + '\u2026'), c.started ? h('span', { class: 'muted' }, ' ' + elapsed(c.started)) : null));
    kids.push(h('div', { class: 'wz-actions' }, h('button', { class: 'btn', type: 'button', onclick: cancelRun }, 'Cancel')));
  } else if (st === 'waiting-user') {
    kids.push(h('div', { class: 'wz-turn' }, h('div', null, h('span', { class: 'badge wait' }, 'Your turn')),
      c.note ? h('p', { class: 'wz-lead' }, c.note) : null,
      h('ol', null, instructions(c.id).map((t) => h('li', null, t))), displayHint()));
    kids.push(h('div', { class: 'wz-note' }, h('strong', null, 'No CD key on this page'), NOKEY));
    kids.push(h('div', { class: 'wz-actions' }, h('button', { class: 'btn', type: 'button', onclick: cancelRun }, 'Cancel setup')));
  } else if (st === 'failed') {
    kids.push(h('div', { class: 'wz-note bad', role: 'alert' }, h('strong', null, 'What went wrong'), c.error || 'The step stopped without a message. The log below may say more.'));
    kids.push(h('p', null, 'Nothing you did earlier is lost. You can try this step again' + (c.id === 'sources' ? ', or change your answers first.' : '. If it keeps failing, check your computer or read the log.')));
    kids.push(h('div', { class: 'wz-actions' },
      h('button', { class: 'btn primary big', type: 'button', onclick: () => retryStep(c.id) }, 'Try again'),
      c.id === 'sources' || c.id === 'data-dir' ? h('button', { class: 'btn', type: 'button', onclick: () => go('sources') }, 'Change my answers') : null,
      h('button', { class: 'btn ghost', type: 'button', onclick: () => { checks = null; go('checks'); } }, 'Check my computer')));
  } else {
    kids.push(h('p', { class: 'wz-lead' }, c.note || 'Setup is paused. Continue when you are ready.'));
    kids.push(h('div', { class: 'wz-actions' }, h('button', { class: 'btn primary big', type: 'button', onclick: startRun }, 'Continue setup')));
  }
  // finished steps summary
  const doneN = S.steps.filter((x) => x.status === 'done' || x.status === 'skipped').length;
  if (doneN) kids.push(h('p', { class: 'wz-hint' }, doneN + ' of ' + S.steps.length + ' steps finished.' + (st === 'failed' ? '' : ' You can close this page and come back. Setup keeps going and this page picks up where it was.')));
  kids.push(logBox());
  kids.push(resetBox());
  return h('section', { class: 'wz-card', 'aria-labelledby': 'wz-h' }, kids);
}
function logText() { const l = S.log || []; return (showFullLog ? l : l.slice(-8)).join('\n'); }
function logBox() {
  const n = (S.log || []).length;
  return h('div', { style: 'display:grid;gap:6px' },
    h('div', { class: 'card-head', style: 'margin:0' }, h('h3', { style: 'margin:0' }, 'What is happening'),
      n > 8 ? h('button', { class: 'btn small ghost', type: 'button', onclick: () => { showFullLog = !showFullLog; render(true); } }, showFullLog ? 'Show less' : 'Show all ' + n + ' lines') : null),
    h('pre', { class: 'wz-log', id: 'wz-log', tabindex: '0', role: 'log', 'aria-label': 'Setup log' }, logText()));
}
function resetBox() {
  if (S.running) return null;
  return confirmReset
    ? h('div', { class: 'wz-note warn' }, h('strong', null, 'Start over from the beginning?'), 'D.idle forgets which steps are done. Your game files are not deleted.',
        h('div', { class: 'wz-actions' }, h('button', { class: 'btn danger', type: 'button', onclick: startOver }, 'Yes, start over'), h('button', { class: 'btn', type: 'button', onclick: () => { confirmReset = false; render(true); } }, 'Keep my progress')))
    : h('div', null, h('button', { class: 'btn ghost small', type: 'button', onclick: () => { confirmReset = true; render(true); } }, 'Start over'));
}

function screenCharacter() {
  const failed = cur() && cur().status === 'failed';
  const e = formErr.charName;
  return h('section', { class: 'wz-card', 'aria-labelledby': 'wz-h' },
    h('h2', { id: 'wz-h' }, 'Create your first test character'),
    h('p', { class: 'wz-lead' }, 'Everything is installed. The last step makes a throwaway character so D.idle can prove that it can play your game.'),
    h('p', null, 'It is a normal softcore Expansion character on Normal difficulty. The game opens in a hidden window for about a minute and closes again. You can delete this character later.'),
    topErr(),
    h('form', { id: 'wz-char', novalidate: true, onsubmit: submitCharacter, style: 'display:grid;gap:16px;max-width:26rem' },
      h('div', { class: 'wz-field' }, h('label', { for: 'f-charName' }, 'Character name'),
        h('div', { class: 'hint', id: 'h-charName' }, '2 to 15 letters. No spaces, numbers or symbols.'),
        h('input', { type: 'text', id: 'f-charName', value: form.charName, placeholder: 'Setupbot', maxlength: '15', autocomplete: 'off', spellcheck: 'false', 'aria-describedby': 'h-charName e-charName', 'aria-invalid': e ? 'true' : null, oninput: (ev) => { form.charName = ev.target.value; dirty.charName = true; } }),
        h('div', { class: 'wz-err', id: 'e-charName', role: 'alert' }, e || '')),
      h('div', { class: 'wz-field' }, h('label', { for: 'f-charClass' }, 'Class'),
        h('select', { id: 'f-charClass', onchange: (ev) => { form.charClass = ev.target.value; dirty.charClass = true; } },
          CLASSES.map((c) => h('option', { value: c, selected: form.charClass === c ? true : null }, c[0].toUpperCase() + c.slice(1))))),
      h('div', { class: 'wz-field' }, h('label', { for: 'f-profile' }, 'What should it work on?'),
        h('select', { id: 'f-profile', onchange: (ev) => { form.pKind = ev.target.value; render(true); const x = $('f-profile'); if (x) x.focus(); } },
          GOAL_PRESETS.map(([v, t]) => h('option', { value: v, selected: form.pKind === v ? true : null }, t)))),
      h('div', { class: 'wz-field' }, h('label', { for: 'f-handicap' }, 'How well should it play?'),
        h('select', { id: 'f-handicap', onchange: (ev) => { form.handicap = ev.target.value; } },
          HANDICAPS.map(([v, t]) => h('option', { value: v, selected: form.handicap === v ? true : null }, t)))),
      form.pKind === 'levelto' || form.pKind === 'farm' ? h('div', { class: 'wz-field' },
        h('label', { for: 'f-pValue' }, form.pKind === 'levelto' ? 'Target level' : 'Area key'),
        h('input', form.pKind === 'levelto'
          ? { type: 'number', id: 'f-pValue', min: '2', max: '99', value: String(form.pLevel), 'aria-describedby': 'e-profile', 'aria-invalid': formErr.profile ? 'true' : null, oninput: (ev) => { form.pLevel = ev.target.value; } }
          : { type: 'text', id: 'f-pValue', value: form.pFarm, placeholder: 'e.g. mephisto', maxlength: '40', autocomplete: 'off', spellcheck: 'false', 'aria-describedby': 'e-profile', 'aria-invalid': formErr.profile ? 'true' : null, oninput: (ev) => { form.pFarm = ev.target.value; } }),
        h('div', { class: 'wz-err', id: 'e-profile', role: 'alert' }, formErr.profile || '')) : null,
      h('div', { class: 'wz-actions' }, h('button', { class: 'btn primary big', type: 'submit' }, failed ? 'Try again' : 'Create my test character'))),
    logBox());
}

function screenDone() {
  const i = (S && S.inputs) || {};
  return h('section', { class: 'wz-card', 'aria-labelledby': 'wz-h' },
    h('div', { class: 'wz-done-mark', 'aria-hidden': 'true' }, '\u2713'),
    h('h2', { id: 'wz-h' }, 'You are all set'),
    h('p', { class: 'wz-lead' }, 'Diablo II is installed and updated to 1.14d, the game data is read, and your test character ' + (i.charName ? i.charName + ' ' : '') + 'is ready.'),
    h('p', null, 'Open the dashboard to give D.idle its first goal.'),
    h('div', { class: 'wz-actions' }, h('a', { class: 'btn primary big', href: 'index.html' }, 'Open the dashboard'),
      h('button', { class: 'btn ghost', type: 'button', onclick: () => { checks = null; go('checks'); } }, 'Check my computer again')),
    logBox());
}

// ---- render --------------------------------------------------------------------------------------
function render(force) {
  const main = $('wz-main');
  if (!S) {
    if (!main.firstChild) main.append(h('section', { class: 'wz-card' }, h('div', { class: 'wz-run', role: 'status' }, h('span', { class: 'spin', 'aria-hidden': 'true' }), h('span', null, 'Loading setup\u2026'))));
    if (offline) { clear(main); main.append(h('section', { class: 'wz-card' }, h('div', { class: 'wz-note bad', role: 'alert' }, h('strong', null, 'Cannot reach D.idle'), 'The setup server is not answering. Make sure D.idle is running. This page keeps trying.'))); }
    return;
  }
  const next = pickScreen();
  const c = cur();
  const key = [next, c && c.id, c && c.status, c && c.error, S.running, S.done, offline, showFullLog, confirmReset, checksBusy, checks && checks.status].join('|');
  const changedScreen = next !== screen;
  screen = next;
  $('wz-dash').hidden = !S.done;
  renderStepper();
  if (!force && key === renderKey && !changedScreen) {
    const lg = $('wz-log');
    if (lg && lg.textContent !== logText()) { const atEnd = lg.scrollTop + lg.clientHeight >= lg.scrollHeight - 8; lg.textContent = logText(); if (atEnd) lg.scrollTop = lg.scrollHeight; }
    return;
  }
  renderKey = key;
  const active = document.activeElement && document.activeElement.id;
  const view = { welcome: screenWelcome, checks: screenChecks, sources: screenSources, progress: screenProgress, character: screenCharacter, done: screenDone }[screen]();
  clear(main);
  if (offline) main.append(h('div', { class: 'wz-note warn', role: 'alert' }, h('strong', null, 'Lost contact with D.idle'), 'Trying again every few seconds. What you see may be out of date.'));
  main.append(view);
  const lg = $('wz-log'); if (lg) lg.scrollTop = lg.scrollHeight;
  if (changedScreen) { if (!['sources', 'character'].includes(screen) || !formErr._top) main.focus({ preventScroll: false }); window.scrollTo(0, 0); }
  else if (active && $(active)) $(active).focus();
  const ann = screen + '|' + (c ? c.id + c.status : '');
  if (ann !== announced) { announced = ann; $('wz-live').textContent = screen === 'progress' && c ? (FRIENDLY[c.id] || c.title) + ': ' + c.status : screen === 'done' ? 'Setup complete' : ''; }
}

// ---- theme ---------------------------------------------------------------------------------------
function applyTheme(t) {
  if (t === 'light' || t === 'dark') document.documentElement.dataset.theme = t; else delete document.documentElement.dataset.theme;
  $('btn-theme').textContent = 'Theme: ' + (t === 'light' || t === 'dark' ? t : 'auto');
}
let theme = 'auto';
try { theme = localStorage.getItem('d2theme') || 'auto'; } catch (e) { /* storage blocked */ }
$('btn-theme').onclick = () => {
  theme = { auto: 'light', light: 'dark', dark: 'auto' }[theme] || 'auto';
  try { localStorage.setItem('d2theme', theme); } catch (e) { /* ignore */ }
  applyTheme(theme);
};
applyTheme(theme);
{ const n = location.hash.slice(1); if (['welcome', 'checks', 'sources'].includes(n)) manual = n; }
if (manual === 'checks') runChecks(false);
render(true);
refresh();
})();
