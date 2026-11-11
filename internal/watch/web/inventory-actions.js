// Item action menu of the inventory page (docs/inventory-actions.md, docs/ui-design-system.md components 13 and 14):
// click, tap or press Enter on an item (backpack, stash, belt, a worn slot) and a small menu offers Equip, Mark for
// sale, Keep, Trash and Unpin. A choice QUEUES an action the bot performs at its next open opportunity; this file never
// touches the game. A phone gets the same menu as a bottom sheet. The queue card is redrawn here with the states
// queued, doing, done, failed and cancelled, a reason line and a Cancel button for a queued request.
// Input: the view /api/inventory sends, handed over by inventory.js as a 'd2:inventory' event on document (the same
// feed the item tooltip uses). Text goes in through textContent only; nothing scrolls the page (focus is always taken
// with preventScroll) and the menu is position: fixed.
// EXTENSION POINT (a later pass, not built): drag an item onto a doll slot. Every doll slot button carries data-slot
// and data-key="slot-<slot>"; a drop handler would call window.d2InvActions.equip(itemId, slot) below.
(function () {
  'use strict';
  var MENU_ID = 'inv-act';
  var LABEL = { head: 'helm', neck: 'amulet', torso: 'armour', rhand: 'right hand', lhand: 'left hand', rring: 'right ring', lring: 'left ring', belt: 'belt', feet: 'boots', gloves: 'gloves' };
  var view = null, menu = null, scrim = null, openKey = '', openSig = '', confirming = false, kbd = false, sheet = false;

  function el(t, c, x) { var e = document.createElement(t); if (c) e.className = c; if (x !== undefined) e.textContent = x; return e; }
  function clear(n) { while (n.firstChild) n.removeChild(n.firstChild); }
  function say(t) { var m = document.getElementById('msg'); if (!m) return; m.textContent = t || ''; m.hidden = !t; }
  function slotLabel(s) { return LABEL[s] || s; }

  // ---- data ----
  function find(key) {
    if (!view || !key) return null;
    var m = /^(item|slot)-(.+)$/.exec(key);
    if (!m) return null;
    if (m[1] === 'slot') return { it: view.equipment && view.equipment[m[2]], slot: m[2], place: 'worn' };
    var id = Number(m[2]), hit = null;
    [['pack', view.backpack], ['stash', view.stash], ['belt', view.belt], ['cube', view.cube]].forEach(function (l) {
      (l[1] || []).forEach(function (x) { if (x.id === id) hit = { it: x, slot: '', place: l[0] }; });
    });
    return hit;
  }
  function requestsFor(id) { return (view && view.queue || []).filter(function (r) { return r.id === id; }); }
  function kindOf(r) { return r.kind || 'equip'; }
  function verb(r) { return kindOf(r) === 'trash' ? 'Trash ' + r.name : 'Equip ' + r.name + ' in ' + slotLabel(r.slot); }

  // model builds the menu of one item: {title, cls, rows: [{text, state, cancel}], entries: [{act, label, reason, slot}]}.
  // An entry with a reason is shown but not available. The server checks everything again.
  function model(f) {
    var it = f.it, v = view, pinned = f.slot && (v.pins || []).some(function (p) { return p.slot === f.slot; });
    var reqs = requestsFor(it.id), trashQ = reqs.some(function (r) { return kindOf(r) === 'trash'; });
    var m = { target: 'hero', /* the menu structure is per target: the mercenary stage adds its own entries */ title: it.name || it.code || 'item', cls: 'q-' + (it.quality || 'normal'), id: it.id, rows: [], entries: [] };
    reqs.forEach(function (r) { m.rows.push({ text: verb(r), state: r.state, kind: kindOf(r), cancel: r.state === 'queued' }); });
    // equip
    if (f.place === 'pack') {
      var can = it.canWear || [];
      if (!can.length) m.entries.push({ act: 'equip', label: 'Equip', reason: it.noWear || 'it cannot be worn right now' });
      else can.forEach(function (s) {
        m.entries.push({ act: 'equip', slot: s, label: can.length > 1 ? 'Equip (' + slotLabel(s) + ')' : 'Equip', reason: trashQ ? 'it is queued to be trashed: cancel that first' : '' });
      });
    } else {
      m.entries.push({ act: 'equip', label: 'Equip', reason: f.place === 'worn' ? 'it is already worn' : (it.noWear || 'move it to the backpack first') });
    }
    // sell and keep marks
    var sold = it.mark === 'sell';
    m.entries.push({ act: 'sell', label: sold ? 'Clear sell mark' : 'Mark for sale', reason: sold ? '' : (it.noSell || (f.place !== 'pack' ? 'only backpack items are sold' : '')) });
    m.entries.push({ act: 'keep', label: it.mark === 'keep' ? 'Clear keep mark' : 'Keep', reason: '' });
    // trash, with its confirm step
    var tr = { act: 'trash', label: confirming ? 'Confirm: trash it' : 'Trash', reason: it.noTrash || (trashQ ? 'it is already queued to be trashed' : '') };
    if (confirming && !tr.reason) tr.confirm = true;
    m.entries.push(tr);
    if (pinned) m.entries.push({ act: 'unpin', slot: f.slot, label: 'Unpin slot', reason: '' });
    return m;
  }

  // ---- network (same gate as the other writes: the answer listener, cookie or token) ----
  function base() {
    return fetch('/api/config', { cache: 'no-store' }).then(function (r) { return r.json(); }).then(function (c) {
      return c.answerBase ? c.answerBase : location.protocol + '//' + location.hostname + ':' + c.answerPort;
    });
  }
  function post(path, body) {
    return base().then(function (u) {
      return fetch(u + path, { method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
    }).then(function (r) { return r.json().then(function (j) { if (!r.ok) throw new Error(j.error || ('error ' + r.status)); return j; }); })
      .then(function () { say(''); return refresh(); }, function (e) { say(e.message); setNote(e.message); });
  }
  function refresh() {
    return fetch('/api/inventory', { cache: 'no-store' }).then(function (r) { return r.json(); }).then(onView, function () {});
  }
  // The public calls, also the hook of the later drag and drop pass.
  function equip(id, slot) { return post('/api/inventory/equip', { id: id, slot: slot || '' }); }
  function trash(id) { return post('/api/inventory/trash', { id: id, confirm: true }); }
  function cancel(id, kind) { return post('/api/inventory/cancel', { id: id, kind: kind || 'equip' }); }

  function run(e, m) {
    if (e.reason) { setNote(e.reason); return; }
    if (e.act !== 'trash') confirming = false;
    switch (e.act) {
      case 'equip': closeMenu(true); equip(m.id, e.slot); return;
      case 'sell': { var it = find(openKey).it; closeMenu(true); post('/api/inventory/mark', { id: m.id, mark: it.mark === 'sell' ? '' : 'sell' }); return; }
      case 'keep': { var k = find(openKey).it; closeMenu(true); post('/api/inventory/mark', { id: m.id, mark: k.mark === 'keep' ? '' : 'keep' }); return; }
      case 'unpin': closeMenu(true); post('/api/inventory/unpin', { slot: e.slot }); return;
      case 'trash':
        if (!e.confirm) { confirming = true; draw(true); return; } // the first click only asks
        closeMenu(true); trash(m.id); return;
    }
  }

  // ---- menu DOM ----
  function setNote(t) { if (menu) { var n = menu.querySelector('.act-note'); if (n) n.textContent = t || ''; } }
  function ensure() {
    if (menu) return menu;
    menu = el('div', 'inv-act'); menu.id = MENU_ID; menu.setAttribute('role', 'menu'); menu.hidden = true;
    scrim = el('div', 'inv-act-scrim'); scrim.hidden = true;
    scrim.addEventListener('click', function () { closeMenu(true); });
    document.body.appendChild(scrim); document.body.appendChild(menu);
    return menu;
  }
  function render(m) {
    var n = ensure(); clear(n);
    n.setAttribute('aria-label', 'Actions for ' + m.title);
    n.appendChild(el('div', 'act-title ' + m.cls, m.title));
    m.rows.forEach(function (r) {
      var row = el('div', 'act-state'); row.dataset.state = r.state; row.dataset.kind = r.kind;
      row.appendChild(el('span', 'act-st', r.state)); row.appendChild(el('span', 'act-tx', r.text));
      if (r.cancel) {
        var c = el('button', 'act act-cancel', 'Cancel'); c.type = 'button'; c.dataset.act = 'cancel'; c.setAttribute('role', 'menuitem');
        c.addEventListener('click', function () { closeMenu(true); cancel(m.id, r.kind); });
        row.appendChild(c);
      }
      n.appendChild(row);
    });
    m.entries.forEach(function (e) {
      var b = el('button', 'act', e.label); b.type = 'button'; b.dataset.act = e.act + (e.slot ? '-' + e.slot : '');
      b.setAttribute('role', 'menuitem');
      if (e.act === 'trash') b.dataset.danger = '';
      if (e.confirm) b.dataset.confirm = '';
      if (e.reason) { b.setAttribute('aria-disabled', 'true'); b.dataset.off = ''; }
      b.addEventListener('click', function () { run(e, m); });
      n.appendChild(b);
      if (e.reason) { var w = el('div', 'act-why', e.reason); w.dataset.for = b.dataset.act; n.appendChild(w); }
    });
    var note = el('div', 'act-note'); note.setAttribute('role', 'status'); n.appendChild(note);
    if (confirming) { var no = el('button', 'act', 'Keep it'); no.type = 'button'; no.dataset.act = 'back'; no.setAttribute('role', 'menuitem'); no.addEventListener('click', function () { confirming = false; draw(true); }); n.appendChild(no); }
  }
  function place(a, w, h, vw, vh) { // beside the anchor, else below or above; always inside the viewport
    var mg = 8, x = a.right + mg + w <= vw - mg ? a.right + mg : a.left - mg - w >= mg ? a.left - mg - w : Math.round((a.left + a.right) / 2 - w / 2), y = a.top;
    if (x !== a.right + mg && x !== a.left - mg - w) y = a.bottom + mg + h <= vh - mg ? a.bottom + mg : a.top - mg - h;
    return { x: Math.max(mg, Math.min(Math.round(x), vw - w - mg)), y: Math.max(mg, Math.min(Math.round(y), vh - h - mg)) };
  }
  function anchorOf(key) { return document.querySelector('[data-key="' + key + '"]'); }
  function position() {
    if (sheet) { menu.style.transform = ''; return; }
    var a = anchorOf(openKey); if (!a) { closeMenu(false); return; }
    var p = place(a.getBoundingClientRect(), menu.offsetWidth, menu.offsetHeight, document.documentElement.clientWidth, document.documentElement.clientHeight);
    menu.style.transform = 'translate(' + p.x + 'px, ' + p.y + 'px)';
  }
  function focusables() { return menu ? menu.querySelectorAll('button.act') : []; }
  function draw(keepFocus) {
    var f = find(openKey);
    if (!f || !f.it) { closeMenu(false); return; }
    var m = model(f), sig = JSON.stringify(m) + (confirming ? 'c' : '');
    if (sig === openSig && !menu.hidden) return;
    var act = keepFocus && document.activeElement && document.activeElement.dataset ? document.activeElement.dataset.act : '';
    render(m); openSig = sig;
    menu.dataset.mode = sheet ? 'sheet' : 'pop'; menu.hidden = false; scrim.hidden = !sheet;
    var list = focusables(), target = null;
    for (var i = 0; i < list.length; i++) if (act && list[i].dataset.act === act) target = list[i];
    if (keepFocus && !target) for (var j = 0; j < list.length; j++) if (!list[j].hasAttribute('aria-disabled')) { target = list[j]; break; }
    if (target) target.focus({ preventScroll: true });
    position();
  }
  function openMenu(key) {
    var f = find(key);
    if (!f || !f.it) return;
    ensure();
    confirming = false; openKey = key; openSig = '';
    if (window.matchMedia) sheet = window.matchMedia('(pointer: coarse)').matches || window.matchMedia('(max-width: 40rem)').matches;
    menu.hidden = true;
    var a = anchorOf(key); if (a) a.setAttribute('aria-haspopup', 'menu');
    draw(false);
    var list = focusables(); // the first available entry takes focus so the keyboard works at once
    for (var i = 0; i < list.length; i++) if (!list[i].hasAttribute('aria-disabled')) { list[i].focus({ preventScroll: true }); break; }
  }
  function closeMenu(refocus) {
    var key = openKey;
    openKey = ''; openSig = ''; confirming = false;
    if (menu) menu.hidden = true;
    if (scrim) scrim.hidden = true;
    if (refocus && key) { var a = anchorOf(key); if (a && a.focus) a.focus({ preventScroll: true }); }
  }

  // ---- queue card ----
  function renderQueue() {
    var q = document.getElementById('queue'); if (!q || !view) return;
    var t = document.getElementById('t-queue'); if (t) t.textContent = 'Item actions';
    clear(q);
    var rows = (view.queue || []).concat(view.recent || []);
    if (!rows.length) { q.appendChild(el('p', 'd-hint', 'Nothing queued. Click an item to equip, sell or trash it.')); return; }
    var ul = el('ul', 'd-list');
    rows.forEach(function (r) {
      var li = el('li'); li.dataset.state = r.state;
      li.appendChild(el('span', 't', r.state));
      li.appendChild(el('span', 'txt', verb(r)));
      if (r.reason) li.appendChild(el('span', 'sub', r.reason));
      if (r.state === 'queued') {
        var c = el('button', 'act act-cancel', 'Cancel'); c.type = 'button'; c.dataset.key = 'cancel-' + kindOf(r) + '-' + r.id;
        c.addEventListener('click', function () { cancel(r.id, kindOf(r)); });
        li.appendChild(c);
      }
      ul.appendChild(li);
    });
    q.appendChild(ul);
  }

  // ---- events ----
  function onView(v) {
    view = v;
    // inventory.js redraws the queue card right after it hands the view over; redraw ours once that call has returned
    Promise.resolve().then(renderQueue);
    if (openKey) draw(true);
  }
  function keyOf(target) {
    var a = target && target.closest ? target.closest('[data-key]') : null;
    return a && a.dataset.key && /^(item|slot)-/.test(a.dataset.key) ? a.dataset.key : '';
  }
  function init() {
    document.addEventListener('d2:inventory', function (e) { onView(e.detail); });
    document.addEventListener('keydown', function (e) {
      kbd = true;
      if (!openKey) return;
      if (e.key === 'Escape') { closeMenu(true); return; }
      var list = Array.prototype.slice.call(focusables()), i = list.indexOf(document.activeElement);
      if (!list.length) return;
      if (e.key === 'ArrowDown' || e.key === 'ArrowUp' || e.key === 'Home' || e.key === 'End' || e.key === 'Tab') {
        var n = e.key === 'Home' ? 0 : e.key === 'End' ? list.length - 1 : e.key === 'ArrowUp' || (e.key === 'Tab' && e.shiftKey) ? (i <= 0 ? list.length - 1 : i - 1) : (i + 1) % list.length;
        e.preventDefault(); list[n].focus({ preventScroll: true });
      }
    });
    document.addEventListener('pointerdown', function () { kbd = false; });
    document.addEventListener('pointerdown', function (e) {
      if (openKey && menu && !menu.contains(e.target) && keyOf(e.target) !== openKey) closeMenu(false);
    });
    document.addEventListener('click', function (e) {
      if (menu && menu.contains(e.target)) return;
      var key = keyOf(e.target);
      if (!key || !find(key)) return;
      if (key === openKey) { closeMenu(true); return; }
      openMenu(key);
    });
    window.addEventListener('resize', function () { if (openKey && menu && !menu.hidden) position(); }, { passive: true });
    window.addEventListener('scroll', function () { if (openKey && menu && !menu.hidden) position(); }, { passive: true });
  }

  var api = { equip: equip, trash: trash, cancel: cancel, open: openMenu, close: function () { closeMenu(false); } };
  if (typeof window !== 'undefined') window.d2InvActions = api;
  if (typeof window !== 'undefined' && window.__actTest) {
    window.__actTest({ init: init, onView: onView, model: model, find: find, open: openMenu, close: closeMenu, place: place,
      state: function () { return { key: openKey, node: menu, confirming: confirming, sheet: sheet }; }, renderQueue: renderQueue, setSheet: function (s) { sheet = !!s; } });
    return;
  }
  init();
})();
