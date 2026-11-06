'use strict';
// d2d first-run setup page: six steps (docs/research/onboarding-ux-design.md). All text is inserted with textContent.
// There is deliberately no CD key field. The server derives how far setup got (st.phase); this page only decides
// which step to SHOW (Back goes to an earlier one, never past what the server allows).
const $ = (id) => document.getElementById(id);
function h(tag, attrs, ...kids) {
  const el = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs || {})) {
    if (k === 'text') el.textContent = v;
    else if (k.startsWith('on')) el.addEventListener(k.slice(2), v);
    else el.setAttribute(k, v);
  }
  for (const k of kids.flat()) if (k != null) el.append(typeof k === 'string' ? document.createTextNode(k) : k);
  return el;
}
const STEPS = ['Welcome', 'Go-ahead', 'Game files', 'Hero', 'Access', 'Start'];
const FURTHEST = { ahead: 1, data: 2, character: 3, access: 4, start: 5, running: 5, failed: 5, done: 5 };
let st = null, err = '', fix = '', at = -1, timer = 0, watching = false, form = { pass: '', pass2: '', kind: 'passcode' };

async function api(method, path, body) {
  let r;
  try {
    r = await fetch('/api' + path, { method, cache: 'no-store', credentials: 'same-origin',
      headers: body ? { 'Content-Type': 'application/json' } : undefined, body: body ? JSON.stringify(body) : undefined });
  } catch (_) {
    throw Object.assign(new Error('The setup page cannot be reached.'), { fix: 'Check that the container is still running, then press Retry.' });
  }
  const j = await r.json().catch(() => ({}));
  if (!r.ok) throw Object.assign(new Error(j.error || 'The request failed (' + r.status + ').'), { fix: j.fix || '' });
  return j;
}
function poll() {
  clearTimeout(timer);
  if (st && (st.phase === 'running' || st.phase === 'done')) timer = setTimeout(load, 1500);
}
async function load() {
  try { st = await api('GET', '/state'); err = fix = ''; } catch (e) { if (!(st && st.phase === 'done')) { err = e.message; fix = e.fix; } }
  render(); poll();
}
// act runs a request, stores the new state, and moves to step `to` (when given and allowed) once it worked.
async function act(fn, to) {
  try { err = fix = ''; st = await fn(); if (to != null) at = Math.min(to, FURTHEST[st.phase]); } catch (e) { err = e.message; fix = e.fix; }
  render(); poll();
}
function go(to) { at = to; err = fix = ''; render(); }

function stepper() {
  $('ob-stepper').replaceChildren(h('ol', { class: 'ob-steps' }, STEPS.map((s, i) =>
    h('li', { class: i < at ? 'done' : i === at ? 'cur' : '', ...(i === at ? { 'aria-current': 'step' } : {}) },
      h('i', { 'aria-hidden': 'true' }), (i + 1) + '', h('span', { text: '. ' + s })))));
}
function why(t) { return h('div', { class: 'ob-why' }, h('b', { text: 'Why we need this. ' }), t); }
function nav(back, ...right) {
  return h('div', { class: 'ob-nav' }, back ? h('button', { class: 'btn', type: 'button', text: 'Back', onclick: () => go(at - 1) }) : h('span'), h('div', { class: 'r' }, right));
}
function btn(text, onclick, cls, disabled) {
  const b = h('button', { class: 'btn ' + (cls || ''), type: 'button', text, onclick });
  if (disabled) b.disabled = true;
  return b;
}
function box(kind, title, body, fixText) {
  return h('div', { class: 'ob-box ' + kind, role: kind === 'err' ? 'alert' : 'status' }, h('strong', { text: title }), body ? h('span', { text: body }) : null, fixText ? h('p', { text: fixText }) : null);
}
function errBox() { return err ? box('err', err, '', fix) : null; }

// ---- 1 welcome
function sWelcome() {
  return [h('h2', { text: 'Welcome to d.idle' }),
    h('p', { text: 'd.idle is an idle game made from Diablo II. Five things to know:' }),
    h('ul', { class: 'ob-list' }, [
      'Your hero, a Sorceress, plays Diablo II on its own. You do not steer it.',
      'When it is unsure, it asks you a question. Your answers teach it, and after a few matching answers it stops asking and does that itself.',
      'You can also send it somewhere, and read a journal of what it did and why.',
      'It plays offline, single player only. It never connects to Battle.net and never needs a CD key or an account.',
      'You bring your own Diablo II game files. This takes about 5 minutes the first time.'].map((t) => h('li', { text: t }))),
    why('So you know what you are about to start before anything is checked or stored.'),
    nav(false, btn('Get started', () => { sessionStorage.setItem('ob-welcomed', '1'); go(1); }, 'primary'))];
}
// ---- 2 go-ahead
function sAhead() {
  const cb = h('input', { type: 'checkbox', id: 'ob-go', onchange: (ev) => act(() => api('PUT', '/ahead', { ok: ev.target.checked })) });
  cb.checked = st.ahead.ok;
  const item = (a, b) => h('li', {}, h('b', { text: a + ' ' }), b);
  return [h('h2', { text: 'Before we start: what runs on this computer' }),
    errBox(),
    h('ul', { class: 'ob-list' }, [
      item('What runs.', 'One container on this computer: a copy of the Diablo II engine (the game engine) and the player program (d.idle). Nothing runs anywhere else.'),
      item('Your game files stay here.', 'They are read in place, read-only. They are never uploaded and never put in the image or the project.'),
      item('Offline only.', 'The hero plays single player. No Battle.net, no online play, no other players.'),
      item('What we store.', 'In the data volume only: your hero\'s save, what you answered, the rules it learned, logs, and your passcode (scrambled). Nothing is sent back to us; there is no telemetry.'),
      item('How to stop.', 'The Stop button on the watch screen, or "docker compose stop". The hero saves first. Reset deletes the hero and what it learned, and nothing else.'),
      item('Early test build.', 'The hero can die, get stuck, or ask odd questions. That is what you are here to find.')]),
    h('label', { class: 'ob-opt', for: 'ob-go' }, cb, h('strong', { text: 'I understand and I want to continue.' })),
    st.ahead.ok && st.ahead.at ? h('p', { class: 'ob-hint', text: 'Ticked on ' + st.ahead.at.slice(0, 10) + '. It is saved here, not sent anywhere.' }) : null,
    why('We would rather you decide with the facts than find out later what it does.'),
    nav(true, btn('Continue', () => go(2), 'primary', !st.ahead.ok))];
}
// ---- 3 game files
const VERDICT = {
  ok: ['ok', 'All files found. Version 1.14d confirmed.'],
  unconfirmed: ['warn', 'Version not confirmed yet.'],
  wrong_version: ['err', 'These files are not from version 1.14d.'],
  resurrected: ['err', 'This looks like Diablo II: Resurrected, or the wrong folder.'],
  missing: ['err', 'Some files are missing.'],
  too_small: ['err', 'Some files look incomplete.'],
  not_mpq: ['err', 'Some files are not Diablo II archives.'],
  unreadable: ['err', 'Some files cannot be read.'],
  problems: ['err', 'Some files need attention.'],
  no_folder: ['err', 'The game folder cannot be read.'],
};
function fileRow(f) {
  let label = ['OK', 'good'];
  if (f.status === 'fail') label = [{ missing: 'Missing', too_small: 'Too small', not_mpq: 'Not an archive', unreadable: 'Cannot read' }[f.issue] || 'Problem', 'bad'];
  else if (f.status === 'warn') label = ['Problem (optional)', 'warn'];
  else if (f.status === 'info') label = ['Not needed', ''];
  return h('li', {}, h('span', { class: 'n', text: f.name }), h('span', { class: 's ' + label[1], text: label[0] }), h('span', { class: 'm', text: f.message }));
}
function sFiles() {
  const c = st.check, v = VERDICT[c.verdict] || VERDICT.problems;
  const ver = c.verdict === 'unconfirmed';
  let detail = c.summary;
  if (c.verdict === 'wrong_version') detail = 'This build was made for 1.14d, the last classic patch. Other versions play differently and the hero would misjudge things.';
  if (c.verdict === 'resurrected') detail = 'This build needs the classic game, Diablo II: Lord of Destruction, version 1.14d. Its files are named like d2data.mpq. Resurrected keeps its data in a format we cannot use.';
  if (ver) detail = 'All files are there, but this early build has no verified fingerprint for 1.14d, so we cannot prove the version. You may continue; if the hero behaves strangely, say so in your report.' + (c.patch_sha256 ? ' Fingerprint (for the report): ' + c.patch_sha256 : '');
  const shown = c.verdict === 'resurrected' ? [] : c.files.filter((f) => f.status !== 'info');
  const optional = c.files.filter((f) => f.status === 'info').length;
  return [h('h2', { text: 'Your Diablo II game files' }),
    errBox(),
    h('p', { text: 'This is the folder you gave the container (it sees it as ' + c.dir + ', read-only).' }),
    shown.length ? h('ul', { class: 'ob-files' }, shown.map(fileRow)) : null,
    optional ? h('p', { class: 'ob-hint', text: 'Sound, music and speech files are not needed: the hero plays silently.' }) : null,
    box(v[0], v[1], ' ' + detail, c.fix),
    h('p', { class: 'ob-hint', text: 'Your files are only read, here, on this computer. They are not uploaded and not copied anywhere.' }),
    why('The game engine cannot run without the original data, and we cannot give it to you. The version check stops you losing an evening to a hero that plays the wrong game.'),
    nav(true, btn('Check again', () => act(() => api('POST', '/check'), null)), btn('Next', () => go(3), 'primary', !c.ok))];
}
// ---- 4 hero
function sHero() {
  const nameIn = h('input', { type: 'text', id: 'ob-name', maxlength: '15', autocomplete: 'off', spellcheck: 'false', value: st.inputs.char_name || '' });
  const cls = h('select', { id: 'ob-class' }, st.classes.map((c) => {
    const o = h('option', { value: c.id, text: c.name + (c.supported ? '' : ' (not supported in this build)') });
    if (!c.supported) o.disabled = true;
    if (c.id === st.inputs.class) o.selected = true;
    return o;
  }));
  const goals = h('select', { id: 'ob-goals' }, st.presets.map((p) => {
    const o = h('option', { value: p.id, text: p.name });
    if (p.id === st.inputs.goals) o.selected = true;
    return o;
  }));
  const e = h('div', { class: 'ob-err', id: 'ob-e-name', role: 'alert' });
  const save = () => {
    const n = nameIn.value.trim();
    if (!/^[A-Za-z]{2,15}$/.test(n)) { e.textContent = 'Use 2 to 15 letters, no digits or spaces.'; nameIn.setAttribute('aria-invalid', 'true'); nameIn.focus(); return; }
    act(() => api('PUT', '/inputs', { char_name: n, class: cls.value, goals: goals.value }), 4);
  };
  const field = (id, label, control, hint) => h('div', { class: 'ob-field' }, h('label', { for: id, text: label }), control, hint ? h('div', { class: 'ob-hint', text: hint }) : null);
  return [h('h2', { text: 'Your hero' }), errBox(),
    field('ob-name', 'Name (letters only, 2 to 15)', nameIn, 'A character with this name is created, or loaded if it already exists.'), e,
    field('ob-class', 'Class', cls, 'This build plays the Sorceress. The other classes are listed but not supported yet.'),
    field('ob-goals', 'Goal', goals, 'You can change the goal later.'),
    h('p', { class: 'ob-hint', text: 'No CD key is needed or accepted here. The game runs offline.' }),
    why('The hero needs a name for its save file and a goal so it knows what to work on.'),
    nav(true, btn('Next', save, 'primary'))];
}
// ---- 5 access
function sAccess() {
  const kind = st.access && !form.touched ? st.access : form.kind;
  const radio = (v, title, text) => {
    const r = h('input', { type: 'radio', name: 'ob-kind', value: v, onchange: () => { form.kind = v; form.touched = true; render(); } });
    r.checked = kind === v;
    return h('label', { class: 'ob-opt' }, r, h('div', {}, h('strong', { text: title }), h('span', { text: text })));
  };
  const p1 = h('input', { type: 'password', id: 'ob-p1', autocomplete: 'new-password', value: form.pass, oninput: (ev) => { form.pass = ev.target.value; hint(); } });
  const p2 = h('input', { type: 'password', id: 'ob-p2', autocomplete: 'new-password', value: form.pass2, oninput: (ev) => { form.pass2 = ev.target.value; hint(); } });
  const msg = h('div', { class: 'ob-err', id: 'ob-e-pass', role: 'alert' });
  const next = btn('Next', () => {
    if (st.access === kind && !form.pass && !form.pass2) { go(5); return; } // already chosen, nothing new typed
    if (kind === 'token') { act(() => api('PUT', '/access', { kind: 'token' }), 5); return; }
    const pass = form.pass; form.pass = form.pass2 = '';
    act(() => api('PUT', '/access', { kind: 'passcode', passcode: pass, confirm: p2.value }), 5);
  }, 'primary');
  function hint() {
    const short = form.pass.length > 0 && [...form.pass].length < 6;
    const diff = form.pass2.length > 0 && form.pass !== form.pass2;
    msg.textContent = short ? 'Too short. Use 6 or more characters. Four random words work well.' : diff ? 'The two passcodes are not the same yet.' : '';
    p1.setAttribute('aria-invalid', short ? 'true' : 'false');
    const keep = st.access === kind && !form.pass && !form.pass2;
    next.disabled = kind === 'passcode' && !keep && ([...form.pass].length < 6 || form.pass !== form.pass2);
  }
  const fields = kind === 'passcode' ? [
    h('div', { class: 'ob-field' }, h('label', { for: 'ob-p1', text: 'Passcode' }), p1),
    h('div', { class: 'ob-field' }, h('label', { for: 'ob-p2', text: 'Type it again' }), p2), msg] : [];
  const out = [h('h2', { text: 'Who may open the watch screen?' }), errBox(),
    h('p', { text: 'The watch screen shows your game and lets whoever opens it answer the hero\'s questions.' }),
    st.access === 'passcode' && kind === 'passcode' ? h('p', { class: 'ob-hint', text: 'A passcode is already set. Leave the boxes empty to keep it, or type a new one.' }) : null,
    radio('passcode', 'Choose a passcode (recommended)', 'Easy to type on a phone. Works from any device on your home network. 6 or more characters; four random words are strong and easy to remember. You can reset it later from the container.'),
    radio('token', 'Use the generated token', '32 random characters, no choosing. Awkward to type on a phone. Shown once; you can read it again from the container.'),
    ...fields,
    h('div', { class: 'ob-why' }, h('b', { text: 'Other devices. ' }), 'To watch from your phone, the container must be published on your home network and started with your address, for example http://my-pc:8891. The guide shows the two lines to change. Never put this on the open internet: the page is plain HTTP.'),
    why('Anyone who can open the page can answer for your hero, and the answers change how it plays. A passcode or token keeps other people on your network out.'),
    nav(true, next)];
  queueMicrotask(hint);
  return out;
}
// ---- 6 start
const KIND_TEXT = { passcode: 'your passcode', token: 'the generated token' };
function sStart() {
  switch (st.phase) {
    case 'running': return startRunning();
    case 'done': return startDone();
    default: return startReady();
  }
}
function startReady() {
  const cn = (st.classes.find((c) => c.id === st.inputs.class) || {}).name || st.inputs.class;
  const pn = (st.presets.find((p) => p.id === st.inputs.goals) || {}).name || st.inputs.goals;
  const failed = st.phase === 'failed';
  return [h('h2', { text: failed ? 'The game tables could not be read' : 'Ready to start' }),
    errBox(),
    failed ? box('err', st.error, '', st.fix) : null,
    st.interrupted ? box('warn', 'Setup was interrupted.', ' It stopped before it finished (the container restarted). Press Start to continue; nothing was lost.') : null,
    h('ul', { class: 'ob-list' }, [
      h('li', {}, 'Hero: ', h('b', { text: st.inputs.char_name + ', ' + cn })),
      h('li', { text: 'Goal: ' + pn }),
      h('li', { text: 'Game files: ' + (st.check.version === '1.14d' ? '1.14d confirmed' : 'found, version not confirmed yet') }),
      h('li', { text: 'Access: ' + (KIND_TEXT[st.access] || '') })]),
    box('warn', 'The first start does a one-time job.', ' It reads the game\'s data tables. About 1 to 3 minutes (an estimate, not measured yet). Keep this page open, or close it and come back: it picks up where it was.'),
    why('This is the one slow step. Doing it now means the hero starts at once next time.'),
    nav(true, btn(failed ? 'Retry' : 'Start d.idle', () => act(() => api('POST', '/start')), 'primary'))];
}
function startRunning() {
  const p = st.progress || { percent: 0, steps: [], elapsed_sec: 0, estimate_sec: 120 };
  const fillBar = h('i');
  const bar = h('div', { class: 'ob-bar', role: 'progressbar', 'aria-label': 'Setup progress', 'aria-valuemin': '0', 'aria-valuemax': '100', 'aria-valuenow': String(p.percent) }, fillBar);
  fillBar.style.width = p.percent + '%';
  const left = Math.max(0, p.estimate_sec - p.elapsed_sec);
  return [h('h2', { text: 'Starting up' }),
    h('p', { text: 'Reading the game data tables. This is the slow part and happens once.' }),
    bar,
    h('ul', { class: 'ob-run' }, (p.steps || []).map((s, i) => h('li', { 'data-state': s.state }, h('span', { class: s.state === 'doing' ? 'spin' : '', 'aria-hidden': 'true' }), 'Step ' + (i + 1) + ' of 3: ' + s.name + (s.state === 'doing' ? ' (now)' : s.state === 'done' ? ' (done)' : '')))),
    h('p', { class: 'ob-hint', text: p.elapsed_sec + ' s so far. ' + (left > 0 ? 'About ' + left + ' s left (an estimate).' : 'Taking longer than the estimate; that is fine on a slow disk.') + ' You can close this page; setup continues.' })];
}
function startDone() {
  if (st.token) sessionStorage.setItem('ob-tok', st.token);
  const tok = sessionStorage.getItem('ob-tok');
  if (!watching) { watching = true; waitForWatch(); }
  const name = st.inputs.char_name;
  return [h('h2', { text: name + ' is waking up' }),
    box('ok', 'Setup finished.', st.access === 'token' ? ' Keep your token; this browser also remembers it.' : ' Use your passcode to open the watch screen on other devices.'),
    h('div', { class: 'ob-run', role: 'status' }, h('span', { class: 'spin', 'aria-hidden': 'true' }), h('span', { text: 'The game engine is starting. This page moves on by itself.' })),
    tok && st.access === 'token' ? h('div', { class: 'ob-box' }, h('strong', { text: 'Your watch token (shown once)' }), h('code', { class: 'ob-token', text: tok }),
      h('p', { class: 'ob-hint', text: 'Open the watch screen from another device with /watch?t=TOKEN. It is also in the data volume as watch.token.' })) : null,
    h('p', {}, h('a', { href: '/watch', text: 'Open the watch screen now' }))];
}
function waitForWatch() {
  const tick = async () => {
    try { const r = await fetch('/watch', { cache: 'no-store', credentials: 'same-origin' }); if (r.ok) { location.assign('/watch'); return; } } catch (_) { /* the setup is handing over the port */ }
    setTimeout(tick, 2000);
  };
  setTimeout(tick, 3000);
}

const SCREENS = [sWelcome, sAhead, sFiles, sHero, sAccess, sStart];
let shownAt = -2;
function render() {
  const main = $('ob-main');
  if (!st) {
    main.replaceChildren(h('section', { class: 'ob-card' }, h('h2', { text: err ? 'Could not reach the setup' : 'Loading' }), err ? h('p', { class: 'ob-err', text: err }) : null,
      err ? h('p', { text: fix }) : null, err ? h('div', { class: 'ob-nav' }, btn('Retry', load, 'primary')) : null));
    return;
  }
  const far = FURTHEST[st.phase];
  if (at < 0) at = st.phase === 'ahead' && !sessionStorage.getItem('ob-welcomed') ? 0 : far;
  at = Math.max(0, Math.min(at, far));
  if (st.phase === 'running' || st.phase === 'done') at = 5;
  stepper();
  const card = h('section', { class: 'ob-card' }, SCREENS[at]());
  main.replaceChildren(card);
  const live = (st.phase === 'running' ? 'Setup is running' : STEPS[at]);
  if (at !== shownAt) { $('ob-live').textContent = 'Step ' + (at + 1) + ' of 6: ' + live; if (shownAt !== -2) main.focus(); shownAt = at; }
}
load();
