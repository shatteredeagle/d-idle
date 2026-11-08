// Side-panel minimap (docs/senses-map.md). View only: polls GET /api/senses about twice a second, caches the map by
// version, draws it top-down on a canvas. All text via textContent; no inline code (CSP).
(function () {
  'use strict';
  var panel = document.getElementById('sensespanel'), cv = document.getElementById('sensescanvas');
  if (!panel || !cv) return;
  var ctx = cv.getContext('2d'), map = null, cells = null, hero = null, version = 0;
  var zoom = 0, panX = 0, panY = 0, fails = 0, timer = 0, drag = null;

  function decode(m) {
    var n = m.w * m.h, out = new Uint8Array(n), o = 0, r = m.rle || [];
    for (var i = 0; i + 1 < r.length; i += 2) { for (var c = 0; c < r[i + 1] && o < n; c++) out[o++] = r[i]; }
    return out;
  }
  function css(name, fb) { return getComputedStyle(document.documentElement).getPropertyValue(name).trim() || fb; }
  // Base scale fits the whole map; zoom multiplies it. The view centres on the hero plus the pan offset.
  function view() {
    var base = Math.min(cv.width / map.w, cv.height / map.h), s = base * Math.pow(1.5, zoom);
    var hx = hero ? (hero.x - map.x0) / map.step : map.w / 2, hy = hero ? (hero.y - map.y0) / map.step : map.h / 2;
    if (zoom === 0) { hx = map.w / 2; hy = map.h / 2; }
    return { s: s, ox: cv.width / 2 - (hx + panX) * s, oy: cv.height / 2 - (hy + panY) * s };
  }
  function draw() {
    ctx.clearRect(0, 0, cv.width, cv.height);
    if (!map || !cells) return;
    var v = view(), s = v.s, i, x, y;
    ctx.fillStyle = css('--surface', '#fff');
    for (y = 0; y < map.h; y++) for (x = 0; x < map.w; x++) {
      var c = cells[y * map.w + x]; if (!c) continue;
      ctx.fillStyle = c === 1 ? css('--surface', '#fffdf8') : css('--line', '#999');
      ctx.fillRect(v.ox + x * s, v.oy + y * s, Math.ceil(s), Math.ceil(s));
    }
    function at(p) { return [v.ox + ((p.x - map.x0) / map.step + 0.5) * s, v.oy + ((p.y - map.y0) / map.step + 0.5) * s]; }
    var e = map.exits || [], w = map.waypoints || [];
    ctx.fillStyle = css('--info', '#2a63a8');
    for (i = 0; i < e.length; i++) { var a = at(e[i]); ctx.fillRect(a[0] - 4, a[1] - 4, 8, 8); }
    ctx.fillStyle = css('--accent', '#b5760f');
    for (i = 0; i < w.length; i++) { var b = at(w[i]); ctx.beginPath(); ctx.moveTo(b[0], b[1] - 6); ctx.lineTo(b[0] + 5, b[1]); ctx.lineTo(b[0], b[1] + 6); ctx.lineTo(b[0] - 5, b[1]); ctx.closePath(); ctx.fill(); }
    if (hero) {
      var h = at(hero); ctx.strokeStyle = css('--bad', '#b3342a'); ctx.lineWidth = 3;
      ctx.beginPath(); ctx.moveTo(h[0] - 7, h[1]); ctx.lineTo(h[0] + 7, h[1]); ctx.moveTo(h[0], h[1] - 7); ctx.lineTo(h[0], h[1] + 7); ctx.stroke();
    }
  }
  function apply(r) {
    if (r.map) { map = r.map; cells = decode(map); version = r.version; panX = panY = 0; document.getElementById('sensesarea').textContent = map.area || ('area ' + map.areaId); }
    hero = r.hero || hero;
    var note = document.getElementById('sensesnote');
    note.textContent = r.waiting ? 'Waiting for the first area...' : 'Legend: cross hero, diamond waypoint, square exit. Fog clears as the hero walks.';
    draw();
  }
  function poll() {
    fetch('/api/senses?v=' + version, { credentials: 'same-origin' }).then(function (res) {
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
    var k = cv.width / cv.getBoundingClientRect().width, s = view().s;
    panX -= (ev.clientX - drag.x) * k / s; panY -= (ev.clientY - drag.y) * k / s; drag = { x: ev.clientX, y: ev.clientY }; draw();
  });
  poll();
}());
