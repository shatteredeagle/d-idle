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
export const TRAIL_WINDOWS = [30000, 60000, 300000, 1800000];
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

// Activity colours: tokens of the design system, so the dark map and the stream overlay both read them. Index = the
// `a` code of a trail point (legend order of the payload: walk, combat, town, farm, flee, rest).
export const ACT_TOKEN = ['--d-info', '--d-bad', '--d-accent', '--d-good', '--d-warn', '--d-text-2', '--d-good'];
export const ACT_FALLBACK = ['#7ab0e8', '#ec7d6c', '#c7a44a', '#6fbf73', '#e0a030', '#a89d8a', '#6fbf73'];
const ACT_NAME = ['walk', 'combat', 'town', 'farm', 'flee', 'rest', 'loot'];

// Share of the window per activity, and 'stall' for the time spent standing still over STALL_MS, as one line:
// 'walk 41% | combat 33% | stall 5%'. Time of a sample runs until the next sample (the newest runs to the reply).
export function activityShare(plan, names) {
  const nm = names && names.length ? names : ACT_NAME, tot = {}, P = plan.pts;
  let all = 0;
  for (let i = 0; i < P.length; i++) {
    const dur = P[i].ago - (i + 1 < P.length ? P[i + 1].ago : 0);
    if (dur <= 0 || (P[i + 1] && P[i + 1].e)) continue;
    const key = dur > STALL_MS ? 'stall' : (nm[P[i].a || 0] || 'walk');
    tot[key] = (tot[key] || 0) + dur; all += dur;
  }
  if (!all) return '';
  return Object.keys(tot).sort((a, b) => tot[b] - tot[a]).map((k) => k + ' ' + Math.round(tot[k] * 100 / all) + '%').join(' \u00b7 ');
}

// The Events filter: 'key' marks kills, level-ups, deaths and big loot; 'all' every event; 'off' none.
export const EVENT_MODES = ['key', 'all', 'off'];
export function eventShown(ev, mode) {
  if (mode === 'off') return false;
  if (mode === 'all') return ev.k !== 'fight';
  return ev.k === 'roamfail' || ev.k === 'gatehold' || ev.k === 'ignore' || ev.k === 'kill' || ev.k === 'chicken' || ev.k === 'portal' || ev.k === 'levelup' || ev.k === 'death' || (ev.k === 'loot' && !!ev.big);
}

// The lines of a fight card from a fight event (k 'fight', f the summary, ago the end of the fight).
export function fightCardLines(ev) {
  const f = ev && ev.f;
  if (!f) return [];
  const lines = ['Fight ended ' + (ev.ago / 1000).toFixed(ev.ago < 10000 ? 1 : 0) + ' s ago, lasted ' + f.sec + ' s', 'End: ' + (f.end || '?') + (f.left ? ', left by ' + f.left : ''),
    f.kills + (f.kills === 1 ? ' kill' : ' kills') + ', damage ' + (f.damage || 0) + ' life points, lowest life ' + f.min + '% (ended at ' + f.life + '%)'];
  const bits = [];
  [['potions', 'potions'], ['casts', 'casts'], ['kites', 'kites'], ['approach', 'approaches'], ['dodges', 'dodges']].forEach(([k, n]) => { if (f[k]) bits.push(f[k] + ' ' + n); });
  if (bits.length) lines.push(bits.join(', '));
  return lines;
}

// The fight event that covers a combat sample `ago` ms old (the fight ended at ev.ago and began f.sec earlier).
export function fightAt(evs, ago) {
  let best = null;
  (evs || []).forEach((e) => {
    if (e.k !== 'fight' || !e.f) return;
    if (ago >= e.ago - 2000 && ago <= e.ago + e.f.sec * 1000 + 2000 && (!best || e.ago < best.ago)) best = e;
  });
  return best;
}

// The tooltip text of an event, with its fight summary when the server knew one.
export function eventText(ev, now) {
  const f = ev.f;
  if (ev.s) return (ev.t || ev.k) + ', ' + Math.max(0, Math.round(ev.s - (now === undefined ? ev.ago : now) / 1000)) + ' s of ' + ev.s + ' s left';
  return (ev.t || ev.k) + (f ? ', ' + f.sec + ' s fight, life ' + f.life + '% (min ' + f.min + '%), end=' + (f.end || '?') + ', ' + f.kills + (f.kills === 1 ? ' kill' : ' kills') : '');
}

// ---- Heat layers ----
export const HEAT_MODES = ['off', 'kills', 'time', 'damage'];
const HEAT_COL = { kills: 2, time: 3, damage: 4 }; // index of the count in a heat cell [x, y, kills, dwell s, damage]

export function heatMax(cells, mode) {
  const i = HEAT_COL[mode];
  let m = 0;
  if (i) (cells || []).forEach((c) => { if (c[i] > m) m = c[i]; });
  return m;
}

// Square-root scale: a few busy cells do not wash out the rest. 0 draws nothing.
export function heatAlpha(v, max) {
  return v > 0 && max > 0 ? 0.12 + 0.58 * Math.sqrt(v / max) : 0;
}

export function heatLegend(mode, max) {
  if (mode === 'off') return '';
  const unit = mode === 'kills' ? ' kills' : mode === 'time' ? ' s' : ' life points';
  return 'Heat ' + mode + ': from 1 to ' + max + unit + ' per cell (dark to bright).';
}

// ---- Timeline (scrubber and replay) ----
// Activity segments of the visible window for the ribbon: {a, t0, t1} with t = ms ago (t0 older).
export function ribbonSegs(pts) {
  const out = [];
  for (let i = 0; i < pts.length; i++) {
    const t1 = i + 1 < pts.length ? pts[i + 1].ago : 0, a = pts[i].a || 0;
    if (pts[i].ago <= t1 || (pts[i + 1] && pts[i + 1].e)) continue;
    const last = out[out.length - 1];
    if (last && last.a === a && last.t1 === pts[i].ago) last.t1 = t1; else out.push({ a, t0: pts[i].ago, t1 });
  }
  return out;
}

// The hero position at `ago` ms before the reply, interpolated along the trail; null when the trail has no such time.
export function posAt(pts, ago) {
  if (!pts.length || ago > pts[0].ago) return null;
  for (let i = 0; i < pts.length; i++) {
    const t1 = i + 1 < pts.length ? pts[i + 1].ago : 0;
    if (ago <= pts[i].ago && ago >= t1) {
      const nxt = pts[i + 1];
      if (!nxt || nxt.e || pts[i].ago === t1) return { x: pts[i].x, y: pts[i].y };
      const f = (pts[i].ago - ago) / (pts[i].ago - t1);
      return { x: pts[i].x + (nxt.x - pts[i].x) * f, y: pts[i].y + (nxt.y - pts[i].y) * f };
    }
  }
  return null;
}

// The sample of the history nearest to `ago` ms before trailTs (samples carry ages relative to histTs).
export function sampleAt(samples, histTs, trailTs, ago) {
  let best = null, bd = Infinity;
  (samples || []).forEach((s) => {
    const d = Math.abs((trailTs - (histTs - s.ago)) - ago);
    if (d < bd) { bd = d; best = s; }
  });
  return bd <= 6000 ? best : null;
}

// One line for the scrubber: what the bot was doing and its state then.
export function scrubText(sample, names, ago) {
  const when = ago <= 0 ? 'now' : (ago / 1000).toFixed(ago < 10000 ? 1 : 0) + ' s ago';
  if (!sample) return when + ': no sample';
  const nm = (names && names[sample.a || 0]) || ACT_NAME[sample.a || 0] || 'walk';
  return when + ': ' + nm + (sample.w ? ' (' + sample.w + ')' : '') + ', life ' + sample.l + '%, mana ' + sample.m + '%' + (sample.lv ? ', level ' + sample.lv : '') + (sample.xp ? ', xp ' + sample.xp : '');
}

export const REPLAY_SPEEDS = [1, 4, 16];

// Cheap change test: the plan is redrawn only when this string moves.
export function trailSig(plan, win, area, ne) {
  const l = plan.pts[plan.pts.length - 1];
  return [area, win, ne || 0, plan.pts.length, l ? l.x + ',' + l.y + ',' + (plan.stalls.length ? plan.stalls[plan.stalls.length - 1].ms : 0) : '-'].join('|');
}

// Fade: the newest end is opaque, the oldest end of the window is faint (never invisible).
export function trailAlpha(ago, win) {
  return Math.max(0.15, 1 - 0.85 * Math.min(1, ago / win));
}

// The coverage status line: 'roam: 41/118 points seen, 3 failed, leg 17'. cov is the /api/senses coverage block.
export function coverSummary(cov) {
  if (!cov || !cov.cells || !cov.cells.length) return '';
  let seen = cov.seen, failed = cov.failed; // counted by the server over every point, not only the drawn ones
  if (seen === undefined) { seen = 0; failed = 0; cov.cells.forEach((c) => { if (c[2] === 1) seen++; else if (c[2] === 2) failed++; }); }
  const n = cov.total || cov.cells.length;
  return (cov.state === 'explored' ? 'explored: ' : 'roam: ') + seen + '/' + n + ' points seen, ' + failed + ' failed, leg ' + (cov.legs || 0) +
    (cov.held ? ', held ' + cov.held : '') +
    (cov.drawn && cov.drawn < n ? ', drawing ' + cov.drawn + ' of ' + n : '');
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
    const b = ctx.el('button', 'd-btn', ms < 60000 ? '30 s' : ms < 1800000 ? ms / 60000 + ' min' : '30 min');
    b.type = 'button'; b.dataset.size = 'sm'; b.dataset.key = 'trail-' + ms; b.setAttribute('aria-label', 'Trail window ' + b.textContent);
    return b;
  });
  const covStore = ctx.store('coverage');
  let covOn = false;
  try { covOn = covStore.get() === '1'; } catch (e) { /* storage blocked */ }
  const covBtn = ctx.el('button', 'd-btn', 'Coverage');
  covBtn.type = 'button'; covBtn.dataset.size = 'sm'; covBtn.dataset.key = 'coverage'; covBtn.setAttribute('aria-label', 'Show the exploration coverage grid');
  covBtn.setAttribute('aria-pressed', String(covOn));
  const covLine = ctx.el('p', 'note cover-line');
  let cov = null;
  function covText() {
    covLine.textContent = covOn ? (coverSummary(cov) || 'Coverage: the bot is not roaming this area.') : '';
    covLine.hidden = !covOn;
  }
  covBtn.addEventListener('click', () => {
    covOn = !covOn;
    try { covStore.set(covOn ? '1' : '0'); } catch (e) { /* storage blocked */ }
    covBtn.setAttribute('aria-pressed', String(covOn));
    covText(); draw();
  });
  const evStore = ctx.store('events');
  let evMode = 'key';
  try { const v = evStore.get(); if (EVENT_MODES.indexOf(v) >= 0) evMode = v; } catch (e) { /* storage blocked */ }
  const evBtn = ctx.el('button', 'd-btn', 'Events: ' + evMode);
  evBtn.type = 'button'; evBtn.dataset.size = 'sm'; evBtn.dataset.key = 'events'; evBtn.setAttribute('aria-label', 'Event markers on the trail: key events, all, or off');
  evBtn.addEventListener('click', () => {
    evMode = EVENT_MODES[(EVENT_MODES.indexOf(evMode) + 1) % EVENT_MODES.length];
    try { evStore.set(evMode); } catch (e) { /* storage blocked */ }
    evBtn.textContent = 'Events: ' + evMode;
    draw();
  });
  const heatStore = ctx.store('heat');
  let heatMode = 'off', heat = null;
  try { const v = heatStore.get(); if (HEAT_MODES.indexOf(v) >= 0) heatMode = v; } catch (e) { /* storage blocked */ }
  const heatBtn = ctx.el('button', 'd-btn', 'Heat: ' + heatMode), heatLine = ctx.el('p', 'note heat-line');
  heatBtn.type = 'button'; heatBtn.dataset.size = 'sm'; heatBtn.dataset.key = 'heat'; heatBtn.setAttribute('aria-label', 'Heat layer: kills, time spent or damage taken, per cell');
  function heatText() { heatLine.textContent = heatLegend(heatMode, heat ? heatMax(heat.cells, heatMode) : 0); heatLine.hidden = heatMode === 'off'; }
  heatBtn.addEventListener('click', () => {
    const was = heatMode;
    heatMode = HEAT_MODES[(HEAT_MODES.indexOf(heatMode) + 1) % HEAT_MODES.length];
    try { heatStore.set(heatMode); } catch (e) { /* storage blocked */ }
    heatBtn.textContent = 'Heat: ' + heatMode;
    heatText(); draw();
    if (was === 'off') wantHeat = 0; // ask for the cells at the next poll
  });
  heatText();
  const tlBtn = ctx.el('button', 'd-btn', 'Timeline');
  tlBtn.type = 'button'; tlBtn.dataset.size = 'sm'; tlBtn.dataset.key = 'timeline'; tlBtn.setAttribute('aria-label', 'Show the timeline: scrub and replay the trail'); tlBtn.setAttribute('aria-pressed', 'false');
  const ribbon = ctx.el('canvas', 'ribbon'), range = ctx.el('input', 'scrub'), playBtn = ctx.el('button', 'd-btn', 'Play'), spdBtn = ctx.el('button', 'd-btn', '1x');
  const tl = ctx.el('div', 'timeline'), scrubLine = ctx.el('p', 'note scrub-line');
  ribbon.width = 320; ribbon.height = 14; ribbon.setAttribute('role', 'img'); ribbon.setAttribute('aria-label', 'Activity ribbon over the visible window');
  range.type = 'range'; range.min = '0'; range.max = '1000'; range.value = '1000'; range.setAttribute('aria-label', 'Scrub the trail in time');
  [[playBtn, 'play', 'Play the trail'], [spdBtn, 'speed', 'Replay speed']].forEach(([b, key, label]) => { b.type = 'button'; b.dataset.size = 'sm'; b.dataset.key = key; b.setAttribute('aria-label', label); });
  tl.append(ribbon, range, playBtn, spdBtn, scrubLine);
  tl.hidden = true;
  const card = ctx.el('div', 'fight-card'), cardClose = ctx.el('button', 'd-btn', 'Close');
  cardClose.type = 'button'; cardClose.dataset.size = 'sm'; cardClose.dataset.key = 'card-close'; cardClose.setAttribute('aria-label', 'Close the fight card');
  card.hidden = true;
  function openCard(ev) {
    const lines = fightCardLines(ev);
    if (!lines.length) return;
    const rows = lines.map((l) => ctx.el('p', 'note', l));
    if (card.replaceChildren) card.replaceChildren(...rows, cardClose); else { card.children = []; card.append(...rows, cardClose); }
    card.hidden = false;
  }
  cardClose.addEventListener('click', () => { card.hidden = true; });
  const legend = ctx.el('p', 'note trail-legend'), share = ctx.el('p', 'note trail-share'), tip = ctx.el('p', 'note trail-tip');
  ctx.body.append(covLine, heatLine, legend, share, tip, card, tl);
  ctx.actions.append(covBtn, heatBtn, evBtn, tlBtn, ...winBtns);
  covText();
  let combatHits = [], wantHeat = 0, scrub = null, hist = null, playing = false, speed = 1, trailTs = 0, polls = 0, wantHist = 0;
  let trail = [], trailArea = 0, plan = { pts: [], stalls: [] }, sig = '', stallHits = [], evs = [], evHits = [], legendNames = [];
  function setWindow(ms) {
    win = ms;
    try { tstore.set(String(ms)); } catch (e) { /* storage blocked */ }
    replan(true);
  }
  winBtns.forEach((b, i) => b.addEventListener('click', () => setWindow(TRAIL_WINDOWS[i])));
  // Scrubber: the range maps the window left (oldest) to right (now); null means live.
  function agoOfRange() { return Math.round(win * (1 - Number(range.value) / 1000)); }
  function scrubTo(ago) {
    scrub = ago === null || ago <= 0 ? null : Math.min(ago, win);
    range.value = String(scrub === null ? 1000 : Math.round((1 - scrub / win) * 1000));
    scrubText_();
    draw();
  }
  function scrubText_() {
    const ago = scrub === null ? 0 : scrub;
    scrubLine.textContent = scrubText(hist && sampleAt(hist.samples, hist.ts, trailTs, ago), hist && hist.legend, ago);
  }
  function drawRibbon() {
    const rg = ribbon.getContext('2d');
    ribbon.width = Math.max(1, Math.round((ribbon.clientWidth || 320) * (window.devicePixelRatio || 1)));
    rg.clearRect(0, 0, ribbon.width, ribbon.height);
    ribbonSegs(plan.pts).forEach((sg) => {
      const x0 = (1 - sg.t0 / win) * ribbon.width, x1 = (1 - sg.t1 / win) * ribbon.width;
      rg.fillStyle = css(ACT_TOKEN[sg.a] || ACT_TOKEN[0], ACT_FALLBACK[sg.a] || ACT_FALLBACK[0]);
      rg.fillRect(x0, 0, Math.max(1, x1 - x0), ribbon.height);
    });
    if (scrub !== null) { rg.fillStyle = css('--d-text', '#eee'); rg.fillRect((1 - scrub / win) * ribbon.width - 1, 0, 2, ribbon.height); }
  }
  function setPlaying(on) {
    playing = on; playBtn.textContent = on ? 'Pause' : 'Play';
    if (on && scrub === null) scrubTo(win);
    if (on) tick();
  }
  // Replay: one step every 200 ms (1 s under reduced motion), advancing speed x the elapsed time.
  function tick() {
    if (!playing || !ctx.later) return;
    const calm = typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches, dt = calm ? 1000 : 200;
    ctx.later(() => {
      if (!playing) return;
      const next = (scrub === null ? 0 : scrub) - dt * speed;
      if (next <= 0) { scrubTo(null); setPlaying(false); return; }
      scrubTo(next); tick();
    }, dt);
  }
  tlBtn.addEventListener('click', () => {
    tl.hidden = !tl.hidden;
    tlBtn.setAttribute('aria-pressed', String(!tl.hidden));
    if (tl.hidden) { setPlaying(false); scrubTo(null); } else { wantHist = 0; drawRibbon(); scrubText_(); }
  });
  range.addEventListener('input', () => { setPlaying(false); scrubTo(agoOfRange()); });
  playBtn.addEventListener('click', () => setPlaying(!playing));
  spdBtn.addEventListener('click', () => { speed = REPLAY_SPEEDS[(REPLAY_SPEEDS.indexOf(speed) + 1) % REPLAY_SPEEDS.length]; spdBtn.textContent = speed + 'x'; });
  function replan(force) {
    plan = trailPlan(trail, win);
    winBtns.forEach((b, i) => b.setAttribute('aria-pressed', String(TRAIL_WINDOWS[i] === win)));
    legend.textContent = plan.pts.length ? 'Trail: last ' + (win >= 60000 ? win / 60000 + ' min' : win / 1000 + ' s') + ' (blue walk, red combat, gold town, green farm, dashed green loot walk-back, orange flee, grey rest; white dots are stops over 3 s, ticks are zone entries).' : '';
    share.textContent = activityShare(plan, legendNames);
    if (!tl.hidden) { drawRibbon(); scrubText_(); }
    const s = trailSig(plan, win, trailArea, evs.length);
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
    drawHeat(at);
    drawCoverage(at);
    drawTrail(at);
    if (scrub !== null) {
      const sp = posAt(plan.pts, scrub);
      if (sp) { const a = at(sp); g.strokeStyle = css('--d-accent', '#c7a44a'); g.lineWidth = 3; g.beginPath(); g.arc(a[0], a[1], 8, 0, 6.2832); g.stroke(); }
    }
    if (hero) {
      const h = at(hero);
      g.strokeStyle = css('--d-bad', '#ec7d6c'); g.lineWidth = 3;
      g.beginPath(); g.moveTo(h[0] - 7, h[1]); g.lineTo(h[0] + 7, h[1]); g.moveTo(h[0], h[1] - 7); g.lineTo(h[0], h[1] + 7); g.stroke();
    }
    if (window.sensesUnitsDraw && units.length) window.sensesUnitsDraw(g, at, units, css, cv);
  }
  // The heat grid sits under the coverage grid: translucent squares of the same cell size, brighter where more happened.
  function drawHeat(at) {
    if (heatMode === 'off' || !heat || !heat.cells || heat.area !== (map && map.areaId)) return;
    const i = HEAT_COL[heatMode], max = heatMax(heat.cells, heatMode), side = (heat.cell || 7) * view().k;
    g.save();
    g.fillStyle = css('--d-bad', '#ec7d6c');
    heat.cells.forEach((c) => {
      const al = heatAlpha(c[i], max);
      if (!al) return;
      const a = at({ x: c[0], y: c[1] });
      g.globalAlpha = al; g.fillRect(a[0] - 0.5 * view().k, a[1] - 0.5 * view().k, side, side);
    });
    g.restore();
  }
  // The coverage grid sits under the trail: faint outline unseen, tinted fill seen, red outline and a slash failed, a ring
  // on the current target. Static on purpose (no pulse), so reduced motion needs nothing special.
  function drawCoverage(at) {
    if (!covOn || !cov || !cov.cells) return;
    const half = (cov.cell || 8) / 2 * view().k, side = half * 2;
    g.save();
    g.lineWidth = 1;
    cov.cells.forEach((c) => {
      const a = at({ x: c[0], y: c[1] }), x = a[0] - half, y = a[1] - half;
      if (c[2] === 1) { g.globalAlpha = 0.28; g.fillStyle = css('--d-good', '#6fbf73'); g.fillRect(x, y, side, side); }
      else if (c[2] === 2) {
        g.globalAlpha = 0.9; g.strokeStyle = css('--d-bad', '#ec7d6c'); g.strokeRect(x, y, side, side);
        g.beginPath(); g.moveTo(x, y + side); g.lineTo(x + side, y); g.stroke();
      } else { g.globalAlpha = 0.35; g.strokeStyle = css('--d-text-2', '#a89d8a'); g.strokeRect(x, y, side, side); }
    });
    if (cov.target) {
      const a = at({ x: cov.target[0], y: cov.target[1] });
      g.globalAlpha = 1; g.lineWidth = 2; g.strokeStyle = css('--d-accent', '#c7a44a');
      g.beginPath(); g.arc(a[0], a[1], Math.max(6, half * 0.7), 0, 6.2832); g.stroke();
    }
    g.restore();
  }
  // The trail sits under the hero cross and the units. Each segment is stroked on its own so its alpha follows its age.
  function drawTrail(at) {
    stallHits = []; evHits = []; combatHits = [];
    if (!plan.pts.length) return;
    const P = plan.pts, col = (k) => css(ACT_TOKEN[k] || ACT_TOKEN[0], ACT_FALLBACK[k] || ACT_FALLBACK[0]);
    g.save();
    g.lineCap = 'round';
    for (let i = 1; i < P.length; i++) {
      if (P[i].e) continue;
      const k = P[i].a || 0, a = at(P[i - 1]), b = at(P[i]);
      g.strokeStyle = col(k); g.lineWidth = k === 1 ? 4 : 2; g.setLineDash(k === 6 ? [4, 3] : []); // loot walk-back is dashed green // combat is the thick red segment
      g.globalAlpha = k === 1 ? Math.max(0.6, trailAlpha((P[i - 1].ago + P[i].ago) / 2, win)) : trailAlpha((P[i - 1].ago + P[i].ago) / 2, win);
      g.beginPath(); g.moveTo(a[0], a[1]); g.lineTo(b[0], b[1]); g.stroke();
    }
    g.globalAlpha = 1; g.lineWidth = 2; g.setLineDash([]);
    P.forEach((p, i) => {
      const a = at(p);
      if (p.e) { g.strokeStyle = col(0); g.beginPath(); g.moveTo(a[0] - 4, a[1] - 4); g.lineTo(a[0] + 4, a[1] + 4); g.stroke(); }
      if ((p.a || 0) === 1) combatHits.push({ x: a[0], y: a[1], ago: p.ago });
      if ((p.a || 0) === 1 && (i === 0 || (P[i - 1].a || 0) !== 1)) { // a fight starts: small crossed swords
        g.strokeStyle = col(1); g.lineWidth = 2;
        g.beginPath(); g.moveTo(a[0] - 5, a[1] - 5); g.lineTo(a[0] + 5, a[1] + 5); g.moveTo(a[0] + 5, a[1] - 5); g.lineTo(a[0] - 5, a[1] + 5); g.stroke();
        g.beginPath(); g.moveTo(a[0] - 3, a[1] + 1); g.lineTo(a[0] + 3, a[1] + 1); g.stroke();
      }
    });
    g.fillStyle = css('--d-text', '#eee');
    plan.stalls.forEach((s) => {
      const a = at(s);
      g.globalAlpha = trailAlpha(s.ago, win);
      g.beginPath(); g.arc(a[0], a[1], 3, 0, 6.2832); g.fill();
      stallHits.push({ x: a[0], y: a[1], text: 'Stood still ' + (s.ms / 1000).toFixed(1) + ' s here.' });
    });
    g.globalAlpha = 1;
    evs.forEach((e) => { if (e.ago <= win && eventShown(e, evMode)) drawEvent(at(e), e); });
    g.restore();
  }
  // Glyphs by kind, bigger when the event matters (boss or unique kill, level-up, death).
  function drawEvent(a, e) {
    const r = (e.big || e.k === 'levelup' || e.k === 'death') ? 6 : 4, x = a[0], y = a[1];
    g.lineWidth = 2; g.lineCap = 'round';
    switch (e.k) {
      case 'kill': g.strokeStyle = css('--d-bad', '#ec7d6c'); g.beginPath(); g.moveTo(x - r, y - r); g.lineTo(x + r, y + r); g.moveTo(x + r, y - r); g.lineTo(x - r, y + r); g.stroke(); break;
      case 'death': g.strokeStyle = css('--d-text', '#eee'); g.fillStyle = css('--d-bad', '#ec7d6c'); g.beginPath(); g.arc(x, y, r, 0, 6.2832); g.fill(); g.stroke(); break;
      case 'damage': g.fillStyle = css('--d-warn', '#e0a030'); g.beginPath(); g.moveTo(x, y - r); g.lineTo(x + r, y + r); g.lineTo(x - r, y + r); g.closePath(); g.fill(); break;
      case 'potion': g.strokeStyle = css('--d-info', '#7ab0e8'); g.beginPath(); g.arc(x, y + 1, r - 1, 0, 6.2832); g.moveTo(x, y - r + 1); g.lineTo(x, y - r - 2); g.stroke(); break;
      case 'loot': g.fillStyle = css('--d-accent', '#c7a44a'); g.beginPath(); g.arc(x, y, r - 1, 0, 6.2832); g.fill(); break;
      case 'levelup': g.fillStyle = css('--d-accent', '#c7a44a'); g.beginPath(); for (let i = 0; i < 10; i++) { const rr = i % 2 ? r / 2 : r, an = i * Math.PI / 5 - Math.PI / 2; g.lineTo(x + rr * Math.cos(an), y + rr * Math.sin(an)); } g.closePath(); g.fill(); break;
      case 'chicken': g.strokeStyle = css('--d-warn', '#e0a030'); g.lineWidth = 3; g.beginPath(); g.moveTo(x - r, y - r); g.lineTo(x + r, y + r); g.moveTo(x + r, y - r); g.lineTo(x - r, y + r); g.stroke(); break;
      case 'portal': g.strokeStyle = css('--d-info', '#7ab0e8'); g.beginPath(); g.arc(x, y, r, 0, 6.2832); g.arc(x, y, r / 2, 0, 6.2832); g.stroke(); break;
      case 'waypoint': g.fillStyle = css('--d-accent', '#c7a44a'); g.beginPath(); g.moveTo(x, y - r); g.lineTo(x + r, y); g.lineTo(x, y + r); g.lineTo(x - r, y); g.closePath(); g.fill(); break;
      case 'roamfail': g.strokeStyle = css('--d-bad', '#ec7d6c'); g.strokeRect(x - r + 1, y - r + 1, 2 * r - 2, 2 * r - 2); break;
      case 'gatehold': g.fillStyle = css('--d-warn', '#e0a030'); g.fillRect(x - r + 1, y - r + 1, 2 * r - 2, 2 * r - 2); break;
      case 'ignore': { // hatched circle; the arc is the share of the ignore window still to run
        const left = e.s ? Math.max(0, 1 - e.ago / 1000 / e.s) : 1;
        g.strokeStyle = css('--d-warn', '#e0a030'); g.beginPath(); g.arc(x, y, r + 2, 0, 6.2832 * left); g.stroke();
        g.lineWidth = 1; g.beginPath(); g.moveTo(x - r, y + r); g.lineTo(x + r, y - r); g.moveTo(x - r, y); g.lineTo(x, y - r); g.moveTo(x, y + r); g.lineTo(x + r, y); g.stroke(); break;
      }
      case 'fallback': g.fillStyle = css('--d-text-2', '#a89d8a'); g.beginPath(); g.moveTo(x - r, y - r); g.lineTo(x + r, y - r); g.lineTo(x, y + r); g.closePath(); g.fill(); break;
      case 'yield': g.strokeStyle = css('--d-text-2', '#a89d8a'); g.beginPath(); g.moveTo(x - r, y); g.lineTo(x + r, y); g.stroke(); break;
      default: return;
    }
    evHits.push({ x, y, text: eventText(e), ev: e });
  }
  // Hover or tap on a stop dot names how long the hero stood there.
  function showStall(ev) {
    const r = cv.getBoundingClientRect(), q = cv.width / (r.width || cv.width);
    const px = (ev.clientX - (r.left || 0)) * q, py = (ev.clientY - (r.top || 0)) * q;
    let best = null, bd = 12 * (window.devicePixelRatio || 1);
    evHits.concat(stallHits).forEach((h) => { const d = Math.hypot(h.x - px, h.y - py); if (d <= bd) { bd = d; best = h; } });
    tip.textContent = best ? best.text : '';
    if (best && best.ev && best.ev.f) openCard(best.ev);
    else if (!best) {
      let cb = null, cd = 10 * (window.devicePixelRatio || 1);
      combatHits.forEach((h) => { const d = Math.hypot(h.x - px, h.y - py); if (d <= cd) { cd = d; cb = h; } });
      const fe = cb && fightAt(evs, cb.ago);
      if (fe) openCard(fe);
    }
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
      // the timeline's samples ride the poll every 10th time (about 5 s), and at once when it opens
      const heatOn = heatMode !== 'off', due = (!tl.hidden || heatOn) && (polls++ % 10 === 0 || wantHist === 0 || wantHeat === 0);
      if (due) { wantHist = 1; wantHeat = 1; }
      return '?v=' + version + (due ? '&hist=1' + (heatOn ? '&heat=1' : '') : '') + (Object.keys(nbCache).length ? '&nb=' + encodeURIComponent(nbParam(nbCache)) : '');
    },
    update(r) {
      if (!r || !r.tiers_unlocked || r.tiers_unlocked.indexOf(1) < 0) { ctx.hide(true); return; }
      ctx.hide(false);
      if (r.map) {
        map = r.map; cells = decode(map); version = r.version; panX = panY = 0;
        ctx.meta.textContent = map.area || ('area ' + map.areaId);
      }
      if (r.history && r.history.samples) {
        hist = r.history; if (!tl.hidden) scrubText_();
        if (r.history.heat) { heat = r.history.heat; heatText(); }
      }
      hero = r.hero || hero;
      takeNeighbours(r.neighbours);
      units = r.units || [];
      cov = r.coverage && r.coverage.area === (map && map.areaId) ? r.coverage : null;
      covText();
      if (r.trail) { trail = r.trail; trailTs = r.ts || 0; trailArea = r.trail_area || 0; evs = r.events || []; legendNames = r.trail_legend || []; } else if (!r.waiting) { trail = []; evs = []; }
      note.textContent = r.waiting ? 'Waiting for the first area...' : 'Legend: cross hero, diamond waypoint, square exit' +
        (nbs.length ? '; dimmer areas are the ones next door' : '') + '. Fog clears as the hero walks.';
      replan(true);
    },
  };
}
