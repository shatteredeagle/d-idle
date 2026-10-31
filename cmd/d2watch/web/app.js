'use strict';
// View-only: only GETs images and JSON, never sends input to a game.
const grid = document.getElementById('grid');
const W = 800, H = 587; // game client area as captured (UI coordinates of the mouse are in the same pixels)
const tiles = new Map();
let paused = false, big = null, data = [];
const LS = { get(k) { try { return localStorage.getItem(k); } catch (e) { return null; } }, set(k, v) { try { localStorage.setItem(k, v); } catch (e) {} } };

// theme
const root = document.documentElement;
function applyTheme(t) { if (t) root.dataset.theme = t; else delete root.dataset.theme; }
applyTheme(LS.get('theme'));
document.getElementById('theme').onclick = () => {
  const dark = root.dataset.theme ? root.dataset.theme === 'dark' : matchMedia('(prefers-color-scheme:dark)').matches;
  const t = dark ? 'light' : 'dark'; applyTheme(t); LS.set('theme', t);
};
const pauseBtn = document.getElementById('pause');
pauseBtn.onclick = () => {
  paused = !paused; pauseBtn.setAttribute('aria-pressed', paused); pauseBtn.textContent = paused ? 'Resume' : 'Pause';
  for (const t of tiles.values()) syncStream(t);
};
addEventListener('keydown', e => { if (e.key === 'Escape' && big) setBig(null); });

function mk(tag, cls, text) { const e = document.createElement(tag); if (cls) e.className = cls; if (text != null) e.textContent = text; return e; }

function makeTile(id) {
  const el = mk('div', 'tile');
  const img = mk('img'); img.alt = id; img.draggable = false;
  const cv = mk('canvas');
  const label = mk('div', 'label');
  const note = mk('div', 'note'); note.hidden = true;
  const bars = mk('div', 'bars'); const l = mk('span'), m = mk('span'); const li = mk('i', 'l'), mi = mk('i', 'm'); l.append(li); m.append(mi); bars.append(l, m);
  el.append(img, cv, label, note, bars);
  const t = { id, el, img, cv, label, note, li, mi, streaming: false, d: null, prev: null, last: null, retry: 0, fails: 0, since: 0 };
  // A multipart stream that ends (server dropped a stalled client, restart) fires onload, not onerror, and the
  // img then freezes on its last frame: treat both as "stream gone" and reconnect with backoff.
  img.onerror = img.onload = () => streamLost(t);
  el.onclick = () => setBig(big === t ? null : t);
  grid.append(el);
  return t;
}

function wantStream(t) {
  if (paused || document.hidden || !t.d || !t.d.window) return false;
  return big ? big === t : true;
}
function syncStream(t) {
  const want = wantStream(t);
  if (want && !t.streaming && !t.retry) { t.img.src = (t.url || '/stream/' + t.id + '.mjpg') + '?' + Date.now(); t.streaming = true; t.since = performance.now(); }
  else if (!want && t.streaming) { t.img.removeAttribute('src'); t.streaming = false; }
}
function streamLost(t) {
  if (!t.streaming || !t.img.getAttribute('src')) return;
  t.streaming = false;
  // a stream that lived a while resets the backoff; quick failures back off 1s .. 30s
  t.fails = performance.now() - t.since > 10000 ? 0 : t.fails + 1;
  const wait = Math.min(30000, 1000 * 2 ** Math.min(t.fails, 5));
  clearTimeout(t.retry);
  t.retry = setTimeout(() => { t.retry = 0; syncStream(t); }, wait);
}
// A hidden tab is throttled by the browser and stops draining the stream; drop it and reconnect on return.
document.addEventListener('visibilitychange', () => {
  for (const t of tiles.values()) {
    if (document.hidden) { clearTimeout(t.retry); t.retry = 0; if (t.streaming) { t.img.removeAttribute('src'); t.streaming = false; } }
    else { t.fails = 0; syncStream(t); }
  }
});

function setBig(t) {
  big = t;
  for (const x of tiles.values()) { x.el.classList.toggle('big', x === t); syncStream(x); }
  layout();
}

// pick the column count that gives the largest 4:3-ish tiles
function layout() {
  const list = [...tiles.values()];
  const n = list.length;
  const aw = grid.clientWidth - 24, ah = grid.clientHeight - 24, gap = 12, ar = W / H;
  let best = 0, bw = 200;
  for (let c = 1; c <= Math.max(n, 1); c++) {
    const r = Math.ceil(n / c);
    const w = Math.min((aw - gap * (c - 1)) / c, ((ah - gap * (r - 1)) / r) * ar);
    if (w > best) { best = w; bw = w; }
  }
  bw = Math.max(160, Math.floor(bw));
  const bigW = Math.floor(Math.min(innerWidth * .94, innerHeight * .94 * ar));
  for (const t of list) {
    const w = t === big ? bigW : bw;
    t.el.style.width = w + 'px'; t.el.style.height = Math.round(w / ar) + 'px';
    t.el.style.visibility = big && t !== big ? 'hidden' : '';
  }
  for (const t of list) draw(t);
}
addEventListener('resize', layout);

function label(t) {
  const d = t.d;
  t.label.textContent = '';
  const parts = [mk('span', 'ver d2lod', 'd2lod'), mk('b', '', d.id), mk('span', '', d.agent || '')];
  parts[0].title = 'd2lod: the original Game.exe backend';
  const who = [d.character, d.level ? 'L' + d.level : '', d.class].filter(Boolean).join(' ');
  if (who) parts.push(mk('span', '', who));
  if (d.areaName) parts.push(mk('span', 'dim', d.areaName));
  t.label.append(...parts);
  const chip = mk('span', 'chip', d.state === 'menu' ? 'menu' : d.state === 'ingame' ? 'in game' : d.mem === 'ok' ? (d.state || '') : 'capture only');
  t.label.append(chip);
}

function update(list) {
  data = list;
  const seen = new Set();
  for (const d of list) {
    seen.add(d.id);
    let t = tiles.get(d.id);
    if (!t) { t = makeTile(d.id); tiles.set(d.id, t); layout(); }
    t.prev = t.d; t.d = d;
    t.at = performance.now();
    label(t);
    t.note.hidden = d.window;
    t.note.textContent = d.window ? '' : 'waiting for the game window';
    t.li.style.width = d.maxLife ? Math.max(0, Math.min(100, d.life * 100 / d.maxLife)) + '%' : '0';
    t.mi.style.width = d.maxMana ? Math.max(0, Math.min(100, d.mana * 100 / d.maxMana)) + '%' : '0';
    syncStream(t);
    // activity hints
    if (t.prev) {
      if (d.mouse && t.prev.mouse && Math.hypot(d.mouse.x - t.prev.mouse.x, d.mouse.y - t.prev.mouse.y) > 220) flash(t, 'flash-mouse');
      if (d.maxLife && t.prev.maxLife && d.life < t.prev.life && (t.prev.life - d.life) >= Math.max(3, d.maxLife * 0.05)) flash(t, 'flash-life');
    }
  }
  for (const [id, t] of tiles) if (!seen.has(id) && id !== D2D) { if (big === t) big = null; t.el.remove(); tiles.delete(id); layout(); }
  document.getElementById('empty').hidden = tiles.size > 0;
  const up = list.filter(d => d.state === 'ingame').length;
  document.getElementById('summary').textContent = list.length + ' running, ' + up + ' in game';
}
function flash(t, cls) { t.el.classList.remove('flash-mouse', 'flash-life'); void t.el.offsetWidth; t.el.classList.add(cls); }

// cursor dot + fading trail; the trail points carry their age at poll time, advanced here by local elapsed time
function draw(t) {
  const cv = t.cv, d = t.d;
  const w = cv.clientWidth, h = cv.clientHeight;
  if (cv.width !== w || cv.height !== h) { cv.width = w; cv.height = h; }
  const g = cv.getContext('2d'); g.clearRect(0, 0, w, h);
  if (!d || !d.mouse) return;
  const sx = w / W, sy = h / H, el = performance.now() - t.at;
  const pts = (d.trail || []).map(p => ({ x: p.x * sx, y: p.y * sy, a: p.age + el })).filter(p => p.a < 4000);
  g.lineCap = g.lineJoin = 'round';
  for (let i = 1; i < pts.length; i++) {
    const a = pts[i], b = pts[i - 1];
    const f = Math.max(0, 1 - a.a / 4000);
    g.strokeStyle = 'rgba(255,205,60,' + (0.8 * f).toFixed(3) + ')'; g.lineWidth = 1 + 2 * f;
    g.beginPath(); g.moveTo(b.x, b.y); g.lineTo(a.x, a.y); g.stroke();
  }
  const m = d.mouse, mx = m.x * sx, my = m.y * sy;
  g.fillStyle = '#ff4040'; g.strokeStyle = '#fff'; g.lineWidth = 1.5;
  g.beginPath(); g.arc(mx, my, 5, 0, 7); g.fill(); g.stroke();
}
function frameLoop() { for (const t of tiles.values()) if (t.d && t.d.trail && t.d.trail.length) draw(t); requestAnimationFrame(frameLoop); }
requestAnimationFrame(frameLoop);

async function poll() {
  if (!paused && !document.hidden) {
    try {
      const r = await fetch('api/instances', { cache: 'no-store' });
      if (r.ok) update((await r.json()).instances || []);
    } catch (e) {}
  }
  setTimeout(poll, 250);
}
poll();

// d2d tile (agent d2dview2): the sped-up native d2d bot, served by cmd/d2dview on :8891 (docs/d2d-view.md). Not a
// capture of ours: <img> of its MJPEG stream, so d2watch spends no CPU on it. Shown only while d2dview answers (probed
// every 5 s with an opaque no-cors GET of a cheap endpoint); the stream opens only while the tile is wanted (not paused,
// tab visible, not hidden behind another big tile), and d2dview samples frames only while a client reads.
const D2D = '__d2d';
const d2dBase = location.protocol + '//' + location.hostname + ':8891';
async function probeD2D() {
  let up = false;
  if (!document.hidden) {
    try { await fetch(d2dBase + '/api/state', { mode: 'no-cors', cache: 'no-store', signal: AbortSignal.timeout(3000) }); up = true; } catch (e) {}
  } else { up = tiles.has(D2D); }
  let t = tiles.get(D2D);
  if (up && !t) {
    t = makeTile(D2D);
    t.url = d2dBase + '/stream.mjpg';
    t.d = { id: 'd2d', window: true };
    t.label.append(Object.assign(mk('span', 'ver d2d', 'd2d'), { title: 'd2d: the reimplementation backend' }), mk('b', '', 'd2d bot'), mk('span', 'dim', 'sampled frames, sped up (full page with map: :8891)'));
    t.el.title = 'Sampled frames of the sped-up d2d bot (about 1 frame per 100+ virtual ticks in fast mode). Open ' + d2dBase + '/ for the map.';
    t.el.onclick = e => { if (e.shiftKey) open(d2dBase + '/'); else setBig(big === t ? null : t); };
    tiles.set(D2D, t); layout(); syncStream(t);
  } else if (!up && t && !document.hidden) {
    if (big === t) big = null;
    t.img.removeAttribute('src'); t.el.remove(); tiles.delete(D2D); layout();
  }
  setTimeout(probeD2D, 5000);
}
probeD2D();
