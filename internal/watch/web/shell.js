// The watch shell (docs/ui-design-system.md section 4): theme and modes, panel placement from /watch/panels.json, one
// fetch scheduler (the SSE change event plus per-panel polls), focus keeping across updates, and the ctx a panel gets.
// Text only through textContent; no inline styles (the page CSP is style-src 'self').

import '/watch/toast.js';
import { mountReset } from '/watch/resetui.js';
import { mountPause } from '/watch/pauseui.js';
import { mountSupport } from '/watch/supportui.js';
import { mountLootRules } from '/watch/lootrulesui.js';
import { overlays } from '/watch/overlay.js';
import { COCKPIT_SHORT, plan, breakpointFor, fitRows, contentHeight, selectPanels, withShadows, PRESETS, isCockpit, cockpitPlan, effectiveChoice, missingSlots } from '/watch/layout.js';

const SAFETY_POLL = 5000; // a missed SSE event costs at most this long
const FAIL_WAIT = 4000;   // after a failed fetch
const FETCH_TIMEOUT = 8000; // ms; a panel request older than this is aborted
const MAX_FAILS = 5;      // 404 or 401 this often in a row: the route is not mounted, stop polling it

const html = document.documentElement;
const qs = new URLSearchParams(location.search);
const CHARS_URL = '/watch/characters' + (/^(dark|light|auto)$/.test(qs.get('theme') || '') ? '?theme=' + qs.get('theme') : '');
if (qs.get('view') === 'characters') location.replace(CHARS_URL); // /watch?view=characters is the Characters page (docs/charselect-ui.md)

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

// ---- toasts ---- the one toast (web/toast.js, window.dToast); act ({label, run, ms}): one action button (Undo), the action runs once.
function toast(sev, text, act) { return window.dToast.show(sev, text, act); }

// A sheet over a tile: the shell owns overlays. where 'self' covers the panel's own tile; 'beside' is the area-map tile when it
// stands to the right of this one on the same rows (a two-column rail), else the Orders tile above this one, else the own tile
// (a phone tab). place() moves the sheet to where it belongs now; the zone menu uses it so it never covers the zone buttons.
// The sheet carries data-panel of its panel, so that panel's own selectors style what it holds.
function tileSheet(id, root, where) {
  const sheet = el('div', 'd-tilesheet');
  sheet.dataset.panel = id;
  sheet.dataset.scroll = '1'; // a sheet taller than its tile scrolls inside it
  sheet.hidden = true;
  overlays.register('sheet-' + id, sheet);
  const shows = (t) => { if (!t || t.hidden || t.dataset.off || t === root) return null; const r = t.getBoundingClientRect(); return r.width > 40 && r.height > 40 ? r : null; };
  const host = () => {
    if (where !== 'beside') return root;
    const me = root.getBoundingClientRect(), side = document.querySelector('.d-panel[data-panel="area-map"]'), sr = shows(side);
    if (sr && sr.left >= me.right - 4 && sr.top < me.bottom && sr.bottom > me.top) return side;
    const up = document.querySelector('.d-panel[data-panel="orders"]'), ur = shows(up);
    if (ur && ur.bottom <= me.top + 4) return up;
    return root;
  };
  const place = () => { const h = host(); if (sheet.parentElement !== h) h.appendChild(sheet); sheet.dataset.host = h === root ? 'self' : h.dataset.panel; return sheet.dataset.host; };
  // A tile that a short window leaves only a few rows cannot hold its sheet (the Orders More list showed one row at 870x520, card
  // morefit): the sheet then floats over the page at the tile's top-left (position: fixed is not clipped by the tile), at least
  // FLOAT_MIN tall and never past the window; it keeps the tile's width. A tall enough tile keeps the sheet inside it.
  const FLOAT_MIN = 232;
  const fit = () => {
    if (where === 'beside') return; // the zone menu has its own placement (it must not cover the zone buttons)
    if (sheet.hidden) { sheet.removeAttribute('data-float'); sheet.style.removeProperty('--ts-top'); return; }
    const r = sheet.parentElement ? sheet.parentElement.getBoundingClientRect() : null;
    if (!r || r.height >= FLOAT_MIN) { sheet.removeAttribute('data-float'); return; }
    const h = Math.min(FLOAT_MIN, window.innerHeight - 8), top = Math.max(0, Math.min(r.top, window.innerHeight - h - 4));
    sheet.dataset.float = '1';
    sheet.style.setProperty('--ts-top', Math.round(top) + 'px'); sheet.style.setProperty('--ts-left', Math.round(r.left) + 'px');
    sheet.style.setProperty('--ts-w', Math.round(r.width) + 'px'); sheet.style.setProperty('--ts-h', Math.round(h) + 'px');
  };
  new MutationObserver(fit).observe(sheet, { attributes: true, attributeFilter: ['hidden'] });
  window.addEventListener('resize', fit);
  return { el: sheet, place };
}

// ---- panel roots ----
// Status-band panels are bare (no card); stage and grid panels are cards with a head (title, meta, actions) and body.
function makeRoot(m, inCockpit) {
  if (inCockpit) return makeSlotRoot(m);
  const card = m.band !== 'status';
  const root = el(card ? 'section' : 'div', card ? 'd-card d-panel' : 'd-panel');
  root.dataset.panel = m.id;
  root.dataset.size = m.size;
  root.dataset.tab = m.tab;
  root.dataset.state = 'loading';
  if (m.fit) root.dataset.fit = '1'; // its rows follow its content (layout.js fitRows)
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

// A panel in a cockpit slot is bare: no card, no title bar, its content fills the box the slot gives it. The title bar's
// two parts (meta, actions) become small labels on the box's edge, so a panel that fills them (the picture's "Full
// screen") works unchanged.
function makeSlotRoot(m) {
  const root = el('div', 'd-panel');
  root.dataset.panel = m.id;
  root.dataset.size = m.size;
  root.dataset.tab = m.tab;
  root.dataset.state = 'loading';
  root.dataset.cockpitPanel = m.band === 'status' ? 'band' : 'slot';
  root.setAttribute('aria-label', m.title);
  const body = el('div', 'd-slotbody'), labels = el('div', 'd-slotlabels');
  const meta = el('span', 'meta'), actions = el('span', 'd-btngroup');
  labels.append(meta, actions);
  root.append(body, labels);
  return { root, body, head: null, meta, actions };
}

function place(bands, m, root) {
  const band = bands[m.band];
  if (!band) return false;
  // the status band keeps size-l panels on their own row and the rest inline in its main slot
  if (m.band === 'status' && m.size !== 'l') {
    (band.querySelector('.d-band-main') || band).appendChild(root);
    return true;
  }
  band.appendChild(root);
  return true;
}

// A band with no visible panel is hidden (the stage disappears when there is neither video nor map).
function syncBand(band) {
  if (band && band.matches('.d-slotcol, .d-slottabs')) { growLast(band); return; }
  if (!band || !band.matches('.d-band-stage, .d-band-side')) return;
  band.hidden = !band.querySelector(':scope > .d-panel:not([hidden])');
}

// In a cockpit column the last panel that shows takes the rest of the height (rows "auto, auto, 1fr"); a hidden
// panel (Needs you with nothing to say) takes none.
function growLast(col) {
  if (col.dataset.nogrow) { for (const n of col.children) delete n.dataset.grow; return; }
  const shown = [...col.children].filter((n) => n.matches('.d-panel') && !n.hidden && !n.dataset.off);
  for (const n of col.children) delete n.dataset.grow;
  if (shown.length) shown[shown.length - 1].dataset.grow = '1';
  // the growing panel's least height (manifest minBlock, px): the panels above it give way (shell.css, data-growmin) so the act map
  // keeps its ten readable rows on a short window
  const min = shown.length ? (ck.meta.get(shown[shown.length - 1].dataset.panel) || {}).minBlock | 0 : 0;
  if (min > 0) { col.dataset.growmin = '1'; col.style.setProperty('--c-growmin', min + 'px'); } else { delete col.dataset.growmin; col.style.removeProperty('--c-growmin'); }
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
let theSched = null; // the page's scheduler (ctx.refresh and the drawers poke it)
function makeScheduler() {
  const slots = [];
  let changed = false;
  function due(s, now) {
    if (s.stopped || s.inflight || !s.m.endpoint) return false;
    if (document.hidden && s.m.band !== 'status') return false;
    if (s.m.drawer && s.root.dataset.open !== '1') return false; // a drawer panel polls only while its drawer is open
    if (s.m.poll === 0 && changed) return true;
    return now >= s.next;
  }
  // One request per address at a time, whoever asks: five panels read /api/hero, and five requests every two seconds
  // (each slow when the bot is busy) held the browser's few connections per host next to the video and event streams,
  // so every other panel and card starved in its loading state. Panels that are due together share one reply.
  const flying = new Map();
  function load(url) {
    let p = flying.get(url);
    if (p) return p;
    const ac = typeof AbortController === 'function' ? new AbortController() : null;
    const timer = ac ? setTimeout(() => ac.abort(), FETCH_TIMEOUT) : 0;
    p = fetch(url, { cache: 'no-store', credentials: 'same-origin', signal: ac ? ac.signal : undefined }).then((r) => {
      if (!r.ok) { const e = new Error('status ' + r.status); e.status = r.status; throw e; }
      return r.json();
    }).finally(() => { clearTimeout(timer); flying.delete(url); });
    flying.set(url, p);
    return p;
  }
  function run(s) {
    s.inflight = true;
    const q = s.api.query ? s.api.query() : '';
    // A stalled request (a proxy, the browser's per-host connection cap behind the video and event streams) must not
    // hold the slot for ever: abort it, count a failure and poll again.
    load(s.m.endpoint + q).then((d) => {
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
    // the page's one event stream (window.d2events); another script may have opened it already
    const es = window.d2events || (window.d2events = new EventSource('/api/events'));
    es.addEventListener('change', () => { changed = true; });
  }
  // A page restored from the back-forward cache (the Back button from the Vault) keeps its in-flight requests, which the browser
  // killed while it was frozen: forget them and poll everything now instead of waiting for a timeout that may never come.
  const rewake = () => { flying.clear(); slots.forEach((s) => { s.inflight = false; s.stopped = false; s.fails = 0; s.next = 0; }); };
  return { add(s) { s.next = 0; s.fails = 0; slots.push(s); }, start: tick, rewake, poke(id) { slots.forEach((s) => { if (s.m.id === id) s.next = 0; }); } };
}

// ---- layout manifest (docs/ui-design-system.md section 6.4) ----
// The layout in force comes from GET /api/layout (web/layout.json, or the profile's stored copy); ?layout=<preset>
// loads one of the editor's presets instead and ?layout=none keeps the manifest placement. Every panel root in the
// main area is a target; the plan (layout.js) says where each goes. Without a
// layout, or for a breakpoint the file does not cover, nothing is set and the manifest rules of shell.css place.
const lay = { layout: null, conf: {}, targets: new Map(), plan: null, main: null, editor: null, bp: '', warned: false, rows: new Map(), fitAt: 0, fitQueued: false };

function layoutTargets() {
  const out = [];
  lay.targets.clear();
  lay.main.querySelectorAll('.d-panel[data-panel]').forEach((n) => {
    const id = n.dataset.panel;
    lay.targets.set(id, n);
    out.push({ id, size: n.dataset.size || 's', hidden: n.hidden, rows: n.dataset.fit ? lay.rows.get(id) : 0 });
  });
  return out;
}

// Fit content: measure what every fit panel needs at its present width; true when a span changed (the caller plans
// again). Only on the grid breakpoints (the flow ones have no fixed rows) and never while the editor owns the cells.
function measureFit() {
  const p = lay.plan;
  if (!p || p.flow || p.bp === 'overlay' || lay.editor) return false;
  const gap = parseFloat(getComputedStyle(lay.main).rowGap) || 0;
  let changed = false;
  for (const it of p.items) {
    const n = lay.targets.get(it.id);
    if (!n || !n.dataset.fit || it.hidden || it.collapsed || n.hidden) continue;
    const h = contentHeight(n, getComputedStyle(n));
    if (!h) continue;
    const rows = fitRows(h, p.rowHeight, gap);
    n.dataset.rows = String(rows);
    if (lay.rows.get(it.id) !== rows) { lay.rows.set(it.id, rows); changed = true; }
  }
  return changed;
}
// Content changes (a poll redraws a card, the picture collapses, the window is resized) ask for a new fit; at most
// one measure per 300 ms, and a new plan only when a span really changed.
function scheduleFit() {
  if (lay.fitQueued || lay.editor) return;
  lay.fitQueued = true;
  const run = () => { lay.fitQueued = false; lay.fitAt = performance.now(); if (measureFit()) applyLayout(1); };
  const wait = 300 - (performance.now() - lay.fitAt);
  if (wait > 0) setTimeout(() => requestAnimationFrame(run), wait); else requestAnimationFrame(run);
}

function applyLayout(pass) {
  const main = lay.main;
  if (!main) return;
  const targets = layoutTargets();
  const p = lay.layout ? plan(lay.layout, window.innerWidth, html.dataset.mode || '', lay.conf, targets, !!lay.editor) : null;
  lay.plan = p;
  lay.bp = breakpointFor(window.innerWidth, html.dataset.mode || '');
  lay.targets.forEach((n) => { n.style.gridColumn = ''; n.style.gridRow = ''; n.style.order = ''; delete n.dataset.collapsed; delete n.dataset.layoutOff; });
  if (!p || p.bp === 'overlay') { delete main.dataset.layout; return; }
  main.dataset.layout = p.flow ? 'flow' : 'grid';
  main.style.setProperty('--l-cols', String(p.cols));
  main.style.setProperty('--l-rowh', p.rowHeight + 'px');
  for (const id of p.off) { const n = lay.targets.get(id); if (n) n.dataset.layoutOff = '1'; } // switched off in the editor
  for (const it of p.items) {
    const n = lay.targets.get(it.id);
    if (!n || it.hidden) continue;
    // flow: the span and the order only, so cards fill each row left to right whatever is hidden by a phone tab
    n.style.gridColumn = p.flow ? 'span ' + it.colSpan : it.col + ' / span ' + it.colSpan;
    if (p.flow) n.style.order = String(it.order);
    else n.style.gridRow = it.row + ' / span ' + (it.collapsed ? 1 : it.rowSpan);
    if (it.collapsed) n.dataset.collapsed = '1';
  }
  if (p.unknown.length && !lay.warned) {
    lay.warned = true;
    console.warn('watch layout: no panel for ' + p.unknown.join(', ') + ' (ignored)');
  }
  // the widths are set: measure the fit panels and, when a span changed, plan once more (twice at most: the content
  // height depends on the width only, so the second plan is final)
  if ((pass || 0) < 2 && measureFit()) applyLayout((pass || 0) + 1);
}

// Which layout is in force (docs/ui/redesign-2026-11-14/D-layouts.md section 4): ?layout=<name> (a preset, or none);
// then the preset this browser chose in the "..." menu (localStorage d2layout); then the hero's stored copy (GET
// /api/layout says so in X-Layout-Source and it is never rewritten here); else the default: the cockpit (web/layout.json),
// for a new hero too (S7: nothing pins Simple any more; a browser an older shell pinned has no d2layoutpick, see chosen()),
// or Standard when a startup self-check finds a cockpit slot without its panel (start()). 'saved' in d2layout means "the stored copy"; d2layoutbase is the preset a
// saved layout was started from, which Reset returns to. Every read of localStorage is guarded: it may be blocked.
const PREF = 'd2layout', BASE = 'd2layoutbase', NOTE = 'd2layoutnote2', PICK = 'd2layoutpick', FIRST = 'd2firstrun';
const MENU = ['cockpit'].concat(PRESETS); // what the Layout menu offers; the cockpit is the default
const chosen = () => effectiveChoice(pref(PREF), pref(PICK)); // layout.js: an auto-pinned Simple is not a choice
function pref(k, v) {
  try {
    if (v === undefined) return localStorage.getItem(k) || '';
    if (v) localStorage.setItem(k, v); else localStorage.removeItem(k);
  } catch (e) { /* storage blocked */ }
  return '';
}
const asLayout = (l) => (l && l.schema === 'watch.layout/1' ? l : null); // another schema: this shell does not read it
async function getJSON(url) {
  try {
    const r = await fetch(url, { cache: 'no-store', credentials: 'same-origin' });
    return r.ok ? { body: await r.json(), source: r.headers.get('X-Layout-Source') || '' } : null;
  } catch (e) { return null; }
}
const presetFile = async (name) => asLayout((await getJSON('/watch/layouts/' + name + '.json') || {}).body);
async function newHero() {
  const o = await getJSON('/api/onboarding');
  // new = the welcome card is up and no checklist step has ticked; an old hero who never dismissed it is not new
  return !!(o && o.body && o.body.showWelcome === true && !(o.body.steps || []).some((x) => x.done));
}

// -> { layout, preset, origin: url|choice|stored|default, hasStored } or null (no layout: the manifest places)
const REMOVED = ['classic', 'simple', 'standard', 'debug', 'mockup', 'live', 'command']; // presets E11a deleted: they load the cockpit and say so
const MOVED = 'This panel moved. The new layout has it in the band, Progress or the log.';
async function loadLayout() {
  const sel = await loadLayout1();
  if (sel && !sel.moved && (REMOVED.includes(qs.get('layout') || '') || REMOVED.includes(pref(PREF)) || REMOVED.includes(pref(BASE)))) sel.moved = true;
  return sel;
}
async function loadLayout1() {
  const want = qs.get('layout') || '';
  if (want === 'none') return null;
  if (want === 'cockpit') { // "the cockpit, as I have it": a stored copy that started from it wins over the shipped file
    const srv = await getJSON('/api/layout');
    if (srv && srv.source !== 'default' && isCockpit(asLayout(srv.body))) return { layout: srv.body, preset: '', origin: 'stored', hasStored: true };
    const l = await presetFile('cockpit');
    if (l) return { layout: l, preset: '', origin: 'url', hasStored: !!srv && srv.source !== 'default' };
  }
  if (PRESETS.includes(want) || /^cockpit[a-z0-9-]*$/.test(want)) {
    const l = await presetFile(want);
    if (l) return { layout: l, preset: PRESETS.includes(want) ? want : '', origin: 'url', hasStored: false };
  }
  const srv = await getJSON('/api/layout');
  const hasStored = !!srv && srv.source !== 'default'; // a missing header (a proxy) reads as stored: never replace a stored layout by guess
  const choice = chosen(), base = pref(BASE);
  if (MENU.includes(choice)) {
    const l = await presetFile(choice);
    if (l) return { layout: l, preset: choice, origin: 'choice', hasStored };
  }
  if (hasStored && asLayout(srv.body)) return { layout: srv.body, preset: '', origin: 'stored', hasStored };
  // no layout of the hero's own: the default is the cockpit (web/layout.json). A new hero meets it without the layout note.
  let name = html.dataset.mode === 'overlay' ? 'stream' : 'cockpit', origin = 'default'; // origin 'default' shows the layout note; the OBS strip keeps the layout it always had
  if (choice === 'saved' && MENU.includes(base)) { name = base; origin = 'choice'; } else if (await newHero()) origin = 'newhero';
  let l = asLayout(srv && srv.body); // web/layout.json is the cockpit file
  if (name !== 'cockpit' || !l) l = await presetFile(name) || l;
  return l ? { layout: l, preset: name, origin, hasStored: false } : null;
}

function postJSON(answerBase, path, body) {
  return fetch(answerBase + path, {
    method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body || {}),
  }).then((r) => r.json()).catch(() => { toast('bad', 'the answer route is not reachable'); return null; });
}

function setupLayout(conf, sel, answerBase, ckRoots, ckList) {
  const layout = sel ? sel.layout : null;
  if (ckRoots) { setupCockpit(sel, answerBase, ckRoots, ckList); return; }
  lay.main = document.querySelector('.d-main');
  lay.conf = conf || {};
  lay.layout = layout;
  lay.sel = sel;
  if (!lay.main) return;
  if (layout && layout.density === 'comfortable' && !html.dataset.density) html.dataset.density = 'comfortable';
  applyLayout();
  // a panel that hides or opens (a 404 route, an unlock) changes which targets are packed
  let queued = false;
  const again = () => { queued = false; if (!lay.editor) applyLayout(); };
  new MutationObserver((ms) => {
    if (lay.editor) return;
    if (!queued && ms.some((m) => m.type === 'attributes' && m.target.matches('.d-panel[data-panel]'))) { queued = true; requestAnimationFrame(again); }
    else scheduleFit(); // a card redrew itself or folded something: its content height may have changed
  }).observe(lay.main, { subtree: true, childList: true, characterData: true, attributes: true, attributeFilter: ['hidden'] });
  if (typeof ResizeObserver === 'function') new ResizeObserver(scheduleFit).observe(lay.main);
  document.addEventListener('fullscreenchange', scheduleFit);
  window.addEventListener('resize', () => {
    if (breakpointFor(window.innerWidth, html.dataset.mode || '') === lay.bp) return;
    if (lay.editor) lay.editor.stop();
    applyLayout();
  });
  const btn = document.getElementById('btn-layout');
  if (!btn) return;
  btn.addEventListener('click', async () => {
    if (lay.editor) return;
    if (!lay.plan || lay.plan.flow) { toast('info', lay.layout ? 'Edit layout needs a window at least 1024 px wide.' : 'No layout is loaded.'); return; }
    let mod;
    try { mod = await import('/watch/layoutedit.js'); } catch (e) { toast('bad', 'The layout editor did not load.'); return; }
    const saved = lay.layout;
    lay.editor = { stop() {} }; // from here the plan carries the hidden panels along (flagged) and the editor owns the cells
    applyLayout();
    lay.editor = mod.start({
      el, toast, main: lay.main, targets: lay.targets, layout: saved, demo: qs.get('drag') || '',
      plan: () => lay.plan,
      preview(l) { lay.layout = l; applyLayout(); return lay.plan; },
      // the stored copy wins from now on: the preset it started from is remembered for Reset, and a stand-in's displaced entry is put back
      async save(l) {
        const r = await postJSON(answerBase, '/api/layout', { layout: withShadows(l, saved) });
        if (r && r.ok) {
          if (MENU.includes(lay.sel && lay.sel.preset)) pref(BASE, lay.sel.preset);
          pref(PREF, 'saved'); pref(PICK, '1');
        }
        return r;
      },
      // Reset removes the stored copy; the page then loads again, on the preset this browser chose (or the default)
      async reset() {
        const r = await postJSON(answerBase, '/api/layout', { reset: true });
        if (r && r.ok) {
          const b = pref(BASE);
          pref(PREF, MENU.includes(b) ? b : ''); pref(BASE, '');
          setTimeout(() => location.reload(), 700);
        }
        return r;
      },
      done(l) { lay.editor = null; lay.layout = l || saved; applyLayout(); },
    });
  });
}

// ---- the cockpit (layout template cockpit/1; layout.js cockpitPlan, shell.css) ----
// The panels the layout names exist before the first module loads (roots, a Map by id); this puts them into the slot
// elements by the plan of the present width and puts them again when the width crosses a mode (phone, page, wide).
// Slot elements and the plan are rebuilt only when the arrangement changes; a size change only sets two properties.
const ck = { layout: null, roots: new Map(), plan: null, sig: '', tab: '', want: '', second: false, meta: new Map(), tabBar: null, tabBody: null, tabs: [], main: null, shell: null, editor: null };

function ckSlot(name, cls) {
  const n = el('div', cls || '');
  n.dataset.cockpitSlot = name;
  return n;
}

// A phone tab shows one group of panels. The bar is built once per arrangement and kept, so a click moves the panels
// and the keyboard focus stays on the tab (the first version rebuilt the bar on every click and dropped the focus).
function showTab(name, focus) {
  if (!ck.tabBar || !ck.tabs.length) return;
  const tab = ck.tabs.find((x) => x.name === name) || ck.tabs[0];
  ck.tab = tab.name;
  for (const b of ck.tabBar.children) {
    const on = b.dataset.tab === tab.name;
    b.setAttribute('aria-selected', String(on));
    b.tabIndex = on ? 0 : -1;
    if (on && focus) b.focus({ preventScroll: true });
  }
  ck.tabBody.replaceChildren(...tab.ids.map((id) => ck.roots.get(id)));
  mapSwitch(ck.tabBody);
}

// Where a column (or a phone tab) holds two panels that share it (a manifest's switchWith: the act map and the area map), a
// two-button switch shows one at a time: the rail of a 1440 window has room for one. On a wide rail they stand side by side
// in two columns and no switch is built.
function mapSwitch(box) {
  for (const r of ck.roots.values()) r.querySelectorAll(':scope > .d-mapswitch').forEach((n) => n.remove());
  const kids = [...box.children].filter((n) => n.matches('.d-panel[data-panel]'));
  for (const n of kids) delete n.dataset.off;
  const links = (x, y) => (ck.meta.get(x.dataset.panel) || {}).switchWith === y.dataset.panel;
  let pair = null;
  for (const a of kids) {
    const b = kids.find((n) => n !== a && (links(a, n) || links(n, a)));
    if (b) { pair = kids.indexOf(a) < kids.indexOf(b) ? [a, b] : [b, a]; break; }
  }
  if (!pair) { growLast(box); ck.showMap = null; return; }
  // the switch sits on the top edge of whichever map shows, so it takes no height of its own (the zone rows need every pixel)
  const bar = el('div', 'd-mapswitch');
  bar.setAttribute('role', 'group');
  bar.setAttribute('aria-label', 'Which map');
  const paint = () => {
    pair[0].dataset.off = ck.second ? '1' : '';
    pair[1].dataset.off = ck.second ? '' : '1';
    pair.forEach((n) => { if (!n.dataset.off) delete n.dataset.off; });
    for (const b of bar.children) b.setAttribute('aria-pressed', String((b.dataset.view === '1') === ck.second));
    pair[ck.second ? 1 : 0].appendChild(bar);
    growLast(box);
  };
  pair.forEach((n, i) => {
    const m = ck.meta.get(n.dataset.panel) || {}, b = ui.btn(m.switchLabel || m.title || n.dataset.panel, { size: 'sm', variant: 'ghost' });
    b.dataset.view = String(i);
    if (m.title) { b.title = m.title; b.setAttribute('aria-label', m.title); }
    b.addEventListener('click', () => { ck.second = i === 1; paint(); const on = bar.children[i]; if (on) on.focus({ preventScroll: true }); });
    bar.appendChild(b);
  });
  ck.showMap = (id) => { const i = pair.findIndex((n) => n.dataset.panel === id); if (i >= 0 && ck.second !== (i === 1)) { ck.second = i === 1; paint(); } };
  paint();
}
// the Go to verb asks for the act map: its phone tab opens, and the switch shows it
document.addEventListener('d2:showmap', (e) => {
  const t = (ck.tabs || []).find((x) => x.ids.includes(e.detail));
  if (t && t.name !== ck.tab) showTab(t.name);
  if (ck.showMap) ck.showMap(e.detail);
});

function applyCockpit() {
  const p = cockpitPlan(ck.layout, window.innerWidth, html.dataset.mode || '', (id) => ck.roots.has(id), window.innerHeight);
  if (!p) return;
  ck.plan = p;
  const main = ck.main, shell = ck.shell;
  shell.dataset.cmode = main.dataset.cmode = p.mode;
  const editBtn = document.getElementById('btn-layout'); // a phone cannot edit the layout: no entry that only refuses
  if (editBtn) editBtn.hidden = p.mode === 'phone';
  if (p.mode === 'phone') shell.dataset.narrow = 'phone'; else if (window.innerWidth < 1000) shell.dataset.narrow = 'mid'; else delete shell.dataset.narrow; // the header links (components.css)
  main.dataset.has = p.has;
  main.style.setProperty('--c-strip', p.sizes.strip + 'px');
  main.style.setProperty('--c-railmin', p.sizes.railMin + 'px');
  if (ck.want) { // ?tab=Map opens that phone tab (the screenshot suite and the fit check look at each)
    const w = p.tabs.find((t) => t.name.toLowerCase() === ck.want.toLowerCase());
    if (w) { ck.tab = w.name; ck.want = ''; }
  }
  if (!p.tabs.some((t) => t.name === ck.tab)) ck.tab = p.tabs.length ? p.tabs[0].name : '';
  const sig = JSON.stringify([p.mode, p.band, p.stage, p.strip, p.top, p.rail, p.tabs]);
  if (sig === ck.sig) return;
  ck.sig = sig;
  main.querySelectorAll(':scope > [data-cockpit-slot]').forEach((n) => n.remove());
  const put = (box, ids) => ids.forEach((id) => box.appendChild(ck.roots.get(id)));
  const bandMain = shell.querySelector('.d-band-status .d-band-main');
  if (bandMain) {
    for (const r of ck.roots.values()) if (r.parentElement === bandMain) r.remove(); // a panel that left the band
    put(bandMain, p.band);
  }
  if (p.stage) { const s = ckSlot('stage'); put(s, [p.stage]); main.appendChild(s); }
  if (p.strip.length) { const s = ckSlot('strip'); put(s, p.strip); main.appendChild(s); }
  if (p.rail.length || p.top.length) {
    const r = ckSlot('rail');
    r.dataset.cols = String(p.rail.length);
    if (p.top.length) {
      const t = el('div', 'd-slotcol d-slottop');
      r.dataset.top = '1';
      if (p.rail.length) t.dataset.nogrow = '1'; // the columns under it take the rest
      put(t, p.top);
      growLast(t);
      r.appendChild(t);
    }
    for (const ids of p.rail) { const c = el('div', 'd-slotcol'); put(c, ids); mapSwitch(c); r.appendChild(c); }
    main.appendChild(r);
  }
  if (p.tabs.length) {
    const t = ckSlot('phoneTabs'), bar = el('div', 'd-tabbar'), body = el('div', 'd-slottabs');
    bar.setAttribute('role', 'tablist');
    body.setAttribute('role', 'tabpanel');
    ck.tabBar = bar; ck.tabBody = body; ck.tabs = p.tabs;
    for (const tab of p.tabs) {
      const b = ui.btn(tab.name, { size: 'sm', variant: 'ghost' });
      b.setAttribute('role', 'tab');
      b.dataset.tab = tab.name;
      b.addEventListener('click', () => showTab(tab.name));
      b.addEventListener('keydown', (e) => { // arrow keys move between the tabs; the bar is kept, so focus stays on it
        const i = ck.tabs.findIndex((x) => x.name === ck.tab), d = e.key === 'ArrowRight' ? 1 : e.key === 'ArrowLeft' ? -1 : 0;
        if (!d) return;
        e.preventDefault();
        showTab(ck.tabs[(i + d + ck.tabs.length) % ck.tabs.length].name, true);
      });
      bar.appendChild(b);
    }
    t.append(bar, body);
    main.appendChild(t);
    showTab(ck.tab);
  } else { ck.tabBar = ck.tabBody = null; }
  if (ck.editor && ck.editor.paint) ck.editor.paint();
}

function setupCockpit(sel, answerBase, roots, slotList) {
  ck.layout = sel.layout;
  ck.roots = roots;
  ck.want = qs.get('tab') || '';
  ck.main = document.querySelector('.d-main');
  ck.shell = document.querySelector('.d-shell');
  if (!ck.main || !ck.shell) return;
  ck.shell.dataset.template = ck.main.dataset.template = 'cockpit';
  const head = ck.shell.querySelector('.d-band-status');
  if (head) head.dataset.cockpitSlot = 'band';
  // the old page's message bar ("the viewer is not reachable", a refused answer) lives in the hidden card area: it floats over the frame instead
  const mb = document.getElementById('msgbar');
  if (mb) { mb.dataset.float = '1'; document.body.appendChild(mb); }
  applyCockpit();
  window.addEventListener('resize', () => applyCockpit());
  const btn = document.getElementById('btn-layout');
  if (!btn) return;
  btn.addEventListener('click', async () => {
    if (ck.editor) return;
    if (ck.plan.mode === 'phone') { toast('info', 'Edit layout needs a window at least 700 px wide.'); return; }
    let mod;
    try { mod = await import('/watch/layoutedit.js'); } catch (e) { toast('bad', 'The layout editor did not load.'); return; }
    const saved = ck.layout;
    ck.editor = mod.startCockpit({
      el, toast, main: ck.main, shell: ck.shell, roots: ck.roots, layout: saved, slotPanels: slotList,
      plan: () => ck.plan,
      preview(l) { ck.layout = l; applyCockpit(); return ck.plan; },
      async save(l) {
        const r = await postJSON(answerBase, '/api/layout', { layout: l });
        if (r && r.ok) { pref(PREF, 'saved'); pref(PICK, '1'); }
        return r;
      },
      async reset() {
        const r = await postJSON(answerBase, '/api/layout', { reset: true });
        if (r && r.ok) { pref(PREF, ''); pref(BASE, ''); setTimeout(() => location.reload(), 700); }
        return r;
      },
      done(l) { ck.editor = null; ck.layout = l || saved; applyCockpit(); },
    });
  });
  if (qs.get('edit') === '1') btn.click();
}

// ---- drawers (a cockpit has no tile for the log or the first-steps card; each opens as a side sheet) ----
// A drawer panel is mounted like any other (shell.json layout.drawer lists them) but its root lives in a sheet that is
// hidden until it is opened, and the scheduler polls it only while it is open. The "..." menu opens them, and so does
// ctx.openDrawer(id) (the status sentence's popover uses it).
const dr = { boxes: new Map(), open: '', opener: null };

function setupDrawers(roots, titles) {
  if (!roots.size) return;
  const back = el('div', 'd-drawer-back');
  back.hidden = true;
  back.addEventListener('click', () => closeDrawer(true));
  document.body.appendChild(back);
  dr.back = back;
  for (const [id, root] of roots) {
    const box = el('aside', 'd-drawer'), head = el('div', 'd-drawer-head'), body = el('div', 'd-drawer-body');
    const h = el('h2', '', titles.get(id) || id), x = ui.btn('Close', { size: 'sm', variant: 'ghost' });
    box.dataset.drawer = id;
    box.setAttribute('role', 'dialog');
    box.setAttribute('aria-label', titles.get(id) || id);
    box.hidden = true;
    x.dataset.key = 'drawer-close';
    x.addEventListener('click', () => closeDrawer(true));
    head.append(h, x);
    const gone = el('p', 'd-drawer-empty', 'Nothing here right now.'); // shown by the CSS while the panel hides itself
    body.append(root, gone);
    box.append(head, body);
    box.addEventListener('keydown', (e) => { if (e.key === 'Escape') { e.preventDefault(); closeDrawer(true); } });
    document.body.appendChild(box);
    overlays.register('drawer-' + id, box, () => { if (dr.open === id) closeDrawer(false); else box.hidden = true; });
    dr.boxes.set(id, { box, root, gone });
  }
}

function openDrawer(id, opener) {
  const d = dr.boxes.get(id);
  if (!d) return false;
  closeDrawer(false);
  dr.open = id;
  dr.opener = opener || document.activeElement;
  d.box.hidden = false;
  if (dr.back) dr.back.hidden = false;
  d.root.dataset.open = '1';
  if (theSched) theSched.poke(id);
  const x = d.box.querySelector('[data-key="drawer-close"]');
  if (x) x.focus({ preventScroll: true });
  return true;
}

function closeDrawer(back) {
  const d = dr.boxes.get(dr.open);
  if (!d) return;
  d.box.hidden = true;
  delete d.root.dataset.open;
  if (dr.back) dr.back.hidden = true;
  const o = dr.opener;
  dr.open = ''; dr.opener = null;
  if (back && o && o.focus && o.isConnected) o.focus({ preventScroll: true });
}

// The "Getting started 1 of 5" chip of the menu: one read of the onboarding reply whenever the menu opens (not a poll).
async function onboardingChip(btn) {
  try {
    const v = await (await fetch('/api/onboarding', { cache: 'no-store', credentials: 'same-origin' })).json();
    const total = (v.steps || []).length;
    if (v.allDone || !total) { btn.hidden = true; return; }
    btn.textContent = 'Getting started ' + (v.done || 0) + ' of ' + total;
    btn.hidden = false;
  } catch (e) { /* the chip keeps its last text */ }
}

// ---- the More sheet of the shared navigation (web/nav.js): this live screen adds its own groups to it ----
// nav.js draws the bottom bar, the band links and the sheet with the pages (Characters, Return screen, Hall of Fame, Policy). The
// buttons already on the page move into the sheet (they keep their ids and listeners); the preset buttons store the choice per
// browser and load the page again, because which panels are mounted (opt-in panels, the status band) follows the layout.
const PRESET_LABEL = { cockpit: 'Cockpit', stream: 'Stream' };
async function setupMenu(sel) {
  const nv = window.dNav;
  const host = document.querySelector('.d-band-status .d-links');
  if (!nv || !host) return;
  try { await Promise.race([nv.ready, new Promise((_, no) => setTimeout(() => no(new Error('no nav')), 3000))]); } catch (e) { return; }
  if (document.getElementById('btn-reset')) return;
  const more = nv.moreButton();
  const tools = [], hero = [], presets = []; // the sheet's groups: Hero, Tools and the layout presets
  for (const id of ['btn-layout', 'btn-help']) { const b = document.getElementById(id); if (b) tools.push(b); }
  nv.drop('help'); // this screen has the Help drawer and the Layout group itself
  const labelsItem = ui.btn('Item labels', { size: 'sm', variant: 'ghost' }); // the video tile's Labels button, so a tile in a drawer or a phone tab is still one tap away
  labelsItem.dataset.key = 'labels-menu';
  labelsItem.setAttribute('aria-label', 'Item labels: show or hide the names of items on the ground');
  labelsItem.hidden = true;
  labelsItem.addEventListener('click', () => { const b = document.querySelector('[data-key="labels"]'); if (b && !b.hidden) b.click(); });
  tools.push(labelsItem);
  let chip = null;
  for (const [id, text] of [['log', 'Log'], ['quest-board', 'Quests'], ['onboarding', 'Getting started']]) { // the drawers a cockpit has
    if (!dr.boxes.has(id)) continue;
    const b = ui.btn(text, { size: 'sm', variant: 'ghost' });
    b.dataset.drawerOpen = id;
    b.addEventListener('click', () => openDrawer(id, more));
    (id === 'log' ? tools : hero).push(b);
    if (id === 'onboarding') { chip = b; b.hidden = true; }
  }
  if (chip) onboardingChip(chip);
  // the Return screen link of the sheet shows only where the bot mounts the return screen (probeReturn sets data-return-screen on the page)
  const syncReturn = () => nv.hide('return', html.dataset.returnScreen !== '1');
  syncReturn();
  new MutationObserver(syncReturn).observe(html, { attributes: true, attributeFilter: ['data-return-screen'] });
  document.addEventListener('d2:nav', (e) => { // Help opens its drawer in place
    if (e.detail && e.detail.id === 'help') { e.preventDefault(); openDrawer('onboarding', more); window.dispatchEvent(new Event('d2help')); }
  });
  const rb = ui.btn('Reset', { size: 'sm', variant: 'ghost' });
  rb.dataset.key = 'reset-open';
  rb.setAttribute('aria-label', 'Reset bot (leaves the game and rejoins)');
  rb.addEventListener('click', () => document.dispatchEvent(new CustomEvent('d2:reset-ask', { detail: { from: more } })));
  const sb = ui.btn('Send support bundle...', { size: 'sm', variant: 'ghost' });
  sb.dataset.key = 'support-open';
  sb.setAttribute('aria-label', 'Help: send a support bundle to the owner');
  sb.addEventListener('click', () => document.dispatchEvent(new CustomEvent('d2:support-ask', { detail: { from: more } })));
  tools.push(sb);
  const lb = ui.btn('Loot rules...', { size: 'sm', variant: 'ghost' });
  lb.dataset.key = 'lootrules-open';
  lb.setAttribute('aria-label', 'Loot rules: what the bot picks up, stashes and sells');
  lb.addEventListener('click', () => document.dispatchEvent(new CustomEvent('d2:lootrules-ask', { detail: { from: more } })));
  hero.push(lb);
  const cur = sel ? sel.preset : '';
  const names = (sel && sel.hasStored ? ['saved'] : []).concat(MENU);
  for (const name of names) {
    const b = ui.btn(name === 'saved' ? 'Saved layout' : 'Layout: ' + PRESET_LABEL[name], { size: 'sm', variant: 'ghost' });
    b.dataset.preset = name;
    b.setAttribute('aria-checked', String(name === 'saved' ? !!sel && sel.origin === 'stored' : name === cur));
    b.setAttribute('role', 'menuitemradio');
    b.addEventListener('click', () => choosePreset(name));
    presets.push(b);
  }
  nv.primary.appendChild(rb); // the primary row (Reset, then Pause from pauseui.js) never scrolls out of reach
  nv.group('hero', 'Hero', hero, true);
  nv.group('tools', 'Tools', tools, true);
  nv.group('layout', 'Layout', presets, false);
  overlays.register('menu', nv.sheet, () => nv.close(false));
  document.addEventListener('d2:nav-sheet', () => { // opening the sheet: the chips that read the bot
    if (chip) onboardingChip(chip);
    const b = document.querySelector('[data-key="labels"]');
    labelsItem.hidden = !b || b.hidden;
    if (b) labelsItem.setAttribute('aria-pressed', b.getAttribute('aria-pressed') || 'false');
  });
  // the break-glass Reset: always visible in the band, one tap, no confirm (resetui.js wires it)
  const rbtn = ui.btn('Reset', { size: 'sm', variant: 'ghost' });
  rbtn.id = 'btn-reset';
  rbtn.classList.add('d-reset-btn');
  rbtn.dataset.key = 'reset-header';
  host.append(rbtn);
}

function choosePreset(name) {
  pref(PREF, name); pref(PICK, '1');
  if (qs.has('layout')) { const u = new URL(location.href); u.searchParams.delete('layout'); history.replaceState(null, '', u); }
  location.reload();
}

// One line for a hero who now gets the cockpit by default, dismissed for good in this browser. A cockpit has no row for it
// (the frame fills the window), so there it floats over the page like a toast; the Stream page keeps it under the band.
function layoutNote(sel, floating) {
  if (!sel || sel.origin !== 'default' || html.dataset.mode === 'overlay' || pref(NOTE)) return;
  const shell = document.querySelector('.d-shell'), top = document.querySelector('.d-band-status');
  if (!shell || !top) return;
  const box = el('div', 'd-layoutnote');
  box.setAttribute('role', 'note');
  if (floating) box.dataset.float = '1';
  box.appendChild(el('span', '', sel.fell ? 'The new layout could not load, so this is the Stream layout.' : 'The layout changed.'));
  const x = ui.btn('Dismiss', { size: 'sm', variant: 'ghost' });
  x.addEventListener('click', () => { pref(NOTE, '1'); box.remove(); });
  box.appendChild(x);
  if (floating) document.body.appendChild(box); else top.after(box);
}

// The Return screen link shows only where the bot mounts the route behind it: GET /api/tally answers there and 404s elsewhere.
function probeReturn() {
  fetch('/api/tally', { cache: 'no-store', credentials: 'same-origin' })
    .then((r) => { if (r.ok) html.dataset.returnScreen = '1'; else delete html.dataset.returnScreen; })
    .catch(() => { delete html.dataset.returnScreen; });
}

// A browser that still asks for a removed preset (?layout=classic, a saved choice) gets the cockpit and this one line, dismissed once.
function movedNote(sel) {
  if (!sel || !sel.moved) return;
  const box = el('div', 'd-layoutnote');
  box.setAttribute('role', 'note');
  box.dataset.float = '1';
  box.dataset.key = 'moved-note';
  box.appendChild(el('span', '', MOVED));
  const x = ui.btn('Dismiss', { size: 'sm', variant: 'ghost' });
  x.addEventListener('click', () => { pref(PREF, ''); pref(BASE, ''); box.remove(); });
  box.appendChild(x);
  document.body.appendChild(box);
}

// A slot panel of the cockpit that fails to load leaves a hole: say so once, with the way back one click away.
let failedShown = false;
function failedNote() {
  if (failedShown) return;
  failedShown = true;
  const box = el('div', 'd-layoutnote');
  box.setAttribute('role', 'alert');
  box.dataset.float = '1';
  box.appendChild(el('span', '', 'Part of this layout did not load.'));
  const go = ui.btn('Reload', { size: 'sm', variant: 'primary' });
  go.addEventListener('click', () => location.reload());
  const x = ui.btn('Dismiss', { size: 'sm', variant: 'ghost' });
  x.addEventListener('click', () => box.remove());
  box.append(go, x);
  document.body.appendChild(box);
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
    tileSheet(where) { return tileSheet(m.id, parts.root, where); },
    hide(on) { parts.root.hidden = !!on; syncBand(parts.root.parentElement); },
    state(st) { parts.root.dataset.state = st; },
    later(fn, ms) { const t = setTimeout(() => { timers.delete(t); fn(); }, ms); timers.add(t); return t; },
    // the d2dview cookie authenticates same-origin reads; the token is never copied into a fetch address
    // one read of a viewer route outside the manifest poll (a panel's own state, e.g. the video card's Labels button); null when it fails
    get: (path) => fetch(path, { cache: 'no-store', credentials: 'same-origin' }).then((r) => r.json()).catch(() => null),
    post: (path, body) => postJSON(answerBase, path, body),
    askReset(from) { document.dispatchEvent(new CustomEvent('d2:reset-ask', { detail: { from } })); }, // the Reset confirm sheet (resetui.js)
    refresh() { if (theSched) theSched.poke(m.id); }, // poll this panel now (a drawer's tab changed)
    openDrawer(id) { return openDrawer(id); },
    short() { return !!ck.plan && ck.plan.mode !== 'phone' && window.innerHeight < COCKPIT_SHORT; }, // a short cockpit window (the panels trim their lists)
  };
}

// The default flips to the cockpit only while every slot has its panel: when the manifest lacks one (a half-updated install) the
// page falls back to Standard instead of showing a hole. A layout the hero picked or stored is never second-guessed.
async function selfCheck(sel, list, conf0) {
  if (!sel || sel.origin !== 'default' && sel.origin !== 'newhero' || !isCockpit(sel.layout) || sel.preset !== 'cockpit') return sel;
  const missing = missingSlots(sel.layout, list.filter((m) => m.slot || (conf0.slot || []).includes(m.id)).map((m) => m.id));
  if (!missing.length) return sel;
  console.warn('d2lod: cockpit slot panel missing (' + missing.join(', ') + '), using Stream');
  const l = await presetFile('stream');
  return l ? { layout: l, preset: 'stream', origin: sel.origin, hasStored: false, fell: true } : sel;
}

async function start() {
  setupTheme();
  const bands = {};
  document.querySelectorAll('[data-band]').forEach((b) => { bands[b.dataset.band] = b; });
  let cfg = {};
  try { cfg = await (await fetch('/api/config', { cache: 'no-store' })).json(); } catch (e) { /* the old code says so */ }
  const answerBase = cfg.answerBase || (cfg.answerPort ? location.protocol + '//' + location.hostname + ':' + cfg.answerPort : '');
  let list = [], conf = {};
  const layoutReq = document.querySelector('[data-band="grid"]') ? loadLayout() : null; // the other pages share this shell without a layout
  try {
    const pv = await (await fetch('/watch/panels.json', { cache: 'no-store' })).json();
    list = (pv.panels || []).filter((m) => !m.mergedInto); // a merged panel's id is an alias in the layout (shell.json)
    conf = pv.layout || {};
  } catch (e) { return; }
  // the layout decides which panels mount (opt-in panels, the status band's list), so it is read before any root exists
  let sel = layoutReq ? await layoutReq : null;
  sel = await selfCheck(sel, list, conf);
  ck.meta = new Map(list.map((m) => [m.id, m])); // the manifests (a cockpit's switch pairs read switchWith)
  const cockpit = !!sel && isCockpit(sel.layout) && html.dataset.mode !== 'overlay';
  // panels the layout may put in a slot: their manifest says so, or shell.json lists them until it does
  for (const m of list) if (m.slot || (conf.slot || []).includes(m.id)) m.slot = true;
  if (cockpit) for (const m of list) if ((conf.drawer || []).includes(m.id)) m.drawer = true; // a grid layout keeps the log a plain tile
  const slotList = list.filter((m) => m.slot).map((m) => ({ id: m.id, title: m.title }));
  if (layoutReq) list = selectPanels(list, sel && sel.layout, html.dataset.mode);
  const sched = theSched = makeScheduler();
  window.addEventListener('pageshow', (e) => { if (e.persisted) sched.rewake(); });
  // the Help button opens the onboarding panel's Help entries (panels/onboarding), which listens for this event
  const helpBtn = document.getElementById('btn-help');
  // in the cockpit the panel lives in a drawer: open it; the panel opens its Help on every d2help while it sits in a drawer, so a
  // second event after its first poll (it may not have mounted yet) does no harm
  if (helpBtn) helpBtn.addEventListener('click', () => { openDrawer('onboarding', helpBtn); window.dispatchEvent(new Event('d2help')); setTimeout(() => window.dispatchEvent(new Event('d2help')), 900); });
  // every root exists before the first module loads, so the layout places the whole page at once
  const made = [], ckRoots = new Map(), drRoots = new Map(), drTitles = new Map();
  for (const m of list) {
    const parts = makeRoot(m, cockpit);
    if (cockpit && m.drawer) { drRoots.set(m.id, parts.root); drTitles.set(m.id, m.title); made.push([m, parts]); }
    else if (cockpit) { ckRoots.set(m.id, parts.root); made.push([m, parts]); } else if (place(bands, m, parts.root)) made.push([m, parts]);
  }
  setupLayout(conf, sel, answerBase, cockpit ? ckRoots : null, slotList);
  if (cockpit) setupDrawers(drRoots, drTitles);
  if (layoutReq) { probeReturn(); await setupMenu(sel); layoutNote(sel, cockpit); movedNote(sel); }
  // a new hero meets the first-steps card once, without opening a menu
  if (cockpit && sel.origin === 'newhero' && !pref(FIRST) && dr.boxes.has('onboarding') && !qs.has('drawer') && qs.get('first') !== '0') { pref(FIRST, '1'); openDrawer('onboarding'); }
  if (cockpit && qs.get('drawer')) openDrawer(qs.get('drawer')); // a link (and the screenshot suite) can open a drawer
  // The poll loop runs before the first module loads: the imports below are plain GETs queued behind the picture and event
  // streams (a browser keeps about six connections per host, tabs together), and a slow one used to hold the loop back, so
  // every panel stayed in its first state ("waiting for the bot"). A slot polls as soon as its own panel is added.
  sched.start();
  for (const [m, parts] of made) {
    const css = el('link');
    css.rel = 'stylesheet'; css.href = '/watch/panels/' + m.id + '/panel.css';
    document.head.appendChild(css);
    const timers = new Set();
    let api = {};
    try {
      const mod = await Promise.race([import('/watch/panels/' + m.id + '/panel.js'), new Promise((_, no) => setTimeout(() => no(new Error('panel module timed out')), FETCH_TIMEOUT * 2))]);
      api = mod.mount(parts.root, makeCtx(m, parts, cfg, answerBase, timers)) || {};
    } catch (e) {
      parts.root.dataset.state = 'error';
      parts.body.appendChild(el('p', 'd-empty', 'This panel failed to load.'));
      if (cockpit && sel.origin !== 'url') failedNote();
      continue;
    }
    if (cockpit) { // a slot has no title bar to fold, and a fold saved by an old page must not blank the box
      parts.actions.querySelectorAll('[data-key="collapse"]').forEach((n) => n.remove());
      parts.body.hidden = false;
    }
    if (!m.endpoint && parts.root.dataset.state === 'loading') parts.root.dataset.state = 'ready';
    sched.add({ m, root: parts.root, api });
    syncBand(parts.root.parentElement);
  }
  Object.values(bands).forEach(syncBand);
  mountReset({ ui, toast, doc: document, button: document.getElementById('btn-reset'), later: (fn, ms) => setTimeout(fn, ms),
    post: (path, body) => postJSON(answerBase, path, body),
    get: (path) => fetch(path, { cache: 'no-store' }).then((r) => (r.ok ? r.json() : null)).catch(() => null) });
  mountPause({ ui, toast, doc: document, later: (fn, ms) => setTimeout(fn, ms),
    post: (path, body) => postJSON(answerBase, path, body),
    get: (path) => fetch(path, { cache: 'no-store' }).then((r) => (r.ok ? r.json() : null)).catch(() => null) });
  const rc = document.getElementById('reset-confirm');
  if (rc) overlays.register('reset', rc); // the modal confirm: opening it closes the menu it came from
  mountLootRules({ ui, toast, doc: document, overlays,
    get: (path) => fetch(path, { cache: 'no-store' }).then((r) => (r.ok ? r.json() : null)).catch(() => null),
    put: (path, body) => fetch(answerBase + path, { method: 'PUT', credentials: 'include', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body || {}) })
      .then((r) => r.json().then((j) => ({ status: r.status, j }))).catch(() => { toast('bad', 'the answer route is not reachable'); return null; }),
    post: (path, body) => fetch(answerBase + path, { method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body || {}) })
      .then((r) => r.json().then((j) => ({ status: r.status, j }))).catch(() => { toast('bad', 'the answer route is not reachable'); return null; }) });
  mountSupport({ ui, toast, doc: document, post: (path, body) => postJSON(answerBase, path, body),
    get: (path) => fetch(path, { cache: 'no-store' }).then((r) => (r.ok ? r.json() : null)).catch(() => null) });
  if (qs.get('lootrules') === '1') document.dispatchEvent(new CustomEvent('d2:lootrules-ask', { detail: {} })); // Settings > Rules links here from every page
  // a link (and the screenshot suite) can open the page in Edit layout, once every panel is mounted
  const edit = document.getElementById('btn-layout');
  if (edit && !cockpit && qs.get('edit') === '1') edit.click();
}

start();
