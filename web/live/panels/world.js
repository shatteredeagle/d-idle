import { h, clear } from '../lib.js';

const css = (el, v) => getComputedStyle(el).getPropertyValue(v).trim();

function build(el, ctx) {
  const canvas = h('canvas', { class: 'wcanvas', tabindex: 0, role: 'img', 'aria-label': 'Minimap of the current area. Arrow keys pan, plus and minus zoom, 0 fits the whole map, F follows the player.' });
  const info = h('div', { class: 'muted', style: { fontSize: '12px', minHeight: '18px' }, 'aria-live': 'off' });
  const st = { canvas, info, zoom: 3, cx: 0, cy: 0, follow: false, fitted: false, rev: -1, fetching: false, terrain: null, off: null, offKey: '', d: null, drag: null, mouse: null };
  const btn = (text, label, fn, pressed) => h('button', { type: 'button', 'aria-label': label, 'aria-pressed': pressed ? 'false' : null, onclick: fn }, text);
  const zoomBy = (f, ax, ay) => {
    const r = canvas.getBoundingClientRect();
    ax = ax ?? r.width / 2; ay = ay ?? r.height / 2;
    const wx = st.cx + (ax - r.width / 2) / st.zoom, wy = st.cy + (ay - r.height / 2) / st.zoom;
    st.zoom = Math.max(0.3, Math.min(40, st.zoom * f));
    st.cx = wx - (ax - r.width / 2) / st.zoom; st.cy = wy - (ay - r.height / 2) / st.zoom;
    draw(st);
  };
  const fit = () => {
    const t = st.terrain; if (!t) return;
    const r = canvas.getBoundingClientRect();
    st.zoom = Math.max(0.3, Math.min(r.width / t.w, r.height / t.h) * 0.95);
    st.cx = t.x0 + t.w / 2; st.cy = t.y0 + t.h / 2; st.follow = false; followBtn.setAttribute('aria-pressed', 'false');
    draw(st);
  };
  const setFollow = (on) => { st.follow = on; followBtn.setAttribute('aria-pressed', String(on)); if (on) { if (st.zoom < 4) st.zoom = 6; centerPlayer(st); } draw(st); };
  const followBtn = btn('Follow player', 'Follow player', () => setFollow(!st.follow), true);
  const tools = h('div', { class: 'wtools' }, btn('+', 'Zoom in', () => zoomBy(1.4)), btn('−', 'Zoom out', () => zoomBy(1 / 1.4)), btn('Fit', 'Fit whole map', fit), followBtn, h('span', { class: 'muted', id: 'wname' }));
  st.fit = fit; st.tools = tools;

  canvas.addEventListener('wheel', (e) => { e.preventDefault(); const r = canvas.getBoundingClientRect(); zoomBy(e.deltaY < 0 ? 1.15 : 1 / 1.15, e.clientX - r.left, e.clientY - r.top); }, { passive: false });
  canvas.addEventListener('pointerdown', (e) => { canvas.setPointerCapture(e.pointerId); st.drag = { x: e.clientX, y: e.clientY }; });
  canvas.addEventListener('pointerup', () => { st.drag = null; });
  canvas.addEventListener('pointermove', (e) => {
    const r = canvas.getBoundingClientRect();
    st.mouse = { x: e.clientX - r.left, y: e.clientY - r.top };
    if (st.drag) {
      st.cx -= (e.clientX - st.drag.x) / st.zoom; st.cy -= (e.clientY - st.drag.y) / st.zoom;
      st.drag = { x: e.clientX, y: e.clientY };
      if (st.follow) { st.follow = false; followBtn.setAttribute('aria-pressed', 'false'); }
      draw(st);
    }
    hover(st);
  });
  canvas.addEventListener('pointerleave', () => { st.mouse = null; info.textContent = ''; });
  canvas.addEventListener('keydown', (e) => {
    const s = 30 / st.zoom;
    const k = e.key;
    if (k === 'ArrowLeft') st.cx -= s; else if (k === 'ArrowRight') st.cx += s; else if (k === 'ArrowUp') st.cy -= s; else if (k === 'ArrowDown') st.cy += s;
    else if (k === '+' || k === '=') { zoomBy(1.4); e.preventDefault(); return; } else if (k === '-') { zoomBy(1 / 1.4); e.preventDefault(); return; }
    else if (k === '0') { fit(); return; } else if (k === 'f' || k === 'F') { setFollow(!st.follow); return; } else return;
    e.preventDefault(); st.follow = false; followBtn.setAttribute('aria-pressed', 'false'); draw(st);
  });
  new ResizeObserver(() => draw(st)).observe(canvas);

  const sh = (path, fill) => `<svg viewBox="0 0 12 12" aria-hidden="true"><path d="${path}" fill="${fill}" stroke="currentColor" stroke-width="1"/></svg>`;
  const legend = h('div', { class: 'legend' });
  legend.innerHTML = [
    [sh('M6 1 L11 6 L6 11 L1 6Z', 'none'), 'player'], [sh('M6 1 A5 5 0 1 1 5.9 1Z', 'currentColor'), 'monster'], [sh('M2 2 L10 10 M10 2 L2 10', 'none'), 'dead'],
    [sh('M6 1 L11 6 L6 11 L1 6Z', 'currentColor'), 'npc'], [sh('M2 2H10V10H2Z', 'currentColor'), 'item'], [sh('M6 1 L11 11 L1 11Z', 'currentColor'), 'exit / waypoint'],
  ].map(([s, t]) => `<span>${s}${t}</span>`).join('');
  st.legendEl = legend;
  el.append(tools, canvas, info, legend);
  return st;
}

function centerPlayer(st) { if (st.d) { st.cx = st.d.px; st.cy = st.d.py; } }

async function loadTerrain(st, d) {
  if (st.fetching) return;
  st.fetching = true;
  try {
    const t = await (await fetch(d.terrainPath + '?rev=' + d.rev)).json();
    st.terrain = t; st.rev = t.rev; st.offKey = '';
    if (!st.fitted && t.w > 0) { st.fitted = true; st.fit(); }
  } catch (e) { console.error('terrain', e); }
  st.fetching = false;
  draw(st);
}

function offscreen(st, floor) {
  const key = st.rev + '|' + floor;
  if (st.off && st.offKey === key) return st.off;
  const t = st.terrain;
  const c = document.createElement('canvas'); c.width = Math.max(1, t.w); c.height = Math.max(1, t.h);
  const g = c.getContext('2d'); g.fillStyle = floor;
  t.rows.forEach((runs, y) => { let x = 0, walk = false; for (const n of runs) { if (walk) g.fillRect(x, y, n, 1); x += n; walk = !walk; } });
  st.off = c; st.offKey = key;
  return c;
}

function toScreen(st, r, x, y) { return [(x - st.cx) * st.zoom + r.width / 2, (y - st.cy) * st.zoom + r.height / 2]; }

function draw(st) {
  const cv = st.canvas, r = cv.getBoundingClientRect();
  if (r.width < 2) return;
  const dpr = window.devicePixelRatio || 1;
  if (cv.width !== Math.round(r.width * dpr) || cv.height !== Math.round(r.height * dpr)) { cv.width = Math.round(r.width * dpr); cv.height = Math.round(r.height * dpr); }
  const g = cv.getContext('2d');
  g.setTransform(dpr, 0, 0, dpr, 0, 0);
  g.clearRect(0, 0, r.width, r.height);
  const d = st.d;
  if (st.follow && d) centerPlayer(st);
  const C = { floor: css(cv, '--floor'), bad: css(cv, '--bad'), npc: css(cv, '--mana'), merc: css(cv, '--stam'), item: css(cv, '--q-unique'), acc: css(cv, '--accent'), ink: css(cv, '--ink'), muted: css(cv, '--muted') };
  if (st.terrain && st.terrain.w > 0) {
    const t = st.terrain, off = offscreen(st, C.floor);
    g.imageSmoothingEnabled = false;
    const [sx, sy] = toScreen(st, r, t.x0, t.y0);
    g.drawImage(off, sx, sy, t.w * st.zoom, t.h * st.zoom);
  } else {
    g.fillStyle = C.muted; g.font = '13px system-ui'; g.textAlign = 'center'; g.fillText('no map data yet', r.width / 2, r.height / 2);
  }
  if (!d) return;
  const u = Math.max(3, Math.min(8, st.zoom * 1.1));
  g.lineWidth = 1.5; g.textAlign = 'left'; g.font = '11px system-ui';
  for (const e of d.exits) {
    const [x, y] = toScreen(st, r, e.x, e.y);
    g.fillStyle = C.acc; g.strokeStyle = C.ink;
    g.beginPath(); g.moveTo(x, y - u - 2); g.lineTo(x + u + 2, y + u); g.lineTo(x - u - 2, y + u); g.closePath(); g.fill(); g.stroke();
    if (st.zoom >= 2) { g.fillStyle = C.ink; g.fillText(e.name || (e.toArea ? '→ ' + e.toArea : e.kind), x + u + 4, y + 3); }
  }
  for (const m of d.units) {
    const [x, y] = toScreen(st, r, m.x, m.y);
    if (x < -10 || y < -10 || x > r.width + 10 || y > r.height + 10) continue;
    g.beginPath();
    if (m.dead) { g.strokeStyle = C.muted; g.moveTo(x - u / 2, y - u / 2); g.lineTo(x + u / 2, y + u / 2); g.moveTo(x + u / 2, y - u / 2); g.lineTo(x - u / 2, y + u / 2); g.stroke(); continue; }
    if (m.kind === 'item') { g.fillStyle = C.item; g.fillRect(x - u / 2, y - u / 2, u, u); }
    else if (m.kind === 'object') { g.strokeStyle = C.muted; g.strokeRect(x - u / 2, y - u / 2, u, u); }
    else if (m.kind === 'npc') { g.fillStyle = C.npc; g.moveTo(x, y - u); g.lineTo(x + u, y); g.lineTo(x, y + u); g.lineTo(x - u, y); g.closePath(); g.fill(); }
    else { g.fillStyle = m.kind === 'merc' ? C.merc : C.bad; g.arc(x, y, u * 0.8, 0, 7); g.fill(); }
    if (m.name && st.zoom >= 4 && m.kind !== 'item') { g.fillStyle = C.ink; g.fillText(m.name, x + u + 2, y + 3); }
  }
  const [px, py] = toScreen(st, r, d.px, d.py);
  g.strokeStyle = C.acc; g.fillStyle = C.acc; g.lineWidth = 2;
  g.beginPath(); g.arc(px, py, u + 3, 0, 7); g.stroke();
  g.beginPath(); g.arc(px, py, 2.5, 0, 7); g.fill();
  g.beginPath(); g.moveTo(px - u - 7, py); g.lineTo(px - u, py); g.moveTo(px + u, py); g.lineTo(px + u + 7, py); g.moveTo(px, py - u - 7); g.lineTo(px, py - u); g.moveTo(px, py + u); g.lineTo(px, py + u + 7); g.stroke();
}

function hover(st) {
  if (!st.mouse || !st.d) return;
  const r = st.canvas.getBoundingClientRect();
  const wx = Math.round(st.cx + (st.mouse.x - r.width / 2) / st.zoom), wy = Math.round(st.cy + (st.mouse.y - r.height / 2) / st.zoom);
  let near = null, nd = 12;
  for (const m of [...st.d.units, ...st.d.exits]) {
    const [x, y] = toScreen(st, r, m.x, m.y); const dd = Math.hypot(x - st.mouse.x, y - st.mouse.y);
    if (dd < nd) { nd = dd; near = m; }
  }
  st.info.textContent = `(${wx}, ${wy})` + (near ? `  ${near.name || near.kind}${near.dead ? ' (dead)' : ''} at ${near.x}, ${near.y}` : '');
}

export default {
  id: 'world',
  span: 7,
  render(el, d, ctx) {
    const st = el._w || (el._w = (clear(el), build(el, ctx)));
    st.d = d;
    ctx.aside.textContent = `${d.areaName || 'no area'}  player ${d.px}, ${d.py}`;
    if (d.rev !== st.rev && d.x1 > d.x0) loadTerrain(st, d);
    draw(st);
  },
};
