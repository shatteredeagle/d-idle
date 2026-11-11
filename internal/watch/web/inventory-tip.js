// Item tooltip of the inventory page (docs/inventory-tooltip.md, docs/ui-design-system.md components 13 and 14): hover or
// keyboard focus on desktop, tap to open and tap away to close on a phone, for every item cell and filled doll slot.
// One element, created once and refilled; the page's 4 s poll only re-renders it when the open item's text changed, and
// nothing here scrolls the page: the tip is position: fixed, placed by a transform, and takes no focus.
// Input: the view /api/inventory sends, handed over by inventory.js as a 'd2:inventory' event on document. Text goes in
// through textContent only. The lines are the game's own tooltip text when the backend has it (item.tip, d2d "items"
// verb, S), else built from the game tables' base facts (item.base) and the props the snapshot carries.
(function () {
  'use strict';
  var TIP_ID = 'inv-tip';
  var ARMOR_LINE = /^(defen[cs]e|one-hand damage|two-hand damage|throw damage|damage|durability|required|quantity|item level|sockets?|socketed|ethereal|smite damage|chance to block|quality)/i;
  var lastType = 'mouse', view = null, openKey = '', openSig = '', mode = '', kbd = false, node = null, host = null;

  function el(t, c, x) { var e = document.createElement(t); if (c) e.className = c; if (x !== undefined) e.textContent = x; return e; }
  function clear(n) { while (n.firstChild) n.removeChild(n.firstChild); }

  // The colour class of the name: the quality, except that a plain item that is ethereal or socketed is grey like in
  // the game (low quality is grey too).
  function nameClass(it) {
    var q = it.quality || 'normal';
    if ((q === 'normal' || q === 'superior') && (it.ethereal || it.sockets)) return 'q-low';
    return 'q-' + q;
  }
  function range(a, b) { return a === b ? String(a) : a + ' to ' + b; }

  // build returns {cls, lines: [{k, t}], badges: [text]} for one item. slot is the doll slot key or ''.
  function build(it, slot) {
    var L = [], has = function (re) { return L.some(function (l) { return re.test(l.t); }); };
    var add = function (k, t) { if (t) L.push({ k: k, t: t }); };
    var b = it.base || null, tip = it.tip || [];
    var title = it.name || it.code || '';
    if (tip.length) {
      title = tip[0];
      tip.slice(1).forEach(function (t) { add(ARMOR_LINE.test(t) ? 'base' : 'prop', t); });
    } else {
      if (b && b.name && b.name !== title) add('type', b.name);
      if (b) {
        if (b.minAC || b.maxAC) add('base', 'Base defense: ' + range(b.minAC, b.maxAC));
        // weapons.txt: a true two-hander keeps its damage in the two-hand columns; everything else uses min and max
        if (b.twoHanded && b.twoMaxDam) add('base', 'Two-Hand damage: ' + range(b.twoMinDam, b.twoMaxDam));
        else if (b.minDam || b.maxDam) add('base', (b.twoHanded ? 'Two-Hand' : 'One-Hand') + ' damage: ' + range(b.minDam, b.maxDam));
        if (b.block) add('base', 'Base chance to block: ' + b.block + '%');
        if (b.speed) add('base', 'Base weapon speed: ' + b.speed);
      }
      if (it.maxDur) add('base', 'Durability: ' + (it.dur || 0) + ' of ' + it.maxDur + (!it.dur ? ' (broken)' : ''));
      else if (b && b.dur) add('base', 'Durability: ' + b.dur);
      if (b && b.reqStr) add('req', 'Required strength: ' + b.reqStr);
      if (b && b.reqDex) add('req', 'Required dexterity: ' + b.reqDex);
      if (b && b.reqLvl) add('req', 'Required level: ' + b.reqLvl);
      if (it.ethereal) add('eth', 'Ethereal (cannot be repaired)');
      if (it.sockets) add('sock', 'Socketed (' + (it.filled || 0) + '/' + it.sockets + ')');
      (it.props || []).forEach(function (p) { add('prop', p); });
    }
    if (!it.identified) add('bad', 'Unidentified');
    else if (!tip.length && !(it.props || []).length && (it.quality === 'magic' || it.quality === 'rare' || it.quality === 'set' || it.quality === 'unique' || it.quality === 'crafted')) {
      add('note', 'Magic properties are not read on this backend.');
    }
    if (it.sockets && tip.length && !has(/socket/i)) add('sock', 'Socketed (' + (it.filled || 0) + '/' + it.sockets + ')');
    if (it.ethereal && tip.length && !has(/ethereal/i)) add('eth', 'Ethereal (cannot be repaired)');
    if (it.level) add('meta', 'Item level: ' + it.level);
    if (it.sell) {
      var exact = !it.identified || it.quality === 'normal' || it.quality === 'low' || it.quality === 'superior';
      add('meta', (exact ? 'Sells for ' : 'Sells for at least ') + it.sell + ' gold');
    }
    var badges = [];
    if (it.mark === 'sell') badges.push('SELL');
    if (it.mark === 'keep') badges.push('KEEP');
    return { cls: nameClass(it), title: title, lines: L, badges: badges };
  }

  // place returns the top-left of the tip so that it sits beside the anchor (right, else left, else below or above) and
  // stays inside the viewport with an 8 px margin. a: {left, top, right, bottom}.
  function place(a, w, h, vw, vh) {
    var m = 8, x, y;
    if (a.right + m + w <= vw - m) x = a.right + m;
    else if (a.left - m - w >= m) x = a.left - m - w;
    else x = Math.round((a.left + a.right) / 2 - w / 2);
    y = a.top;
    if (x === Math.round((a.left + a.right) / 2 - w / 2)) y = a.bottom + m + h <= vh - m ? a.bottom + m : a.top - m - h; // narrow screens: above or below
    x = Math.max(m, Math.min(x, vw - w - m));
    y = Math.max(m, Math.min(y, vh - h - m));
    return { x: Math.round(x), y: Math.round(y) };
  }

  function ensure() {
    if (node) return node;
    node = el('div', 'inv-tip'); node.id = TIP_ID; node.setAttribute('role', 'tooltip'); node.hidden = true;
    document.body.appendChild(node);
    return node;
  }
  function render(t) {
    var n = ensure(); clear(n);
    var h = el('div', 'tip-name ' + t.cls, t.title); n.appendChild(h);
    t.lines.forEach(function (l) { var d = el('div', 'tip-line', l.t); d.dataset.k = l.k; n.appendChild(d); });
    if (t.badges.length) {
      var row = el('div', 'tip-badges');
      t.badges.forEach(function (x) { var s = el('span', 'tip-badge', x); s.dataset.badge = x.toLowerCase(); row.appendChild(s); });
      n.appendChild(row);
    }
  }

  function find(key) {
    if (!view || !key) return null;
    var m = /^(item|slot)-(.+)$/.exec(key);
    if (!m) return null;
    if (m[1] === 'slot') return { it: view.equipment && view.equipment[m[2]], slot: m[2] };
    var id = Number(m[2]), hit = null;
    [view.backpack, view.stash, view.belt, view.cube].forEach(function (l) { (l || []).forEach(function (x) { if (x.id === id) hit = x; }); });
    return { it: hit, slot: '' };
  }
  function anchorOf(key) { return document.querySelector('[data-key="' + key + '"]'); }
  function keyOf(target) {
    var a = target && target.closest ? target.closest('[data-key]') : null;
    if (!a || !a.dataset.key || !/^(item|slot)-/.test(a.dataset.key)) return '';
    return a.dataset.key;
  }

  function position(key) {
    var a = anchorOf(key), n = ensure();
    if (!a) { hide(); return; }
    var r = a.getBoundingClientRect(), p = place(r, n.offsetWidth, n.offsetHeight, document.documentElement.clientWidth, document.documentElement.clientHeight);
    n.style.transform = 'translate(' + p.x + 'px, ' + p.y + 'px)';
  }
  function show(key, how) {
    var f = find(key);
    if (!f || !f.it) { hide(); return; }
    var t = build(f.it, f.slot), sig = JSON.stringify(t);
    var n = ensure(), a = anchorOf(key);
    if (a) { if (a.removeAttribute) a.removeAttribute('title'); a.setAttribute('aria-describedby', TIP_ID); }
    if (openKey && openKey !== key) { var o = anchorOf(openKey); if (o) o.removeAttribute('aria-describedby'); }
    if (sig !== openSig || n.hidden) { render(t); openSig = sig; }
    openKey = key; mode = how;
    n.hidden = false;
    position(key);
  }
  function hide() {
    if (openKey) { var o = anchorOf(openKey); if (o && o.removeAttribute) o.removeAttribute('aria-describedby'); }
    openKey = ''; openSig = ''; mode = '';
    if (node) node.hidden = true;
  }

  // the page's poll: refill the open tip only when its text changed, or close it when the item left
  function onView(v) {
    view = v;
    if (!openKey) return;
    var f = find(openKey);
    if (!f || !f.it) { hide(); return; }
    show(openKey, mode);
  }

  function wire(root) {
    root.addEventListener('pointerover', function (e) {
      if (e.pointerType && e.pointerType !== 'mouse') return;
      var k = keyOf(e.target); if (k && mode !== 'tap') show(k, 'hover');
    });
    root.addEventListener('pointerout', function (e) {
      if (mode === 'hover' && keyOf(e.target) === openKey && !(e.relatedTarget && keyOf(e.relatedTarget) === openKey)) hide();
    });
    root.addEventListener('focusin', function (e) { var k = keyOf(e.target); if (k && kbd) show(k, 'focus'); });
    root.addEventListener('focusout', function (e) { if (mode === 'focus' && keyOf(e.target) === openKey) hide(); });
    root.addEventListener('click', function (e) {
      var k = keyOf(e.target);
      if (!k || (lastType !== 'touch' && lastType !== 'pen')) return;
      if (openKey === k && mode === 'tap') hide(); else show(k, 'tap');
    });
  }
  function init() {
    ['equip', 'pack', 'belt', 'stash'].forEach(function (id) { var r = document.getElementById(id); if (r) wire(r); });
    document.addEventListener('d2:inventory', function (e) { onView(e.detail); });
    document.addEventListener('pointerdown', function (e) {
      lastType = e.pointerType || 'mouse'; kbd = false;
      if (mode === 'tap' && !(node && node.contains && node.contains(e.target)) && keyOf(e.target) !== openKey) hide();
    }, true);
    document.addEventListener('keydown', function (e) {
      kbd = true;
      if (e.key === 'Escape' && openKey) hide();
    });
    window.addEventListener('scroll', function () { if (openKey) position(openKey); }, { passive: true });
    window.addEventListener('resize', function () { if (openKey) position(openKey); });
  }

  if (typeof window !== 'undefined' && window.__tipTest) {
    window.__tipTest({ build: build, place: place, nameClass: nameClass, onView: onView, show: show, hide: hide, init: init, state: function () { return { key: openKey, mode: mode, node: node }; } });
    return;
  }
  init();
})();
