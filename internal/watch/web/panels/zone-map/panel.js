// The act map as a cockpit slot panel (docs/ui/redesign-2026-11-14/J-second-attempt.md "The act map in a rail", slice S5):
// the zones of the act as a grid of one-line buttons (column and row from the node's stored X and Y: three columns by ten
// rows for Act I), a stripe for the level band, the name, and at the right the level range, "here", a queue number or a
// failed mark. The route to the order's target is a gold line behind the buttons. A click opens the zone menu: BESIDE the
// map (in the area-map tile) when the rail has two columns, else over the Orders tile above it, never over a zone button.
// From a zone click to an accepted order is two clicks (the zone, then Go now). Orders go through POST /api/commands, the
// client of the Orders panel (sendCommand), so the Loop and the queue share the one dispatcher. Survival margin is never
// drawn as a number: the menu shows the guard's own words. textContent only.
import { sendCommand, replacing } from '../orders/panel.js';

export const DURS = [['10 min', 10], ['20 min', 20], ['30 min', 30], ['1 h', 60], ['Until I cancel', 0]];
export const REPAIR = [['damaged', 'Repair when worn'], ['always', 'Repair every round']];

// The grid of the act: the distinct X values are the columns, the distinct Y values the rows, north (smaller Y) on top.
export function gridOf(nodes) {
  const xs = [...new Set(nodes.map((n) => n.x))].sort((a, b) => a - b), ys = [...new Set(nodes.map((n) => n.y))].sort((a, b) => a - b);
  const at = (n) => ({ c: xs.indexOf(n.x), r: ys.indexOf(n.y) });
  return { cols: xs.length, rows: ys.length, at };
}

// The level band of a zone for this hero: town, no (not orderable), hard (the guard says no), above, in, below.
export function band(n, level) {
  if (!n.orderable) return 'no';
  if (n.kind === 'town') return 'town';
  if (n.status && n.status.hard) return 'hard';
  const m = /clvl (\d+)(?:-(\d+)|\+)?/.exec(n.range || '');
  if (!m || !level) return 'in';
  const lo = +m[1], hi = m[2] ? +m[2] : 0;
  if (level < lo) return 'above';
  if (hi && level > hi) return 'below';
  return 'in';
}

// The right-hand text of a zone: here, the queue place, a failed mark, else the level range.
export function rightText(n) {
  const s = n.status || {};
  if (s.here) return 'here';
  if (s.queued) return '#' + s.queued;
  if (s.failed) return '!';
  return n.range ? n.range.replace('clvl ', '') : '';
}

// The shortest way from the town to a zone along the edges (ids), [] when there is none.
export function routeTo(nodes, edges, from, to) {
  if (!from || !to) return [];
  const adj = {};
  edges.forEach((e) => { (adj[e.from] = adj[e.from] || []).push(e.to); (adj[e.to] = adj[e.to] || []).push(e.from); });
  const prev = { [from]: '' }, q = [from];
  while (q.length) {
    const x = q.shift();
    if (x === to) break;
    for (const y of adj[x] || []) if (!(y in prev)) { prev[y] = x; q.push(y); }
  }
  if (!(to in prev)) return [];
  const out = [];
  for (let x = to; x; x = prev[x]) out.unshift(x);
  return out;
}

// The zone the arrow key moves to: the nearest zone in that direction by grid position (same row or column first).
export function walk(nodes, cur, key) {
  const g = gridOf(nodes), me = nodes.find((n) => n.id === cur);
  if (!me) return nodes.length ? nodes[0].id : '';
  const p = g.at(me), d = { ArrowUp: [0, -1], ArrowDown: [0, 1], ArrowLeft: [-1, 0], ArrowRight: [1, 0] }[key];
  if (!d) return cur;
  let best = null, bs = 1e9;
  for (const n of nodes) {
    if (n === me) continue;
    const q = g.at(n), dx = q.c - p.c, dy = q.r - p.r;
    if (dx * d[0] + dy * d[1] <= 0) continue; // not in that direction
    const along = Math.abs(dx * d[0] + dy * d[1]), across = Math.abs(d[0] ? dy : dx);
    const sc = across * 3 + along;
    if (sc < bs) { bs = sc; best = n; }
  }
  return best ? best.id : cur;
}

// The words of a node for a screen reader and the tooltip.
export function nodeLabel(n, level) {
  const s = n.status || {}, w = [n.name];
  if (!n.orderable) w.push('not orderable yet');
  else {
    if (n.range) w.push(n.range.replace('clvl', 'level'));
    const b = band(n, level);
    w.push({ hard: 'too hard now', above: 'above your level', below: 'below your level', in: 'in range' }[b] || '');
  }
  if (s.here) w.push('you are here');
  if (s.order) w.push('current order');
  if (s.queued) w.push('queued, place ' + s.queued);
  if (s.failed) w.push('the last order failed');
  return w.filter(Boolean).join(', ');
}

const lastDur = (store) => { try { const v = store.get(); const n = v === null ? NaN : +v; return DURS.some((d) => d[1] === n) ? n : 20; } catch (e) { return 20; } };

export function mount(root, ctx) {
  const el = ctx.el, SVGNS = 'http://www.w3.org/2000/svg';
  const box = el('div', 'zm'), grid = el('div', 'zm-grid'), list = el('div', 'zm-list'), empty = el('div', 'zm-empty');
  list.dataset.scroll = '1';
  list.hidden = true;
  empty.hidden = true;
  box.append(grid, list, empty);
  ctx.body.append(box);
  const durStore = ctx.store('dur');
  const st = { view: null, sel: '', dur: lastDur(durStore), charter: null, loop: { minutes: 20, repair: 'damaged', rounds: 5, open: false }, err: '', sig: '', msig: '', deep: true };

  const nodes = () => (st.view && st.view.nodes) || [];
  const node = (id) => nodes().find((n) => n.id === id);
  const level = () => (st.view && st.view.hero && st.view.hero.clvl) || 0;

  // ---- where the menu opens ----
  // The shell's tile sheet: beside the map in a two-column rail, over the Orders tile above it otherwise, never over a zone button.
  const sheet = ctx.tileSheet('beside'), menu = sheet.el;
  menu.setAttribute('role', 'dialog');
  const place = () => sheet.place();
  // Focus: opening a zone moves it to the menu's first button, closing (Esc, the cross, an accepted order) returns it to the zone's button.
  const inside = () => typeof document !== 'undefined' && !!document.activeElement && menu.contains(document.activeElement);
  function close() {
    const back = !menu.hidden && (inside() || (document.activeElement || {}).tagName === 'BODY') ? st.sel : '';
    st.sel = ''; st.charter = null; st.err = ''; st.loop.open = false;
    menu.hidden = true;
    st.sig = '';
    if (st.view) draw();
    if (back && grid.querySelector) { const z = grid.querySelector('[data-key="z-' + back + '"]'); if (z && z.focus) z.focus(); }
  }
  function open(id) {
    st.sel = id; st.charter = null; st.err = ''; st.loop.open = false;
    drawMenu();
    place();
    menu.hidden = false;
    st.sig = '';
    draw();
    const f = menu.querySelector && (menu.querySelector('[data-primary]') || menu.querySelector('button, select'));
    if (f && f.focus) f.focus();
  }

  // ---- orders ----
  function after(j, again) {
    if (j && j.ok) { close(); return; }
    if (j && j.code === 'charter-needed' && j.charter) { st.charter = { price: j.charter.price, text: j.charter.text, again }; st.err = ''; }
    else { st.charter = null; st.err = (j && (j.error || j.code)) || 'could not do that'; }
    drawMenu();
  }
  function go(n, mode, approve) {
    const a = { area: n.order, until: st.dur ? { minutes: st.dur } : 'forever' };
    if (approve) a.approve = true;
    return sendCommand(ctx, 'go', a, mode || 'now', true).then((j) => after(j, () => go(n, mode, true)));
  }
  function stay() {
    return sendCommand(ctx, 'fight_here', { until: st.dur ? { minutes: st.dur } : 'forever' }, 'now', true).then((j) => after(j));
  }
  function errand(intent, args) { return sendCommand(ctx, intent, args, '', true).then((j) => after(j)); }
  function startLoop(n, approve) {
    const l = st.loop, a = { area: n.order, minutes: l.minutes, repair: l.repair };
    if (l.rounds === 0) a.forever = true; else a.rounds = l.rounds;
    if (approve) a.approve = true;
    return sendCommand(ctx, 'loop', a, '', true).then((j) => after(j, () => startLoop(n, true)));
  }

  // ---- the menu ----
  const line = (cls, text, tone) => { const e = el('div', 'zm-l ' + cls, text); e.title = text; if (tone) e.dataset.tone = tone; return e; };
  const btn = (text, key, primary, fn) => {
    const b = ctx.ui.btn(text, { size: 'sm' });
    b.dataset.key = key;
    if (primary) b.dataset.primary = '1';
    if (fn) b.addEventListener('click', fn);
    return b;
  };
  function chips(onPick) {
    const row = el('div', 'zm-chips');
    DURS.forEach((d) => {
      const c = btn(d[0], 'dur-' + d[1], false, () => { st.dur = d[1]; try { durStore.set(String(d[1])); } catch (e) { /* storage blocked */ } onPick(); });
      c.setAttribute('aria-pressed', String(st.dur === d[1]));
      row.appendChild(c);
    });
    return row;
  }
  function select(key, opts, val, fn) {
    const s = el('select', 'zm-sel');
    s.dataset.key = key;
    opts.forEach((o) => { const x = el('option', '', o[1]); x.value = String(o[0]); if (String(o[0]) === String(val)) x.selected = true; s.appendChild(x); });
    s.addEventListener('change', () => fn(s.value));
    return s;
  }
  function drawMenu() {
    // a redraw (a duration chip, the Loop form) must not drop the focus the user is on
    const keep = inside() && document.activeElement.dataset ? document.activeElement.dataset.key : '';
    drawMenuBody();
    if (keep && menu.querySelector) { const f = menu.querySelector('[data-key="' + keep + '"]'); if (f && f.focus) f.focus(); }
  }
  function drawMenuBody() {
    ctx.ui.clear(menu);
    const n = node(st.sel);
    if (!n) { menu.hidden = true; return; }
    menu.setAttribute('aria-label', 'Order for ' + n.name);
    const s = n.status || {}, lad = (st.view && st.view.ladder) || null;
    menu.dataset.kind = n.kind;
    menu.dataset.sel = n.id;
    const head = el('div', 'zm-mh'), x = el('button', 'zm-x', '×');
    x.type = 'button'; x.setAttribute('aria-label', 'Close the zone menu'); x.dataset.key = 'menu-close';
    x.addEventListener('click', close);
    const mt = el('b', 'zm-mt', n.name);
    mt.title = [n.name, n.range ? n.range.replace('clvl', 'level') : '', n.why].filter(Boolean).join(' - ');
    head.append(mt, el('span', 'zm-rg', n.range ? n.range.replace('clvl', 'level') : ''), x);
    menu.appendChild(head);
    const route = routeTo(nodes(), st.view.edges || [], st.view.town, n.id).map((id) => (node(id) || {}).name).filter(Boolean);
    if (route.length > 1) menu.appendChild(line('zm-route', 'Route: ' + route.join(' > ')));
    if (s.failed) menu.appendChild(line('zm-warn', 'The last order here failed' + (s.failedWhy ? ': ' + s.failedWhy : ''), 'bad'));
    if (!n.orderable) { menu.appendChild(line('zm-note', 'Not orderable yet.')); return; }
    if (n.kind === 'town') { // the town errands: the same buttons as the verb row and the More sheet
      const col = el('div', 'zm-col');
      col.append(btn('Rest in town', 'town', true, () => errand('town')), btn('Repair', 'repair', false, () => errand('repair')),
        btn('Identify all', 'identify', false, () => errand('identify', { all: true })), btn('Sell now', 'sell_now', false, () => errand('sell_now')),
        btn('Revive mercenary', 'revive_merc', false, () => errand('revive_merc')));
      menu.appendChild(col);
      if (st.err) menu.appendChild(line('zm-warn', st.err, 'warn'));
      return;
    }
    const hard = !!s.hard;
    if (hard) menu.appendChild(line('zm-warn', 'Too hard now: ' + (s.hardWhy || 'the hero says no'), 'warn'));
    else if (n.advice) menu.appendChild(line('zm-warn', n.advice, 'warn'));
    menu.appendChild(chips(drawMenu));
    const col = el('div', 'zm-col');
    if (s.here) {
      col.append(btn('Stay and fight', 'stay', true, stay), btn('Back to town', 'town', false, () => errand('town')));
    } else {
      const rep = replacing();
      const goBtn = btn(hard ? 'Ask anyway' : rep ? 'Go now (replaces ' + rep + (rep === 1 ? ' order)' : ' orders)') : 'Go now', 'go', !hard, () => go(n));
      if (rep) goBtn.title = 'Go now cancels ' + rep + ' open ' + (rep === 1 ? 'order' : 'orders') + ' first; Add to queue keeps them';
      col.appendChild(goBtn);
      if (lad) { // Loop and the queue: the control once open, else one muted teaser line
        if (lad.loopOpen) {
          col.appendChild(btn(st.loop.open ? 'Loop: close' : 'Loop...', 'loop', false, () => { st.loop.open = !st.loop.open; drawMenu(); }));
        } else if (lad.loopTeaser) col.appendChild(line('zm-teaser', 'Loop: ' + lad.loopTeaser));
        if (lad.queueOpen) {
          const full = lad.queueSlots > 0 && lad.queueWaiting >= lad.queueSlots;
          const q = btn('Add to queue', 'queue', false, full ? null : () => go(n, 'queue'));
          if (full) { q.disabled = true; col.append(q, line('zm-teaser', lad.queueWaiting + ' of ' + lad.queueSlots + ' waiting' + (lad.queueTeaser ? '; the next slot: ' + lad.queueTeaser : ''))); }
          else col.appendChild(q);
        } else if (lad.queueTeaser) col.appendChild(line('zm-teaser', 'Queue: ' + lad.queueTeaser));
      }
    }
    menu.appendChild(col);
    if (st.loop.open && lad && lad.loopOpen) {
      const f = el('div', 'zm-loop');
      f.append(el('span', 'zm-ll', 'Each round'), select('loop-min', [[10, '10 min'], [20, '20 min'], [30, '30 min'], [60, '1 h']], st.loop.minutes, (v) => { st.loop.minutes = +v; }),
        select('loop-repair', REPAIR, st.loop.repair, (v) => { st.loop.repair = v; }),
        select('loop-rounds', [[3, '3 rounds'], [5, '5 rounds'], [10, '10 rounds'], [20, '20 rounds'], [0, 'until I stop it']], st.loop.rounds, (v) => { st.loop.rounds = +v; }),
        btn('Start loop', 'loop-start', true, () => startLoop(n)));
      menu.appendChild(f);
    }
    if (st.charter) {
      const c = el('div', 'zm-charter');
      c.dataset.pin = 'end';
      c.append(line('zm-warn', 'Above your range. Charter: ' + st.charter.price + ' tally. ' + (st.charter.text || ''), 'warn'), btn('Pay and go', 'charter', true, () => st.charter.again()));
      menu.appendChild(c);
    } else if (st.err) menu.appendChild(line('zm-warn', st.err, 'bad'));
  }

  // ---- the grid ----
  function mark(parent, n) { const s = n.status || {}; if (s.here) parent.dataset.here = '1'; if (s.order) parent.dataset.target = '1'; if (s.queued) parent.dataset.pin = String(s.queued); if (s.failed) parent.dataset.failed = '1'; }
  function draw() {
    const v = st.view;
    if (!v) return;
    const ns = nodes();
    const sig = JSON.stringify([ns, v.edges, level(), st.sel, v.town]);
    if (sig === st.sig) return;
    st.sig = sig;
    ctx.ui.clear(grid); ctx.ui.clear(list);
    grid.hidden = !(v.populated && ns.length);
    list.hidden = v.populated && ns.length > 0;
    empty.hidden = true;
    if (grid.hidden) { drawList(v); return; }
    const g = gridOf(ns);
    grid.style.setProperty('--zm-cols', String(g.cols));
    grid.style.setProperty('--zm-rows', String(g.rows));
    grid.dataset.cols = String(g.cols); grid.dataset.rows = String(g.rows);
    // edges and the gold route as an underlay in grid units
    const svg = document.createElementNS ? document.createElementNS(SVGNS, 'svg') : el('svg', 'zm-lines');
    svg.setAttribute('class', 'zm-lines'); svg.setAttribute('viewBox', '0 0 ' + g.cols + ' ' + g.rows); svg.setAttribute('preserveAspectRatio', 'none'); svg.setAttribute('aria-hidden', 'true');
    const ctr = (id) => { const n = ns.find((x) => x.id === id); if (!n) return null; const p = g.at(n); return [p.c + 0.5, p.r + 0.5]; };
    const tgt = ns.find((n) => (n.status || {}).order), way = tgt ? routeTo(ns, v.edges || [], v.town, tgt.id) : [];
    const onWay = (a, b) => way.some((x, i) => (x === a && way[i + 1] === b) || (x === b && way[i + 1] === a));
    (v.edges || []).forEach((e) => {
      const a = ctr(e.from), b = ctr(e.to);
      if (!a || !b) return;
      const l = document.createElementNS ? document.createElementNS(SVGNS, 'line') : el('line', '');
      l.setAttribute('x1', a[0]); l.setAttribute('y1', a[1]); l.setAttribute('x2', b[0]); l.setAttribute('y2', b[1]);
      l.setAttribute('class', onWay(e.from, e.to) ? 'zm-way' : e.dungeon ? 'zm-edge zm-dash' : 'zm-edge');
      svg.appendChild(l);
    });
    grid.appendChild(svg);
    legend(ns, g);
    const cur = ns.find((n) => n.id === st.sel) || ns.find((n) => (n.status || {}).here) || ns.find((n) => n.orderable) || ns[0];
    ns.forEach((n) => {
      const p = g.at(n), b = el('button', 'zm-n'), bd = band(n, level());
      b.type = 'button'; b.dataset.id = n.id; b.dataset.key = 'z-' + n.id; b.dataset.band = bd; b.dataset.kind = n.kind;
      b.style.gridColumn = String(p.c + 1); b.style.gridRow = String(p.r + 1);
      b.tabIndex = n.id === cur.id ? 0 : -1;
      b.setAttribute('aria-label', nodeLabel(n, level()));
      b.title = nodeLabel(n, level()) + (n.why ? ' - ' + n.why : '');
      mark(b, n);
      if (n.id === st.sel) b.setAttribute('aria-pressed', 'true');
      const rt = el('span', 'zm-rt', rightText(n));
      rt.dataset.r = (n.status || {}).here || (n.status || {}).queued || (n.status || {}).failed ? 'mark' : 'range';
      b.append(el('span', 'zm-nm', n.name), rt);
      const two = [n.range ? n.range.replace('clvl ', '') : '', n.why].filter(Boolean).join(' - '); // the second line, where a row has the room
      if (two) b.appendChild(el('span', 'zm-why', two));
      b.addEventListener('click', () => open(n.id));
      b.addEventListener('keydown', (e) => {
        if (!e.key || !e.key.startsWith('Arrow')) return;
        e.preventDefault();
        const to = walk(ns, n.id, e.key);
        const t = grid.querySelector ? grid.querySelector('[data-id="' + to + '"]') : null;
        if (t && t.focus) { grid.querySelectorAll('.zm-n').forEach((x) => { x.tabIndex = -1; }); t.tabIndex = 0; t.focus(); }
      });
      grid.appendChild(b);
    });
  }
  // The legend of the stripe colours, in the empty cells at the top of the grid (the colours alone say nothing): the first free block of
  // 2x2, 3x2, 1x4 or 2x3 cells, top first. A grid with no free block (a short window) keeps the same words in the title of the grid.
  const LEGEND = [['in', 'in your range'], ['above', 'above it'], ['hard', 'too hard now'], ['no', 'not open yet']];
  grid.title = 'Stripe at the left of a zone: ' + LEGEND.map((l) => l[1]).join(', ');
  function legend(ns, g) {
    const used = new Set(ns.map((n) => { const p = g.at(n); return p.c + ',' + p.r; }));
    const free = (c, r, w, h) => { for (let i = 0; i < w; i++) for (let j = 0; j < h; j++) if (c + i >= g.cols || r + j >= g.rows || used.has((c + i) + ',' + (r + j))) return false; return true; };
    for (let r = 0; r < g.rows; r++) for (let c = 0; c < g.cols; c++) for (const [w, h] of [[2, 2], [3, 2], [1, 4], [2, 3], [1, 3]]) {
      if (!free(c, r, w, h)) continue;
      const lg = el('div', 'zm-legend');
      lg.setAttribute('role', 'list');
      lg.style.gridColumn = (c + 1) + ' / span ' + w;
      lg.style.gridRow = (r + 1) + ' / span ' + h;
      lg.style.setProperty('--zm-lc', String(w > 1 && h > 1 ? 2 : 1));
      (h === 3 && w === 1 ? LEGEND.slice(0, 3) : LEGEND).forEach((l) => { const i = el('span', 'zm-lg'); i.setAttribute('role', 'listitem'); i.dataset.band = l[0]; i.append(el('i'), el('span', '', l[1])); lg.appendChild(i); });
      grid.appendChild(lg);
      return;
    }
  }

  // An act without map data: the orderable areas as a list in the same box (each opens the same Go now through a plain order).
  function drawList(v) {
    const areas = v.areas || [];
    if (!areas.length) { list.hidden = true; empty.hidden = false; empty.textContent = 'No map for this act yet. Orders go through the Go to button.'; return; }
    areas.forEach((a) => {
      const b = el('button', 'zm-n zm-row', a);
      b.type = 'button'; b.dataset.key = 'area-' + a;
      b.addEventListener('click', () => sendCommand(ctx, 'go', { area: a, until: st.dur ? { minutes: st.dur } : 'forever' }, 'now'));
      list.appendChild(b);
    });
  }

  // the second line of a zone shows only where a row has the room (about 38 px)
  function measure() {
    if (!grid.clientHeight || !grid.dataset.rows) return;
    grid.dataset.tall = grid.clientHeight / +grid.dataset.rows >= 38 ? '1' : '';
  }
  if (typeof ResizeObserver !== 'undefined') new ResizeObserver(measure).observe(grid);

  // Go to (g, the verb): focus the hero's zone and make this map the one that shows
  if (typeof document !== 'undefined' && document.addEventListener) {
    document.addEventListener('d2:goto', () => {
      document.dispatchEvent(new CustomEvent('d2:showmap', { detail: 'zone-map' })); // the shell opens this map's phone tab and switch
      const t = grid.querySelector && grid.querySelector('.zm-n[tabindex="0"]');
      if (t && t.focus) t.focus();
    });
    document.addEventListener('d2:zone-close', () => { if (!menu.hidden) close(); });
  }

  return {
    update(v, err) {
      if (!v) {
        if (err && (err.status === 404 || err.status === 503)) { ctx.hide(true); return; }
        ctx.hide(false);
        // never a blank box: before the first answer the tile says so (the legend title alone looked like a broken panel)
        if (!st.view) { grid.hidden = true; list.hidden = true; empty.hidden = false; empty.textContent = 'Waiting for the bot. The map shows as soon as it answers.'; }
        return;
      }
      ctx.hide(false);
      st.view = v;
      try { draw(); } catch (e) { // a zone that cannot be drawn must not leave the tile empty and the panel stuck loading
        grid.hidden = true; list.hidden = true; empty.hidden = false; empty.textContent = 'The map could not be drawn. Orders still work through Go to.';
        ctx.state('ready');
        return;
      }
      measure();
      if (st.deep && ns0(v)) { // a deep link: ?zone=<id> opens that zone's menu once
        st.deep = false;
        try { // ?zone=<id>[&loop=1]: the page opens with that zone's menu (a link, and the fit check's pictures)
          const q = new URLSearchParams(location.search), z = q.get('zone');
          if (z && node(z)) { open(z); if (q.get('loop') === '1' && st.view.ladder && st.view.ladder.loopOpen) { st.loop.open = true; drawMenu(); } }
        } catch (e) { /* no location */ }
      }
      if (!menu.hidden) {
        const sig = JSON.stringify([st.sel, node(st.sel), v.ladder, v.edges && v.edges.length, st.charter && st.charter.price, st.err]);
        if (sig !== st.msig) { st.msig = sig; if (!st.sel || !node(st.sel)) { close(); } else { drawMenu(); } }
        place();
      }
      ctx.state('ready');
    },
  };
}

function ns0(v) { return !!(v.nodes && v.nodes.length); }
