// Inventory page (docs/ui-design-system.md components 13 and 14): a Diablo II paper doll for the worn slots, the
// backpack, belt and stash as cell grids, a detail card for the selected item, the repair card and the equip queue.
// Reads GET /api/inventory and GET /api/repair; writes go to the answer listener. Text through textContent only; the
// only inline style writes are grid areas (CSSOM, allowed under style-src 'self').
// Game art (docs/paperdoll-art-build.md) is an optional layer: when GET /api/art says available and the panel frame
// loads, the doll and the cells get <img> icons from /watch/art/<key>.png; any failure leaves the drawn cells, which
// stay the default path. Not used with ?mode=overlay (the stream view).
(function () {
  'use strict';
  var $ = function (id) { return document.getElementById(id); };
  var sel = null, last = null;
  var SLOTS = ['head', 'neck', 'torso', 'rhand', 'lhand', 'rring', 'lring', 'belt', 'feet', 'gloves'];
  var LABEL = { head: 'helm', neck: 'amulet', torso: 'armour', rhand: 'weapon', lhand: 'shield', rring: 'ring', lring: 'ring', belt: 'belt', feet: 'boots', gloves: 'gloves' };
  var SLOTART = { head: 'helm', neck: 'amulet', torso: 'armor', rhand: 'weapon', lhand: 'weapon', rring: 'ring', lring: 'ring', belt: 'belt', feet: 'boots', gloves: 'gloves' };
  var ART_NAME = /^[a-z0-9_]{1,16}$/;
  var art = { on: false };
  var WARN_PCT = 30; // pips turn amber under this share of the maximum (the legend says so)

  function el(t, c, x) { var e = document.createElement(t); if (c) e.className = c; if (x !== undefined) e.textContent = x; return e; }
  function clear(n) { while (n.firstChild) n.removeChild(n.firstChild); }
  function say(t) { var m = $('msg'); m.textContent = t || ''; m.hidden = !t; }
  function base() {
    return fetch('/api/config', { cache: 'no-store' }).then(function (r) { return r.json(); }).then(function (c) {
      return c.answerBase ? c.answerBase : location.protocol + '//' + location.hostname + ':' + c.answerPort;
    });
  }
  function post(path, body) {
    return base().then(function (u) {
      return fetch(u + path, { method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
    }).then(function (r) { return r.json().then(function (j) { if (!r.ok) throw new Error(j.error || ('error ' + r.status)); return j; }); })
      .then(function () { say(''); load(); }, function (e) { say(e.message); });
  }
  function name(it) { return (it.name || it.code) + (it.ethereal ? ' (eth)' : '') + (it.identified ? '' : ' (unidentified)'); }
  // durability: /api/inventory leaves "dur" out when it is 0 (omitempty), so a broken item has maxDur and no dur.
  function dur(it) { return it.maxDur ? { cur: it.dur || 0, max: it.maxDur } : null; }
  function level(d) { if (!d) return ''; if (d.cur <= 0) return 'bad'; return 100 * d.cur / d.max < WARN_PCT ? 'warn' : 'ok'; }
  function lines(it) {
    var o = [], d = dur(it);
    if (d) o.push('durability ' + d.cur + '/' + d.max + (d.cur <= 0 ? ' (BROKEN)' : ''));
    if (it.level) o.push('item level ' + it.level);
    if (it.sockets) o.push(it.sockets + ' sockets');
    (it.props || []).forEach(function (p) { o.push(p); });
    return o;
  }
  // artImg adds a game-art <img> to host; when it fails to load the image goes away and the host reverts to its drawn look.
  function artImg(host, cls, key) {
    var i = el('img', cls); i.setAttribute('alt', ''); i.setAttribute('src', '/watch/art/' + key + '.png');
    host.dataset.shown = '';
    i.addEventListener('error', function () { host.removeChild(i); delete host.dataset.shown; });
    host.appendChild(i);
  }
  function qual(e, it) { if (it.quality) e.classList.add('q-' + it.quality); }
  function button(text, fn, disabled, opts) {
    var b = el('button', 'd-btn', text); b.type = 'button'; b.dataset.size = 'sm';
    if (opts) Object.keys(opts).forEach(function (k) { b.dataset[k] = opts[k]; });
    if (disabled) b.disabled = true;
    b.addEventListener('click', fn);
    return b;
  }

  // ---- paper doll ----
  function doll(v) {
    var box = $('equip'); clear(box);
    if (art.on) { box.dataset.art = ''; var f = el('img', 'art-panel'); f.setAttribute('alt', ''); f.setAttribute('src', '/watch/art/panel-inv.png'); box.appendChild(f); } else delete box.dataset.art;
    var pins = {}; (v.pins || []).forEach(function (p) { pins[p.slot] = p; });
    var broken = 0, worn = 0;
    SLOTS.forEach(function (s) {
      var it = v.equipment && v.equipment[s];
      var b = el('button', 'd-slot2'); b.type = 'button'; b.dataset.slot = s; b.dataset.key = 'slot-' + s;
      // the icon goes first so the durability pips, PIN and BROKEN stay above it
      if (art.on) {
        if (it && it.art && ART_NAME.test(it.art)) artImg(b, 'ico', 'item-' + it.art);
        else if (!it) artImg(b, 'ghost', 'slot-' + SLOTART[s]);
      }
      b.appendChild(el('span', 'sil', LABEL[s]));
      if (it) {
        b.dataset.filled = ''; qual(b, it);
        b.appendChild(el('span', 'nm', name(it)));
        var d = dur(it);
        if (d) {
          b.dataset.level = level(d);
          if (d.cur <= 0) broken++; else if (b.dataset.level === 'warn') worn++;
          var row = el('span', 'dur'), on = d.cur <= 0 ? 0 : Math.max(1, Math.ceil(6 * d.cur / d.max));
          for (var i = 0; i < 6; i++) row.appendChild(el('i', i < on ? 'on' : ''));
          row.appendChild(el('span', 'num', d.cur + '/' + d.max));
          b.appendChild(row);
          if (d.cur <= 0) b.appendChild(el('span', 'broken', 'Broken'));
        }
        b.title = name(it) + (d ? ', durability ' + d.cur + '/' + d.max : '');
      } else b.title = LABEL[s] + ': empty';
      if (pins[s]) b.appendChild(el('span', 'pin', 'PIN'));
      if (sel && sel.slot === s) b.classList.add('sel');
      b.setAttribute('aria-label', LABEL[s] + ': ' + (it ? name(it) : 'empty') + (pins[s] ? ', pinned' : ''));
      b.addEventListener('click', function () { sel = { slot: s }; draw(last); });
      box.appendChild(b);
    });
    var bits = [];
    if (broken) bits.push(broken + ' broken');
    if (worn) bits.push(worn + ' worn');
    $('equipmeta').textContent = bits.join(', ');
  }

  // ---- cell grids ----
  function cells(id, items, cols, rows) {
    var box = $(id); clear(box); box.dataset.cols = String(cols);
    for (var y = 0; y < rows; y++) for (var x = 0; x < cols; x++) {
      var c = el('div', 'cell'); c.style.gridArea = (y + 1) + ' / ' + (x + 1); box.appendChild(c);
    }
    items.forEach(function (it) {
      var w = Math.max(it.w || 1, 1), h = Math.max(it.h || 1, 1);
      var b = el('button', 'item'); b.type = 'button'; qual(b, it); b.dataset.key = 'item-' + it.id; b.dataset.h = String(h);
      b.style.gridArea = (it.y + 1) + ' / ' + (it.x + 1) + ' / span ' + h + ' / span ' + w;
      b.textContent = name(it); b.title = name(it);
      if (art.on && it.art && ART_NAME.test(it.art)) artImg(b, 'ico', 'item-' + it.art);
      if (it.mark) { b.dataset.mark = it.mark; b.appendChild(el('span', 'mk', it.mark === 'sell' ? 'S' : 'K')); }
      if (sel && sel.id === it.id) b.classList.add('sel');
      b.addEventListener('click', function () { sel = { id: it.id }; draw(last); });
      box.appendChild(b);
    });
  }

  // ---- detail card of the selected slot or item ----
  function detail(v) {
    var d = $('detail'); clear(d);
    if (!sel) { d.appendChild(el('p', 'd-hint', 'Pick a slot or an item to see it here.')); return; }
    var it = null, slot = sel.slot || '';
    if (slot) it = v.equipment && v.equipment[slot];
    else [v.backpack, v.stash, v.belt].forEach(function (l) { (l || []).forEach(function (x) { if (x.id === sel.id) it = x; }); });
    var pinned = slot && (v.pins || []).some(function (p) { return p.slot === slot; });
    if (!it) {
      d.appendChild(el('h3', '', slot ? LABEL[slot] + ': empty' : 'Gone'));
      if (!slot) sel = null;
    } else {
      var h = el('h3', '', name(it)); qual(h, it); d.appendChild(h);
      lines(it).forEach(function (l) { d.appendChild(el('p', 'd-muted', l)); });
      if (it.why) d.appendChild(el('p', '', 'Why: ' + it.why));
      if (it.advice) d.appendChild(el('p', '', it.advice));
    }
    var bar = el('div', 'd-btngroup');
    if (pinned) {
      d.appendChild(el('p', 'd-muted', 'Pinned by you: the bot leaves this slot alone.'));
      bar.appendChild(button('Unpin', function () { post('/api/inventory/unpin', { slot: slot }); }, false, { key: 'unpin' }));
    }
    if (it && !slot) {
      var inPack = (v.backpack || []).some(function (x) { return x.id === it.id; });
      bar.appendChild(button(it.mark === 'sell' ? 'Clear sell mark' : 'Mark for sale', function () {
        post('/api/inventory/mark', { id: it.id, mark: it.mark === 'sell' ? '' : 'sell' });
      }, !inPack || !it.canSell && it.mark !== 'sell', { key: 'sell' }));
      bar.appendChild(button(it.mark === 'keep' ? 'Clear keep mark' : 'Keep', function () {
        post('/api/inventory/mark', { id: it.id, mark: it.mark === 'keep' ? '' : 'keep' });
      }, false, { key: 'keep' }));
      if (inPack) {
        d.appendChild(bar); bar = el('div', 'd-btngroup');
        d.appendChild(el('p', 'd-muted', 'Equip in:'));
        SLOTS.forEach(function (s) {
          var ok = (it.canWear || []).indexOf(s) >= 0;
          var b = button(LABEL[s] + (s === 'lring' ? ' (left)' : s === 'rring' ? ' (right)' : ''), function () { post('/api/inventory/equip', { id: it.id, slot: s }); }, !ok, { key: 'wear-' + s });
          b.title = ok ? 'queue this equip' : 'not possible right now';
          bar.appendChild(b);
        });
        if (!v.inTown) d.appendChild(el('p', 'd-muted', 'The hero is not in town: equip works in town only.'));
      }
    }
    if (bar.firstChild) d.appendChild(bar);
  }

  // the page redraws every 4 s; the focused slot, item or button is found again by its data-key
  var drawn = '';
  function draw(v) {
    var sig = JSON.stringify(v); if (sig === drawn) return; // unchanged: leave the DOM, focus and scroll alone
    drawn = sig;
    var a = document.activeElement, key = a && a.dataset ? a.dataset.key : '';
    paint(v);
    if (key && document.activeElement !== a) { var n = document.querySelector('[data-key="' + key + '"]'); if (n) n.focus({ preventScroll: true }); }
  }
  function paint(v) {
    last = v;
    doll(v);
    cells('pack', v.backpack || [], 10, 4);
    cells('belt', v.belt || [], 4, 1);
    cells('stash', v.stash || [], 6, 8);
    $('packmeta').textContent = '10 by 4';
    $('stashmeta').textContent = (v.stashGold || v.stashGold === 0 ? 'gold ' + v.stashGold : '');
    $('note').textContent = (v.note || '') + (v.class ? v.class + ', gold on the hero ' + v.gold : '');
    detail(v);
    var q = $('queue'); clear(q);
    var rows = (v.queue || []).concat(v.recent || []);
    if (!rows.length) { q.appendChild(el('p', 'd-hint', 'Nothing queued.')); return; }
    var ul = el('ul', 'd-list');
    rows.forEach(function (r) {
      var li = el('li');
      li.appendChild(el('span', 't', r.state));
      li.appendChild(el('span', 'txt', r.name + ' in ' + (LABEL[r.slot] || r.slot)));
      if (r.reason) li.appendChild(el('span', 'sub', r.reason));
      ul.appendChild(li);
    });
    q.appendChild(ul);
  }

  function drawRepair(v) {
    var b = $('repair'); clear(b);
    var q = v.request, pend = q && q.state === 'requested';
    var mode = (v.modes || []).filter(function (m) { return m.current; }).map(function (m) { return m.label; })[0] || v.mode;
    b.appendChild(el('p', 'd-muted', 'Repair mode: ' + mode + ' (change it on the policy sheet)'));
    if (v.worst) { var w = el('p', 'd-note', v.worst); w.dataset.sev = v.worst.indexOf('Broken') === 0 || /broken/i.test(v.worst) ? 'alarm' : 'warn'; b.appendChild(w); }
    if (v.estimate) b.appendChild(el('p', '', 'Repair All costs at least ' + v.estimate + ' gold.'));
    if (q) b.appendChild(el('p', 'd-muted', q.state === 'requested' ? 'Requested: the hero repairs at its next town visit.' + (q.note ? ' (' + q.note + ')' : '') : q.state === 'done' ? 'Last request: done.' : 'Last request: ' + q.state));
    var bar = el('div', 'd-btngroup');
    bar.appendChild(button(pend ? 'Cancel repair' : 'Repair now', function () { post(pend ? '/api/repair/cancel' : '/api/repair/request', {}); }, false, { key: 'repair', variant: pend ? 'ghost' : 'primary' }));
    b.appendChild(bar);
  }

  function getJSON(path) {
    return fetch(path, { cache: 'no-store' }).then(function (r) {
      return r.json().then(function (j) { if (!r.ok) throw new Error(j.error || 'error'); return j; });
    });
  }
  // artProbe: art is on only when the server says available and the panel frame really loads (a frame that does not
  // load would leave the slot layout without its background), and never in the stream overlay.
  function artProbe() {
    if (/(^|[?&])mode=overlay(&|$)/.test(location.search)) return Promise.resolve(false);
    return getJSON('/api/art').then(function (a) {
      if (!a || !a.available) return false;
      return new Promise(function (ok) {
        var i = new Image(); i.onload = function () { ok(true); }; i.onerror = function () { ok(false); }; i.src = '/watch/art/panel-inv.png';
      });
    }).catch(function () { return false; });
  }
  function load() {
    getJSON('/api/repair').then(drawRepair, function (e) { $('repair').textContent = e.message; });
    getJSON('/api/inventory').then(draw, function (e) { say(e.message); });
  }
  if (typeof window !== 'undefined' && window.__invTest) {
    window.__invTest({ dur: dur, level: level, lines: lines, paint: paint, say: say, setArt: function (on) { art.on = !!on; } });
    return;
  }
  artProbe().then(function (on) { art.on = on; load(); setInterval(load, 4000); });
})();
