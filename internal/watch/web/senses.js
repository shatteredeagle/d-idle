// Side-panel minimap (docs/senses-map.md). View only: polls GET /api/senses about twice a second, caches the map by
// version, draws it top-down on a canvas. All text via textContent; no inline code (CSP).
(function () {
  'use strict';
  var panel = document.getElementById('sensespanel'), cv = document.getElementById('sensescanvas');
  if (!panel || !cv) return;
  var ctx = cv.getContext('2d'), map = null, cells = null, hero = null, units = [], version = 0;
  var nbs = [], nbCache = {}; // neighbour areas (docs/senses-map.md "Contiguous areas"): {areaId, version, dx, dy, map, cells}
  var zoom = 0, panX = 0, panY = 0, fails = 0, timer = 0, drag = null;

  function decode(m) {
    var n = m.w * m.h, out = new Uint8Array(n), o = 0, r = m.rle || [];
    for (var i = 0; i + 1 < r.length; i += 2) { for (var c = 0; c < r[i + 1] && o < n; c++) out[o++] = r[i]; }
    return out;
  }
  function css(name, fb) { return getComputedStyle(document.documentElement).getPropertyValue(name).trim() || fb; }
  // The world box in subtiles of the current area's frame: the current map plus every neighbour moved by its dx, dy.
  function union(cur, list) {
    var b = { x0: cur.x0, y0: cur.y0, x1: cur.x0 + cur.w * cur.step, y1: cur.y0 + cur.h * cur.step };
    list.forEach(function (n) {
      var m = n.map; b.x0 = Math.min(b.x0, m.x0 + n.dx); b.y0 = Math.min(b.y0, m.y0 + n.dy);
      b.x1 = Math.max(b.x1, m.x0 + n.dx + m.w * m.step); b.y1 = Math.max(b.y1, m.y0 + n.dy + m.h * m.step);
    });
    return b;
  }
  // The nb= query value: the neighbour versions this page already holds.
  function nbParam(cache) {
    return Object.keys(cache).map(function (k) { return k + ':' + cache[k].version; }).join(',');
  }
  // Base scale fits the whole stitched world; zoom multiplies it. The view centres on the hero plus the pan offset.
  // k is pixels per subtile.
  function view() {
    var u = union(map, nbs), k0 = Math.min(cv.width / (u.x1 - u.x0), cv.height / (u.y1 - u.y0)), k = k0 * Math.pow(1.5, zoom);
    var hx = hero ? hero.x : (u.x0 + u.x1) / 2, hy = hero ? hero.y : (u.y0 + u.y1) / 2;
    if (zoom === 0) { hx = (u.x0 + u.x1) / 2; hy = (u.y0 + u.y1) / 2; }
    return { k: k, ox: cv.width / 2 - (hx + panX) * k, oy: cv.height / 2 - (hy + panY) * k };
  }
  function drawArea(m, cl, dx, dy, v, current) {
    var x, y;
    for (y = 0; y < m.h; y++) for (x = 0; x < m.w; x++) {
      var c = cl[y * m.w + x]; if (!c) continue;
      ctx.fillStyle = c === 1 ? css('--surface', '#fffdf8') : css('--line', '#999');
      ctx.fillRect(v.ox + (m.x0 + dx + x * m.step) * v.k, v.oy + (m.y0 + dy + y * m.step) * v.k, Math.ceil(m.step * v.k), Math.ceil(m.step * v.k));
    }
    function at(p) { return [v.ox + (p.x + dx + 0.5) * v.k, v.oy + (p.y + dy + 0.5) * v.k]; }
    var e = m.exits || [], w = m.waypoints || [], i;
    ctx.fillStyle = css('--info', '#2a63a8');
    for (i = 0; i < e.length; i++) { var a = at(e[i]); ctx.fillRect(a[0] - 4, a[1] - 4, 8, 8); }
    ctx.fillStyle = css('--accent', '#b5760f');
    for (i = 0; i < w.length; i++) { var b = at(w[i]); ctx.beginPath(); ctx.moveTo(b[0], b[1] - 6); ctx.lineTo(b[0] + 5, b[1]); ctx.lineTo(b[0], b[1] + 6); ctx.lineTo(b[0] - 5, b[1]); ctx.closePath(); ctx.fill(); }
  }
  function draw() {
    ctx.clearRect(0, 0, cv.width, cv.height);
    if (!map || !cells) return;
    var v = view();
    ctx.globalAlpha = 0.55; // neighbours sit behind the current area, dimmer
    nbs.forEach(function (n) { drawArea(n.map, n.cells, n.dx, n.dy, v, false); });
    ctx.globalAlpha = 1;
    drawArea(map, cells, 0, 0, v, true);
    function at(p) { return [v.ox + (p.x + 0.5) * v.k, v.oy + (p.y + 0.5) * v.k]; }
    if (hero) {
      var h = at(hero); ctx.strokeStyle = css('--bad', '#b3342a'); ctx.lineWidth = 3;
      ctx.beginPath(); ctx.moveTo(h[0] - 7, h[1]); ctx.lineTo(h[0] + 7, h[1]); ctx.moveTo(h[0], h[1] - 7); ctx.lineTo(h[0], h[1] + 7); ctx.stroke();
    }
    if (window.sensesUnitsDraw && units.length) window.sensesUnitsDraw(ctx, at, units, css, cv);
  }
  // Neighbours arrive with their map only when the page lacks that version; the cache is by area id.
  function takeNeighbours(list) {
    var keep = {};
    nbs = [];
    (list || []).forEach(function (n) {
      var c = nbCache[n.areaId];
      if (n.map) { c = nbCache[n.areaId] = { version: n.version, map: n.map, cells: decode(n.map) }; }
      if (!c || c.version !== n.version) return; // cannot draw it; the next poll asks again
      keep[n.areaId] = true;
      nbs.push({ areaId: n.areaId, dx: n.dx, dy: n.dy, map: c.map, cells: c.cells });
    });
    Object.keys(nbCache).forEach(function (k) { if (!keep[k]) delete nbCache[k]; });
  }
  function apply(r) {
    if (r.map) { map = r.map; cells = decode(map); version = r.version; panX = panY = 0; document.getElementById('sensesarea').textContent = map.area || ('area ' + map.areaId); }
    hero = r.hero || hero;
    takeNeighbours(r.neighbours);
    units = r.units || [];
    var note = document.getElementById('sensesnote');
    note.textContent = r.waiting ? 'Waiting for the first area...' : 'Legend: cross hero, diamond waypoint, square exit' + (nbs.length ? '; dimmer areas are the ones next door' : '') + '. Fog clears as the hero walks.';
    draw();
  }
  function poll() {
    fetch('/api/senses?v=' + version + (Object.keys(nbCache).length ? '&nb=' + encodeURIComponent(nbParam(nbCache)) : ''), { credentials: 'same-origin' }).then(function (res) {
      if (res.status === 404 || res.status === 401) throw new Error('off');
      return res.json();
    }).then(function (r) {
      fails = 0;
      if (!r.tiers_unlocked || r.tiers_unlocked.indexOf(1) < 0) { panel.hidden = true; return; }
      panel.hidden = false; apply(r);
    }).catch(function () { fails++; }).then(function () {
      if (fails < 5) timer = setTimeout(poll, fails ? 4000 : 500);
    });
  }
  document.getElementById('senses-zin').addEventListener('click', function () { zoom = Math.min(zoom + 1, 6); draw(); });
  document.getElementById('senses-zout').addEventListener('click', function () { zoom = Math.max(zoom - 1, 0); if (!zoom) panX = panY = 0; draw(); });
  var hide = document.getElementById('senses-hide');
  hide.addEventListener('click', function () {
    var min = panel.classList.toggle('min'); hide.textContent = min ? 'Show' : 'Hide'; hide.setAttribute('aria-pressed', String(!min));
  });
  cv.addEventListener('pointerdown', function (ev) { drag = { x: ev.clientX, y: ev.clientY }; cv.setPointerCapture(ev.pointerId); });
  cv.addEventListener('pointerup', function () { drag = null; });
  cv.addEventListener('pointermove', function (ev) {
    if (!drag || !map || !zoom) return;
    var q = cv.width / cv.getBoundingClientRect().width, k = view().k;
    panX -= (ev.clientX - drag.x) * q / k; panY -= (ev.clientY - drag.y) * q / k; drag = { x: ev.clientX, y: ev.clientY }; draw();
  });
  window.sensesPanel = { union: union, nbParam: nbParam };
  poll();
}());
