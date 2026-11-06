'use strict';
// d2d first-run wizard page. All text is inserted with textContent. There is deliberately no CD key field.
const $ = (id) => document.getElementById(id);
function h(tag, attrs, ...kids) {
  const el = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs || {})) {
    if (k === 'text') el.textContent = v;
    else if (k.startsWith('on')) el.addEventListener(k.slice(2), v);
    else el.setAttribute(k, v);
  }
  for (const k of kids.flat()) if (k != null) el.append(k);
  return el;
}
const STEPS = ['Game files', 'Character', 'Start'];
const STEP_OF = { data: 0, character: 1, start: 2, running: 2, failed: 2, done: 2 };
let st = null, editing = false, err = '', timer = 0, watching = false;

async function api(method, path, body) {
  const r = await fetch('/api' + path, { method, cache: 'no-store', credentials: 'same-origin',
    headers: body ? { 'Content-Type': 'application/json' } : undefined, body: body ? JSON.stringify(body) : undefined });
  const j = await r.json().catch(() => ({}));
  if (!r.ok) throw new Error(j.error || 'request failed (' + r.status + ')');
  return j;
}
async function load() {
  try { st = await api('GET', '/state'); err = ''; } catch (e) { err = st && st.phase === 'done' ? '' : e.message; }
  render();
  clearTimeout(timer);
  if (st && (st.phase === 'running' || st.phase === 'done')) timer = setTimeout(load, 1500);
}
async function act(fn) {
  try { err = ''; st = await fn(); } catch (e) { err = e.message; }
  render();
  clearTimeout(timer);
  if (st && st.phase === 'running') timer = setTimeout(load, 1500);
}

function stepper() {
  const cur = st ? STEP_OF[st.phase] : 0;
  const ol = h('ol', { class: 'wz-steps' }, STEPS.map((s, i) =>
    h('li', { class: i < cur || (st && st.phase === 'done') ? 'done' : i === cur ? 'cur' : '', ...(i === cur ? { 'aria-current': 'step' } : {}) }, h('span', { text: (i + 1) + '. ' + s }))));
  $('wz-stepper').replaceChildren(ol);
}
function fileRow(f) {
  const st = { pass: 'pass', fail: 'fail', warn: 'warn', info: 'skip' }[f.status] || 'skip';
  return h('li', { class: 'wz-check ' + st },
    h('div', { class: 'wz-check-head' }, h('strong', { text: f.name }), h('span', { class: 'badge ' + st, text: f.status + (f.required ? '' : ' (optional)') })),
    h('p', { text: f.message }));
}
function dataCard() {
  const c = st.check, ok = c.ok;
  const bad = c.files.filter((f) => f.status === 'fail' || f.status === 'warn');
  const rest = c.files.filter((f) => f.status === 'pass' || f.status === 'info');
  return h('section', { class: 'wz-card' },
    h('h2', { text: 'Your game files' }),
    h('p', { class: 'wz-lead', role: 'status', text: c.summary }),
    h('p', { class: 'wz-hint', text: 'Folder: ' + c.dir + '. d.idle reads your own Diablo II Lord of Destruction files (version 1.14d) and never downloads game data.' }),
    bad.length ? h('ul', { class: 'wz-checks' }, bad.map(fileRow)) : null,
    rest.length ? h('details', { class: 'wz-passed', ...(ok ? {} : {}) }, h('summary', { text: 'Found (' + rest.length + ')' }), h('ul', { class: 'wz-checks' }, rest.map(fileRow))) : null,
    ok ? h('p', { class: 'wz-hint', text: c.version === '1.14d' ? 'Version: 1.14d confirmed.' : 'Version: not confirmed yet, the Patch_D2 fingerprint is not on our list. The bot will try anyway.' }) : null,
    h('div', { class: 'wz-actions' }, ok ? null : h('button', { class: 'btn primary big', type: 'button', text: 'Check again', onclick: () => act(() => api('POST', '/check')) })));
}
function field(id, label, control, hint) {
  return h('div', { class: 'wz-field' }, h('label', { for: id, text: label }), control, hint ? h('div', { class: 'hint', text: hint }) : null);
}
function formCard() {
  const nameIn = h('input', { type: 'text', id: 'f-name', maxlength: '15', autocomplete: 'off', spellcheck: 'false', value: st.inputs.char_name || '' });
  const cls = h('select', { id: 'f-class' }, st.classes.map((c) => {
    const o = h('option', { value: c.id, text: c.name + (c.supported ? '' : ' (not supported in this build)') });
    if (!c.supported) o.disabled = true;
    if (c.id === st.inputs.class) o.selected = true;
    return o;
  }));
  const goals = h('select', { id: 'f-goals' }, st.presets.map((p) => {
    const o = h('option', { value: p.id, text: p.name });
    if (p.id === st.inputs.goals) o.selected = true;
    return o;
  }));
  const e = h('div', { class: 'wz-err', id: 'e-form', role: 'alert' });
  const form = h('form', { novalidate: 'true', style: 'display:grid;gap:18px', onsubmit: (ev) => {
    ev.preventDefault();
    const n = nameIn.value.trim();
    if (!/^[A-Za-z]{2,15}$/.test(n)) { e.textContent = 'Use 2 to 15 letters, no digits or spaces.'; nameIn.setAttribute('aria-invalid', 'true'); return; }
    editing = false;
    act(() => api('PUT', '/inputs', { char_name: n, class: cls.value, goals: goals.value }));
  } },
    field('f-name', 'Character name', nameIn, '2 to 15 letters. A character with this name is created, or loaded if it already exists.'),
    field('f-class', 'Class', cls, 'This build plays the Sorceress. The other classes are listed but not supported yet.'),
    field('f-goals', 'Goal', goals),
    e,
    h('div', { class: 'wz-note', text: 'No CD key is needed or accepted here. The game runs offline.' }),
    h('div', { class: 'wz-actions' }, h('button', { class: 'btn primary big', type: 'submit', text: 'Save and continue' })));
  return h('section', { class: 'wz-card' }, h('h2', { text: 'Your character' }), form);
}
function summaryCard() {
  const cn = (st.classes.find((c) => c.id === st.inputs.class) || {}).name || st.inputs.class;
  const pn = (st.presets.find((p) => p.id === st.inputs.goals) || {}).name || st.inputs.goals;
  return h('section', { class: 'wz-card' },
    h('h2', { text: 'Ready to start' }),
    h('p', { class: 'wz-lead', text: st.inputs.char_name + ', ' + cn + '. Goal: ' + pn }),
    st.phase === 'failed' ? h('div', { class: 'wz-note bad', role: 'alert' }, h('strong', { text: 'That did not work' }), st.error) : null,
    h('p', { class: 'wz-hint', text: 'Start reads the game tables once (a minute or so) and then hands over to the watch screen.' }),
    h('div', { class: 'wz-actions' },
      h('button', { class: 'btn primary big', type: 'button', text: st.phase === 'failed' ? 'Try again' : 'Start', onclick: () => act(() => api('POST', '/start')) }),
      h('button', { class: 'btn ghost', type: 'button', text: 'Change my answers', onclick: () => { editing = true; render(); } })));
}
function runningCard() {
  return h('section', { class: 'wz-card' }, h('h2', { text: 'Reading your game files' }),
    h('div', { class: 'wz-run', role: 'status' }, h('span', { class: 'spin', 'aria-hidden': 'true' }), h('span', { text: 'Working. Keep this page open; refreshing is fine.' })));
}
function doneCard() {
  if (st.token) sessionStorage.setItem('d2tok', st.token);
  const tok = sessionStorage.getItem('d2tok');
  if (!watching) { watching = true; waitForWatch(); }
  return h('section', { class: 'wz-card' }, h('h2', { text: 'Starting the watch screen' }),
    h('div', { class: 'wz-run', role: 'status' }, h('span', { class: 'spin', 'aria-hidden': 'true' }), h('span', { text: 'The bot is starting. This page moves on by itself.' })),
    tok ? h('div', { class: 'wz-note' }, h('strong', { text: 'Your watch token (shown once)' }), h('code', { class: 'wz-inline', text: tok }),
      h('p', { class: 'wz-hint', text: 'Open the watch screen from another device with /watch?t=TOKEN. It is also in the data volume as watch.token.' })) : null,
    h('p', null, h('a', { href: '/watch', text: 'Open the watch screen now' })));
}
function waitForWatch() {
  const tick = async () => {
    try { const r = await fetch('/watch', { cache: 'no-store', credentials: 'same-origin' }); if (r.ok) { location.assign('/watch'); return; } } catch (_) { /* the wizard is handing over the port */ }
    setTimeout(tick, 2000);
  };
  setTimeout(tick, 3000);
}
function render() {
  const main = $('wz-main');
  stepper();
  if (!st) { main.replaceChildren(h('section', { class: 'wz-card' }, h('h2', { text: 'Could not reach the setup' }), h('p', { class: 'wz-err', text: err }),
    h('div', { class: 'wz-actions' }, h('button', { class: 'btn primary', type: 'button', text: 'Try again', onclick: load })))); return; }
  const kids = [];
  if (err) kids.push(h('div', { class: 'wz-note bad', role: 'alert' }, h('strong', { text: 'Something went wrong' }), err));
  switch (st.phase) {
    case 'data': kids.push(dataCard()); break;
    case 'character': kids.push(formCard()); break;
    case 'start': case 'failed': kids.push(editing ? formCard() : summaryCard()); break;
    case 'running': kids.push(runningCard()); break;
    case 'done': kids.push(doneCard()); break;
  }
  main.replaceChildren(...kids);
  $('wz-live').textContent = st.phase;
}
load();
