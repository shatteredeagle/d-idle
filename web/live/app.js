// Core of the live display. Knows nothing about individual panels: it loads the
// modules listed in manifest.json (web/live/panels/*.js), matches them to the
// panels the Go hub reports by id, and routes snapshot/update/append messages.
import { h, clear } from './lib.js';

const grid = document.getElementById('grid');
const nav = document.getElementById('nav');
const conn = document.getElementById('conn');
const cards = new Map(); // id -> {card, body, mod}
const mods = new Map();  // id -> module
let snapSeq = 0;
let lastMsg = Date.now();

function setConn(s, text) { conn.dataset.s = s; conn.querySelector('b').textContent = text; }

async function loadModules() {
  let list = [];
  try { list = (await (await fetch('manifest.json')).json()).modules || []; } catch (e) { console.error(e); }
  await Promise.all(list.map(async (path) => {
    try {
      const m = (await import('./' + path)).default;
      if (m && m.id) mods.set(m.id, m);
    } catch (e) { console.error('panel module failed', path, e); mods.set('!' + path, { id: path, error: e }); }
  }));
}

function ensureCard(p) {
  let c = cards.get(p.id);
  if (c) { c.title.textContent = p.title; return c; }
  const mod = mods.get(p.id);
  const title = h('span', { text: p.title });
  const aside = h('span', { class: 'aside' });
  const body = h('div', { class: 'body' });
  const card = h('section', { class: 'card', id: 'panel-' + p.id, 'aria-labelledby': 'h-' + p.id }, h('h2', { id: 'h-' + p.id }, title, aside), body);
  if (mod && mod.span) card.style.setProperty('--span', mod.span);
  c = { card, body, title, aside, mod };
  cards.set(p.id, c);
  return c;
}

function render(p) {
  const c = ensureCard(p);
  try {
    if (c.mod && c.mod.render) c.mod.render(c.body, p.data, { aside: c.aside });
    else { clear(c.body).append(h('pre', { class: 'raw', text: JSON.stringify(p.data, null, 2) })); }
  } catch (e) {
    console.error('render failed', p.id, e);
    clear(c.body).append(h('p', { class: 'empty-note', text: 'This panel failed to render: ' + e.message }));
  }
}

function layout(panels) {
  const ids = panels.map((p) => p.id);
  for (const id of [...cards.keys()]) if (!ids.includes(id)) { cards.get(id).card.remove(); cards.delete(id); }
  clear(grid); clear(nav);
  for (const p of panels) {
    grid.append(cards.get(p.id).card);
    nav.append(h('a', { href: '#panel-' + p.id, text: p.title }));
  }
}

function onSnapshot(s) {
  snapSeq = s.seq;
  for (const p of s.panels) ensureCard(p);
  layout(s.panels);
  s.panels.forEach(render);
}

function connect() {
  const es = new EventSource('api/events');
  es.addEventListener('open', () => setConn('ok', 'live'));
  es.addEventListener('snapshot', (e) => { lastMsg = Date.now(); setConn('ok', 'live'); onSnapshot(JSON.parse(e.data)); });
  es.addEventListener('update', (e) => {
    lastMsg = Date.now();
    const m = JSON.parse(e.data);
    if (m.seq <= snapSeq) return;
    const c = cards.get(m.id);
    if (c) render({ id: m.id, title: c.title.textContent, data: m.data });
  });
  es.addEventListener('append', (e) => {
    lastMsg = Date.now();
    const m = JSON.parse(e.data);
    const c = cards.get(m.id);
    if (m.seq > snapSeq && c && c.mod && c.mod.append) c.mod.append(c.body, m.item, { aside: c.aside });
  });
  es.onerror = () => setConn('bad', 'reconnecting');
}

setInterval(() => { if (conn.dataset.s === 'ok' && Date.now() - lastMsg > 40000) setConn('stale', 'no data'); }, 5000);

// theme: auto / light / dark
const themeBtn = document.getElementById('theme');
function applyTheme(t) {
  if (t === 'auto') document.documentElement.removeAttribute('data-theme'); else document.documentElement.dataset.theme = t;
  themeBtn.textContent = 'theme: ' + t;
}
let theme = 'auto';
try { theme = localStorage.getItem('d2live-theme') || 'auto'; } catch (e) { /* storage unavailable */ }
applyTheme(theme);
themeBtn.addEventListener('click', () => {
  theme = theme === 'auto' ? 'light' : theme === 'light' ? 'dark' : 'auto';
  applyTheme(theme);
  try { localStorage.setItem('d2live-theme', theme); } catch (e) { /* ignore */ }
});

await loadModules();
connect();
