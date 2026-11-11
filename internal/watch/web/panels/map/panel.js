// Side-panel minimap (stage band; docs/senses-map.md, docs/senses-units.md). View only: the shell polls GET /api/senses
// about twice a second with the query this panel builds (v= the map version it holds, nb= the neighbour versions), the
// map is cached by version and drawn top-down on a canvas sized from the panel's width. The unit layer is the
// senses_units.js page script (window.sensesUnitsDraw). Hidden until tier 1 is open.

function decode(m) {
  const n = m.w * m.h, out = new Uint8Array(n), r = m.rle || [];
  let o = 0;
  for (let i = 0; i + 1 < r.length; i += 2) for (let c = 0; c < r[i + 1] && o < n; c++) out[o++] = r[i];
  return out;
}

// The world box in subtiles of the current area's frame: the current map plus every neighbour moved by its dx, dy.
export function union(cur, list) {
  const b = { x0: cur.x0, y0: cur.y0, x1: cur.x0 + cur.w * cur.step, y1: cur.y0 + cur.h * cur.step };
  list.forEach((n) => {
    const m = n.map;
    b.x0 = Math.min(b.x0, m.x0 + n.dx); b.y0 = Math.min(b.y0, m.y0 + n.dy);
    b.x1 = Math.max(b.x1, m.x0 + n.dx + m.w * m.step); b.y1 = Math.max(b.y1, m.y0 + n.dy + m.h * m.step);
  });
  return b;
}

// The nb= query value: the neighbour versions this page already holds.
export function nbParam(cache) {
  return Object.keys(cache).map((k) => k + ':' + cache[k].version).join(',');
}

// Trail windows (ms) the selector offers; the default is one minute. A stored value outside the list is ignored.
export const TRAIL_WINDOWS = [30000, 60000, 300000];
export const STALL_MS = 3000;

export function trailWindow(stored) {
  const n = Number(stored);
  return TRAIL_WINDOWS.indexOf(n) >= 0 ? n : 60000;
}

// Cuts the reply's trail (ago = age in ms at the reply, oldest first) to the window and finds the stalls: a gap longer
// than STALL_MS between two samples (the server only samples when the position changed), and the newest sample if the
// hero has stood there that long. Entry samples break the line and carry a tick.
export function trailPlan(trail, win) {
  const pts = (trail || []).filter((p) => p.ago <= win), stalls = [];
  for (let i = 0; i < pts.length; i++) {
    const next = i + 1 < pts.length ? pts[i + 1].ago : 0, gap = pts[i].ago - next;
    if (gap > STALL_MS && !(pts[i + 1] && pts[i + 1].e)) stalls.push({ x: pts[i].x, y: pts[i].y, ms: gap, ago: next });
  }
  return { pts, stalls };
}

// Cheap change test: the plan is redrawn only when this string moves.
export function trailSig(plan, win, area) {
  const l = plan.pts[plan.pts.length - 1];
  return [area, win, plan.pts.length, l ? l.x + ',' + l.y + ',' + (plan.stalls.length ? plan.stalls[plan.stalls.length - 1].ms : 0) : '-'].join('|');
}

// Fade: the newest end is opaque, the oldest end of the window is faint (never invisible).
export function trailAlpha(ago, win) {
  return Math.max(0.15, 1 - 0.85 * Math.min(1, ago / win));
}

export function mount(root, ctx) {
  const cv = ctx.el('canvas', 'cv');
  cv.width = 320; cv.height = 320;
  cv.setAttribute('role', 'img');
  cv.setAttribute('aria-label', 'Top-down map of the area; the cross is the hero');
  const note = ctx.el('p', 'note');
  ctx.body.append(cv, note);
  const zin = ctx.el('button', 'd-btn', '+'), zout = ctx.el('button', 'd-btn', '-'), hide = ctx.el('button', 'd-btn', 'Hide');
  [[zin, 'Zoom in', 'zin'], [zout, 'Zoom out', 'zout'], [hide, 'Hide the map', 'hide']].forEach(([b, label, key]) => {
    b.type = 'button'; b.dataset.size = 'sm'; b.dataset.key = key; b.setAttribute('aria-label', label);
  });
  hide.setAttribute('aria-expanded', 'true');
  ctx.actions.append(zin, zout, hide);
  ctx.hide(true);

  // Trail: the window selector (remembered per viewer), the legend line and the stall read-out.
  const tstore = ctx.store('trailwin');
  let win = 60000;
  try { win = trailWindow(tstore.get()); } catch (e) { /* storage blocked */ }
  const winBtns = TRAIL_WINDOWS.map((ms) => {
    const b = ctx.el('button', 'd-btn', ms < 60000 ? '30 s' : ms === 60000 ? '1 min' : '5 min');
    b.type = 'button'; b.dataset.size = 'sm'; b.dataset.key = 'trail-' + ms; b.setAttribute('aria-label', 'Trail window ' + b.textContent);
    return b;
  });
  const legend = ctx.el('p', 'note trail-legend'), tip = ctx.el('p', 'note trail-tip');
  ctx.body.append(legend, tip);
  ctx.actions.append(...winBtns);
  let trail = [], trailArea = 0, plan = { pts: [], stalls: [] }, sig = '', stallHits = [];
  function setWindow(ms) {
    win = ms;
    try { tstore.set(String(ms)); } catch (e) { /* storage blocked */ }
    replan(true);
  }
  winBtns.forEach((b, i) => b.addEventListener('click', () => setWindow(TRAIL_WINDOWS[i])));
  function replan(force) {
    plan = trailPlan(trail, win);
    winBtns.forEach((b, i) => b.setAttribute('aria-pressed', String(TRAIL_WINDOWS[i] === win)));
    legend.textContent = plan.pts.length ? 'Trail: last ' + (win >= 60000 ? win / 60000 + ' min' : win / 1000 + ' s') + ' (bright is newest, dots are stops over 3 s, ticks are zone entries).' : '';
    const s = trailSig(plan, win, trailArea);
    if (force || s !== sig) { sig = s; draw(); }
  }

  const g = cv.getContext('2d');
  let map = null, cells = null, hero = null, units = [], version = 0, nbs = [];
  const nbCache = {}; // neighbour areas (docs/senses-map.md "Contiguous areas"), by area id
  let zoom = 0, panX = 0, panY = 0, drag = null;

  const css = (name, fb) => getComputedStyle(root).getPropertyValue(name).trim() || fb;
  // Base scale fits the whole stitched world; zoom multiplies it. The view centres on the hero plus the pan offset.
  function view() {
    const u = union(map, nbs), k0 = Math.min(cv.width / (u.x1 - u.x0), cv.height / (u.y1 - u.y0)), k = k0 * Math.pow(1.5, zoom);
    let hx = hero ? hero.x : (u.x0 + u.x1) / 2, hy = hero ? hero.y : (u.y0 + u.y1) / 2;
    if (zoom === 0) { hx = (u.x0 + u.x1) / 2; hy = (u.y0 + u.y1) / 2; }
    return { k, ox: cv.width / 2 - (hx + panX) * k, oy: cv.height / 2 - (hy + panY) * k };
  }
  function drawArea(m, cl, dx, dy, v) {
    for (let y = 0; y < m.h; y++) for (let x = 0; x < m.w; x++) {
      const c = cl[y * m.w + x];
      if (!c) continue;
      g.fillStyle = c === 1 ? css('--d-surface-3', '#2d2720') : css('--d-line-strong', '#4a4036');
      g.fillRect(v.ox + (m.x0 + dx + x * m.step) * v.k, v.oy + (m.y0 + dy + y * m.step) * v.k, Math.ceil(m.step * v.k), Math.ceil(m.step * v.k));
    }
    const at = (p) => [v.ox + (p.x + dx + 0.5) * v.k, v.oy + (p.y + dy + 0.5) * v.k];
    g.fillStyle = css('--d-info', '#7ab0e8');
    (m.exits || []).forEach((e) => { const a = at(e); g.fillRect(a[0] - 4, a[1] - 4, 8, 8); });
    g.fillStyle = css('--d-accent', '#c7a44a');
    (m.waypoints || []).forEach((w) => {
      const b = at(w);
      g.beginPath(); g.moveTo(b[0], b[1] - 6); g.lineTo(b[0] + 5, b[1]); g.lineTo(b[0], b[1] + 6); g.lineTo(b[0] - 5, b[1]); g.closePath(); g.fill();
    });
  }
  // The canvas buffer follows the panel's width (cqi in CSS), so the map stays sharp at every stage size.
  function fit() {
    const w = Math.round((cv.clientWidth || 320) * (window.devicePixelRatio || 1));
    if (w > 0 && w !== cv.width) { cv.width = w; cv.height = w; }
  }
  function draw() {
    fit();
    g.clearRect(0, 0, cv.width, cv.height);
    if (!map || !cells) return;
    const v = view();
    g.globalAlpha = 0.55; // neighbours sit behind the current area, dimmer
    nbs.forEach((n) => drawArea(n.map, n.cells, n.dx, n.dy, v));
    g.globalAlpha = 1;
    drawArea(map, cells, 0, 0, v);
    const at = (p) => [v.ox + (p.x + 0.5) * v.k, v.oy + (p.y + 0.5) * v.k];
    drawTrail(at);
    if (hero) {
      const h = at(hero);
      g.strokeStyle = css('--d-bad', '#ec7d6c'); g.lineWidth = 3;
      g.beginPath(); g.moveTo(h[0] - 7, h[1]); g.lineTo(h[0] + 7, h[1]); g.moveTo(h[0], h[1] - 7); g.lineTo(h[0], h[1] + 7); g.stroke();
    }
    if (window.sensesUnitsDraw && units.length) window.sensesUnitsDraw(g, at, units, css, cv);
  }
  // The trail sits under the hero cross and the units. Each segment is stroked on its own so its alpha follows its age.
  function drawTrail(at) {
    stallHits = [];
    if (!plan.pts.length) return;
    const col = css('--d-info', '#7ab0e8'), warn = css('--d-warn', '#e0a030'), P = plan.pts;
    g.save();
    g.lineWidth = 2; g.lineCap = 'round'; g.strokeStyle = col;
    for (let i = 1; i < P.length; i++) {
      if (P[i].e) continue;
      const a = at(P[i - 1]), b = at(P[i]);
      g.globalAlpha = trailAlpha((P[i - 1].ago + P[i].ago) / 2, win);
      g.beginPath(); g.moveTo(a[0], a[1]); g.lineTo(b[0], b[1]); g.stroke();
    }
    g.globalAlpha = 1; g.strokeStyle = col; g.lineWidth = 2;
    P.forEach((p) => {
      if (!p.e) return;
      const a = at(p); // zone entry: a short tick across the path
      g.beginPath(); g.moveTo(a[0] - 4, a[1] - 4); g.lineTo(a[0] + 4, a[1] + 4); g.stroke();
    });
    g.fillStyle = warn;
    plan.stalls.forEach((s) => {
      const a = at(s);
      g.globalAlpha = trailAlpha(s.ago, win);
      g.beginPath(); g.arc(a[0], a[1], 3, 0, 6.2832); g.fill();
      stallHits.push({ x: a[0], y: a[1], ms: s.ms });
    });
    g.restore();
  }
  // Hover or tap on a stop dot names how long the hero stood there.
  function showStall(ev) {
    const r = cv.getBoundingClientRect(), q = cv.width / (r.width || cv.width);
    const px = (ev.clientX - (r.left || 0)) * q, py = (ev.clientY - (r.top || 0)) * q;
    let best = null, bd = 12 * (window.devicePixelRatio || 1);
    stallHits.forEach((h) => { const d = Math.hypot(h.x - px, h.y - py); if (d <= bd) { bd = d; best = h; } });
    tip.textContent = best ? 'Stood still ' + (best.ms / 1000).toFixed(1) + ' s here.' : '';
    cv.title = tip.textContent;
  }
  // Neighbours arrive with their map only when the page lacks that version.
  function takeNeighbours(list) {
    const keep = {};
    nbs = [];
    (list || []).forEach((n) => {
      let c = nbCache[n.areaId];
      if (n.map) c = nbCache[n.areaId] = { version: n.version, map: n.map, cells: decode(n.map) };
      if (!c || c.version !== n.version) return; // cannot draw it; the next poll asks again
      keep[n.areaId] = true;
      nbs.push({ areaId: n.areaId, dx: n.dx, dy: n.dy, map: c.map, cells: c.cells });
    });
    Object.keys(nbCache).forEach((k) => { if (!keep[k]) delete nbCache[k]; });
  }

  zin.addEventListener('click', () => { zoom = Math.min(zoom + 1, 6); draw(); });
  zout.addEventListener('click', () => { zoom = Math.max(zoom - 1, 0); if (!zoom) panX = panY = 0; draw(); });
  hide.addEventListener('click', () => {
    const min = !cv.hidden;
    cv.hidden = min; note.hidden = min;
    hide.textContent = min ? 'Show' : 'Hide';
    hide.setAttribute('aria-expanded', String(!min));
  });
  cv.addEventListener('pointerdown', (ev) => { showStall(ev); drag = { x: ev.clientX, y: ev.clientY }; cv.setPointerCapture(ev.pointerId); });
  cv.addEventListener('pointerup', () => { drag = null; });
  cv.addEventListener('pointermove', (ev) => {
    if (!drag) showStall(ev);
    if (!drag || !map || !zoom) return;
    const q = cv.width / cv.getBoundingClientRect().width, k = view().k;
    panX -= (ev.clientX - drag.x) * q / k; panY -= (ev.clientY - drag.y) * q / k;
    drag = { x: ev.clientX, y: ev.clientY };
    draw();
  });

  return {
    query() {
      return '?v=' + version + (Object.keys(nbCache).length ? '&nb=' + encodeURIComponent(nbParam(nbCache)) : '');
    },
    update(r) {
      if (!r || !r.tiers_unlocked || r.tiers_unlocked.indexOf(1) < 0) { ctx.hide(true); return; }
      ctx.hide(false);
      if (r.map) {
        map = r.map; cells = decode(map); version = r.version; panX = panY = 0;
        ctx.meta.textContent = map.area || ('area ' + map.areaId);
      }
      hero = r.hero || hero;
      takeNeighbours(r.neighbours);
      units = r.units || [];
      if (r.trail) { trail = r.trail; trailArea = r.trail_area || 0; } else if (!r.waiting) { trail = []; }
      note.textContent = r.waiting ? 'Waiting for the first area...' : 'Legend: cross hero, diamond waypoint, square exit' +
        (nbs.length ? '; dimmer areas are the ones next door' : '') + '. Fog clears as the hero walks.';
      replan(true);
    },
  };
}
