// The watch shell (docs/ui-design-system.md section 4): theme and modes, panel placement from /watch/panels.json, one
// fetch scheduler (the SSE change event plus per-panel polls), focus keeping across updates, and the ctx a panel gets.
// Text only through textContent; no inline styles (the page CSP is style-src 'self').

const SAFETY_POLL = 5000; // a missed SSE event costs at most this long
const FAIL_WAIT = 4000;   // after a failed fetch
const FETCH_TIMEOUT = 8000; // ms; a panel request older than this is aborted
const MAX_FAILS = 5;      // 404 or 401 this often in a row: the route is not mounted, stop polling it

const html = document.documentElement;
const qs = new URLSearchParams(location.search);

function el(tag, cls, text) {
  const e = document.createElement(tag);
  if (cls) e.className = cls;
  if (text !== undefined) e.textContent = text;
  return e;
}

// ---- theme (dark default, user 2026-11-09) and modes ----
function setupTheme() {
  const btn = document.getElementById('btn-theme');
  let theme = 'dark';
  try { theme = localStorage.getItem('d2theme') || 'dark'; } catch (e) { /* storage blocked */ }
  if (/^(dark|light|auto)$/.test(qs.get('theme') || '')) theme = qs.get('theme'); // ?theme= for links and screenshots, not stored
  const apply = (t) => {
    if (t !== 'light' && t !== 'auto') t = 'dark';
    html.dataset.theme = t;
    if (btn) btn.textContent = 'Theme: ' + t;
  };
  if (btn) btn.addEventListener('click', () => {
    theme = { dark: 'light', light: 'auto', auto: 'dark' }[theme] || 'dark';
    try { localStorage.setItem('d2theme', theme); } catch (e) { /* ignore */ }
    apply(theme);
  });
  apply(theme);
  const mode = qs.get('mode');
  if (mode === 'overlay') html.dataset.mode = mode;
}

// ---- component factories (components.css): panels build their DOM with these and textContent only ----
const ui = {
  // a d-meter; set(pct, text, level) moves the fill through a custom property (allowed under style-src 'self')
  meter(kind, label) {
    const m = el('div', 'd-meter');
    m.dataset.kind = kind;
    m.setAttribute('role', 'meter');
    m.setAttribute('aria-valuemin', '0');
    m.setAttribute('aria-valuemax', '100');
    m.setAttribute('aria-label', label);
    const track = el('span', 'track'), fill = el('span', 'fill'), num = el('span', 'num');
    track.appendChild(fill);
    m.append(el('span', 'lbl', label), track, num);
    return {
      root: m,
      set(pct, text, level) {
        const p = Math.max(0, Math.min(100, pct || 0));
        fill.style.setProperty('--fill', p + '%');
        m.setAttribute('aria-valuenow', String(Math.round(p)));
        num.textContent = text;
        if (level) m.dataset.level = level; else delete m.dataset.level;
      },
    };
  },
  chip(text, level) { const c = el('span', 'd-chip', text); if (level) c.dataset.level = level; return c; },
  badge(text, tone) { const b = el('span', 'd-badge', text); if (tone) b.dataset.tone = tone; return b; },
  btn(text, opts) {
    const b = el('button', 'd-btn', text);
    b.type = 'button';
    Object.assign(b.dataset, opts || {});
    return b;
  },
  stat(label) {
    const s = el('div', 'd-stat'), val = el('span', 'val'), delta = el('span', 'delta');
    s.append(el('span', 'lbl', label), val, delta);
    return { root: s, set(v, d) { val.textContent = v; delta.textContent = d || ''; } };
  },
  clear(n) { while (n.firstChild) n.removeChild(n.firstChild); },
};

// ---- toasts ----
function toast(sev, text) {
  const box = document.getElementById('toasts');
  if (!box) return;
  const t = el('div', 'd-toast');
  t.dataset.sev = sev || 'info';
  t.appendChild(el('span', '', text));
  const b = el('button', 'd-btn', 'Dismiss');
  b.type = 'button'; b.dataset.size = 'sm'; b.dataset.variant = 'ghost';
  b.addEventListener('click', () => t.remove());
  t.appendChild(b);
  box.appendChild(t);
  while (box.children.length > 4) box.firstChild.remove();
  setTimeout(() => t.remove(), 8000);
}

// ---- panel roots ----
// Status-band panels are bare (no card); stage and grid panels are cards with a head (title, meta, actions) and body.
function makeRoot(m) {
  const card = m.band !== 'status';
  const root = el(card ? 'section' : 'div', card ? 'd-card d-panel' : 'd-panel');
  root.dataset.panel = m.id;
  root.dataset.size = m.size;
  root.dataset.tab = m.tab;
  root.dataset.state = 'loading';
  const parts = { root, body: root, head: null, meta: null, actions: null };
  if (card) {
    const head = el('div', 'd-card-head');
    const h = el('h2', '', m.title);
    h.id = 'p-' + m.id + '-title';
    root.setAttribute('aria-labelledby', h.id);
    head.appendChild(h);
    parts.meta = el('span', 'meta');
    parts.actions = el('span', 'd-btngroup');
    head.appendChild(parts.meta);
    head.appendChild(parts.actions);
    root.appendChild(head);
    parts.body = el('div', 'd-card-body');
    if (m.skeleton > 0) {
      const sk = el('div', 'd-skel');
      sk.setAttribute('aria-hidden', 'true');
      for (let i = 0; i < m.skeleton; i++) sk.appendChild(el('i'));
      root.appendChild(sk);
    }
    root.appendChild(parts.body);
    parts.head = head;
  } else {
    root.setAttribute('aria-label', m.title);
  }
  return parts;
}

function place(bands, m, root) {
  const band = bands[m.band];
  if (!band) return false;
  // the status band keeps size-l panels on their own row and the rest inline in its main slot
  if (m.band === 'status' && m.size !== 'l') {
    (band.querySelector('.d-band-main') || band).appendChild(root);
    return true;
  }
  const compat = band.querySelector(':scope > .d-compat');
  if (compat) band.insertBefore(root, compat); else band.appendChild(root);
  return true;
}

// A band with no visible panel is hidden (the stage disappears when there is neither video nor map).
function syncBand(band) {
  if (!band || !band.matches('.d-band-stage, .d-band-side')) return;
  band.hidden = !band.querySelector(':scope > .d-panel:not([hidden])');
}

// Focus keeping: the focused element is found again by data-key after an update rebuilt the panel's DOM.
function keepFocus(root, fn) {
  const a = document.activeElement;
  const key = a && root.contains(a) && a.dataset ? a.dataset.key : '';
  const x = window.scrollX, y = window.scrollY, boxes = [];
  for (const n of root.querySelectorAll('*')) if (n.scrollTop > 0) boxes.push([n, n.scrollTop]);
  fn();
  for (const [n, t] of boxes) if (n.isConnected) n.scrollTop = t;
  if (window.scrollX !== x || window.scrollY !== y) window.scrollTo(x, y);
  if (key && !root.contains(document.activeElement)) {
    const b = root.querySelector('[data-key="' + CSS.escape(key) + '"]');
    if (b) b.focus({ preventScroll: true }); // a poll must never scroll the page
  }
}

// ---- the scheduler ----
function makeScheduler() {
  const slots = [];
  let changed = false;
  function due(s, now) {
    if (s.stopped || s.inflight || !s.m.endpoint) return false;
    if (document.hidden && s.m.band !== 'status') return false;
    if (s.m.poll === 0 && changed) return true;
    return now >= s.next;
  }
  function run(s) {
    s.inflight = true;
    const q = s.api.query ? s.api.query() : '';
    // A stalled request (a proxy, the browser's per-host connection cap behind the video and event streams) must not
    // hold the slot for ever: abort it, count a failure and poll again.
    const ac = typeof AbortController === 'function' ? new AbortController() : null;
    const timer = ac ? setTimeout(() => ac.abort(), FETCH_TIMEOUT) : 0;
    fetch(s.m.endpoint + q, { cache: 'no-store', credentials: 'same-origin', signal: ac ? ac.signal : undefined }).then((r) => {
      if (!r.ok) { const e = new Error('status ' + r.status); e.status = r.status; throw e; }
      return r.json();
    }).then((d) => {
      s.fails = 0;
      keepFocus(s.root, () => s.api.update && s.api.update(d));
      if (s.root.dataset.state === 'loading') s.root.dataset.state = 'ready';
    }).catch((e) => {
      s.fails++;
      // a throw here would skip the slot release below and freeze the panel for good
      try { if (s.api.update) keepFocus(s.root, () => s.api.update(null, { status: e.status || 0, fails: s.fails })); } catch (e2) { /* keep polling */ }
      // a route that is not mounted stops polling; anything else (503 while the bot starts, a network blip) retries slower
      if (s.fails >= MAX_FAILS && (e.status === 404 || e.status === 401)) s.stopped = true;
    }).then(() => {
      clearTimeout(timer);
      s.inflight = false;
      const p = s.m.poll > 0 ? s.m.poll * 1000 : SAFETY_POLL;
      s.next = performance.now() + (s.fails ? Math.min(30000, Math.max(p, FAIL_WAIT * s.fails)) : p);
      syncBand(s.root.parentElement);
    });
  }
  function tick() {
    const now = performance.now();
    // one panel throwing must neither stop the other polls nor the loop itself
    slots.forEach((s) => { try { if (due(s, now)) run(s); } catch (e) { s.inflight = false; s.fails++; s.next = now + FAIL_WAIT; } });
    changed = false;
    setTimeout(tick, 250);
  }
  if (window.EventSource) {
    const es = new EventSource('/api/events');
    es.addEventListener('change', () => { changed = true; });
  }
  return { add(s) { s.next = 0; s.fails = 0; slots.push(s); }, start: tick };
}

// ---- ctx and mounting ----
function makeCtx(m, parts, cfg, answerBase, timers) {
  const store = (key) => {
    const k = 'd2' + m.id + key; // 'd2' + 'video' + 'collapsed' keeps the pre-shell key d2videocollapsed
    return {
      get() { try { return localStorage.getItem(k); } catch (e) { return null; } },
      set(v) { try { localStorage.setItem(k, v); } catch (e) { /* storage blocked */ } },
    };
  };
  return {
    id: m.id, config: cfg, token: qs.get('t') || '',
    body: parts.body, meta: parts.meta, actions: parts.actions,
    el, ui, store, toast,
    hide(on) { parts.root.hidden = !!on; syncBand(parts.root.parentElement); },
    state(st) { parts.root.dataset.state = st; },
    later(fn, ms) { const t = setTimeout(() => { timers.delete(t); fn(); }, ms); timers.add(t); return t; },
    post(path, body) {
      return fetch(answerBase + path, {
        method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body || {}),
      }).then((r) => r.json()).catch(() => { toast('alarm', 'the answer route is not reachable'); return null; });
    },
  };
}

async function start() {
  setupTheme();
  const bands = {};
  document.querySelectorAll('[data-band]').forEach((b) => { bands[b.dataset.band] = b; });
  let cfg = {};
  try { cfg = await (await fetch('/api/config', { cache: 'no-store' })).json(); } catch (e) { /* the old code says so */ }
  const answerBase = cfg.answerBase || (cfg.answerPort ? location.protocol + '//' + location.hostname + ':' + cfg.answerPort : '');
  let list = [];
  try { list = (await (await fetch('/watch/panels.json', { cache: 'no-store' })).json()).panels || []; } catch (e) { return; }
  const sched = makeScheduler();
  for (const m of list) {
    const parts = makeRoot(m);
    if (!place(bands, m, parts.root)) continue;
    const css = el('link');
    css.rel = 'stylesheet'; css.href = '/watch/panels/' + m.id + '/panel.css';
    document.head.appendChild(css);
    const timers = new Set();
    let api = {};
    try {
      const mod = await import('/watch/panels/' + m.id + '/panel.js');
      api = mod.mount(parts.root, makeCtx(m, parts, cfg, answerBase, timers)) || {};
    } catch (e) {
      parts.root.dataset.state = 'error';
      parts.body.appendChild(el('p', 'd-empty', 'This panel failed to load.'));
      continue;
    }
    if (!m.endpoint && parts.root.dataset.state === 'loading') parts.root.dataset.state = 'ready';
    sched.add({ m, root: parts.root, api });
    syncBand(parts.root.parentElement);
  }
  Object.values(bands).forEach(syncBand);
  sched.start();
}

start();
