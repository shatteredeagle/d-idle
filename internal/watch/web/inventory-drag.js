// Drag and drop on the inventory page (docs/gearsets.md): drag an item tile of the backpack onto a paper doll slot (the hero's
// or the mercenary's) and the same Equip request the menu queues is queued; drag a worn item onto the backpack and an
// Unequip is queued; drag a backpack tile outside every card and the menu opens on its Trash confirm step; drag between the
// backpack and the stash for the stash moves. Nothing here decides anything: the green or red of a slot comes from the
// server's own rule (item.canWear and item.noWearSlot, view schema 7), the action is the menu's (window.d2InvActions)
// and the server checks it again. A drop never acts on its own: the request shows in the queue list and can be cancelled.
// Pointer events, not HTML5 drag and drop (unreliable on touch): a mouse starts after a few pixels, a finger after a short
// hold (so the page still scrolls); while a drag runs the page keeps its tiles (body[data-dragging]) and the finger's
// scroll is blocked. Keyboard and screen readers: the item menu already has every action (Equip per slot, Take off); in
// addition an item that is selected (click or Enter on its tile) is worn by pressing Enter or clicking an EMPTY slot it fits.
// Text through textContent only; no inline styles except the ghost's transform (CSSOM).
(function () {
  'use strict';
  var HOLD_MS = 280, MOVE_PX = 6, SWALLOW_MS = 450;
  var view = null, pend = null, drag = null, swallowUntil = 0, ghost = null, lastKey = '', scrollGuard = null;

  function el(t, c, x) { var e = document.createElement(t); if (c) e.className = c; if (x !== undefined) e.textContent = x; return e; }
  function nameOf(it) { return (it.name || it.code || 'item'); }

  // ---- model (pure) ----
  // find resolves a tile or a slot to {it, place, slot, owner}: place is pack, stash, belt, worn (hero) or mercworn.
  function find(v, ref) {
    if (!v || !ref) return null;
    if (ref.owner === 'merc') {
      var m = v.merc && v.merc.equipment && v.merc.equipment[ref.slot];
      return m ? { it: m, place: 'mercworn', slot: ref.slot, owner: 'merc' } : null;
    }
    if (ref.slot) { var w = v.equipment && v.equipment[ref.slot]; return w ? { it: w, place: 'worn', slot: ref.slot, owner: 'hero' } : null; }
    var hit = null;
    [['pack', v.backpack], ['stash', v.stash], ['belt', v.belt]].forEach(function (l) {
      (l[1] || []).forEach(function (x) { if (x.id === ref.id) hit = { it: x, place: l[0], slot: '', owner: 'hero' }; });
    });
    return hit;
  }

  // verdict says what dropping src on tgt does: {kind: 'none'} (not a target for this item: no colour),
  // {kind: 'no', reason} (red, with the server's reason) or {kind: 'ok', act, ...} (green).
  // tgt: {type: 'slot', slot, owner} | {type: 'pack'} | {type: 'stash'} | {type: 'outside'} | {type: 'other'}
  function verdict(v, src, tgt) {
    var it = src.it, no = function (r) { return { kind: 'no', reason: r || 'not possible right now' }; };
    var mercWhy = (v && v.merc && v.merc.actWhy) || '';
    if (tgt.type === 'other') return { kind: 'none' };
    switch (src.place) {
      case 'pack':
        if (tgt.type === 'slot') {
          if (tgt.owner === 'merc') return mercWhy ? no(mercWhy) : { kind: 'ok', act: 'equip', id: it.id, slot: tgt.slot, target: 'merc' };
          if ((it.canWear || []).indexOf(tgt.slot) >= 0) return { kind: 'ok', act: 'equip', id: it.id, slot: tgt.slot };
          return no((it.noWearSlot && it.noWearSlot[tgt.slot]) || it.noWear || 'it does not fit this slot');
        }
        if (tgt.type === 'stash') return it.stashOp === 'stash' ? { kind: 'ok', act: 'stash', id: it.id } : no(it.noStash || 'the stash is not available');
        if (tgt.type === 'outside') return it.noTrash ? no(it.noTrash) : { kind: 'ok', act: 'trash', id: it.id, key: 'item-' + it.id };
        return { kind: 'none' };
      case 'stash':
        if (tgt.type === 'slot') return no(tgt.owner === 'merc' ? 'move it to the backpack first' : (it.noWear || 'move it to the backpack first'));
        if (tgt.type === 'pack') return it.stashOp === 'take' ? { kind: 'ok', act: 'take', id: it.id } : no(it.noStash || 'taking it out is not available');
        if (tgt.type === 'outside') return no(it.noTrash || 'move it to the backpack first');
        return { kind: 'none' };
      case 'belt':
        if (tgt.type === 'slot') return no(it.noWear || 'move it to the backpack first');
        return { kind: 'none' };
      case 'worn':
        if (tgt.type === 'pack') return it.noUnequip ? no(it.noUnequip) : { kind: 'ok', act: 'unequip', slot: src.slot };
        if (tgt.type === 'outside') return no('take it off first (drop it on the backpack), then trash it from there');
        return { kind: 'none' };
      case 'mercworn':
        if (tgt.type === 'pack') return mercWhy ? no(mercWhy) : { kind: 'ok', act: 'unequip', slot: src.slot, target: 'merc' };
        return { kind: 'none' };
    }
    return { kind: 'none' };
  }

  // ---- DOM glue ----
  function slotEls() { return document.querySelectorAll('.d-slot2[data-slot]'); }
  function srcOf(node) {
    if (!node || !node.closest) return null;
    var s = node.closest('.d-slot2[data-slot]');
    if (s && s.dataset.filled !== undefined) return find(view, { slot: s.dataset.slot, owner: s.dataset.owner === 'merc' ? 'merc' : 'hero' });
    var t = node.closest('[data-key^="item-"]');
    if (t) { var m = /^item-(\d+)$/.exec(t.dataset.key); if (m) return find(view, { id: Number(m[1]) }); }
    return null;
  }
  function targetAt(x, y) {
    var n = document.elementFromPoint(x, y);
    if (!n || !n.closest) return { type: 'outside' };
    var s = n.closest('.d-slot2[data-slot]');
    if (s) return { type: 'slot', slot: s.dataset.slot, owner: s.dataset.owner === 'merc' ? 'merc' : 'hero', node: s };
    if (n.closest('#pack')) return { type: 'pack', node: document.getElementById('pack') };
    if (n.closest('#stash')) return { type: 'stash', node: document.getElementById('stash') };
    if (n.closest('.d-card, .top, .inv-act, .d-toasts, .d-note')) return { type: 'other' };
    return { type: 'outside' };
  }
  function markAll(src) {
    Array.prototype.forEach.call(slotEls(), function (s) {
      var vd = verdict(view, src, { type: 'slot', slot: s.dataset.slot, owner: s.dataset.owner === 'merc' ? 'merc' : 'hero' });
      if (vd.kind === 'none') delete s.dataset.drop; else s.dataset.drop = vd.kind;
    });
    ['pack', 'stash'].forEach(function (id) {
      var n = document.getElementById(id); if (!n) return;
      var vd = verdict(view, src, { type: id });
      if (vd.kind === 'none') delete n.dataset.drop; else n.dataset.drop = vd.kind;
    });
  }
  function clearMarks() {
    Array.prototype.forEach.call(slotEls(), function (s) { delete s.dataset.drop; delete s.dataset.over; });
    ['pack', 'stash'].forEach(function (id) { var n = document.getElementById(id); if (n) { delete n.dataset.drop; delete n.dataset.over; } });
  }
  function say(t) { if (window.d2InvActions && window.d2InvActions.say) window.d2InvActions.say(t); }

  function start(p) {
    if (!p.src) return;
    drag = { src: p.src, id: p.id };
    document.body.setAttribute('data-dragging', '');
    ghost = el('div', 'inv-ghost'); ghost.setAttribute('aria-hidden', 'true');
    ghost.appendChild(el('span', 'g-name', nameOf(p.src.it)));
    ghost.appendChild(el('span', 'g-why'));
    document.body.appendChild(ghost);
    if (p.src.it.quality) ghost.classList.add('q-' + p.src.it.quality);
    markAll(p.src);
    lastKey = '';
    scrollGuard = function (e) { if (drag && e.cancelable) e.preventDefault(); };
    document.addEventListener('touchmove', scrollGuard, { passive: false });
    move(p.x, p.y);
  }
  function move(x, y) {
    if (!drag || !ghost) return;
    ghost.style.transform = 'translate(' + Math.round(x + 14) + 'px, ' + Math.round(y + 14) + 'px)';
    var tgt = targetAt(x, y), vd = verdict(view, drag.src, tgt);
    var key = tgt.type + (tgt.slot || '') + (tgt.owner || '') + vd.kind + (vd.reason || '');
    if (key === lastKey) return;
    lastKey = key;
    Array.prototype.forEach.call(document.querySelectorAll('[data-over]'), function (n) { delete n.dataset.over; });
    if (tgt.node && vd.kind !== 'none') tgt.node.dataset.over = vd.kind;
    ghost.dataset.state = vd.kind;
    var why = ghost.querySelector('.g-why');
    why.textContent = vd.kind === 'no' ? vd.reason : vd.kind === 'ok' ? label(vd) : '';
  }
  function label(vd) {
    switch (vd.act) {
      case 'equip': return vd.target === 'merc' ? 'Equip the mercenary' : 'Equip';
      case 'unequip': return 'Take off';
      case 'stash': return 'Move to the stash';
      case 'take': return 'Take out of the stash';
      case 'trash': return 'Trash (asks to confirm)';
    }
    return '';
  }
  function finish(x, y, commit) {
    var d = drag;
    drag = null; pend = null;
    if (scrollGuard) { document.removeEventListener('touchmove', scrollGuard, { passive: false }); scrollGuard = null; }
    if (ghost && ghost.parentNode) ghost.parentNode.removeChild(ghost);
    ghost = null;
    clearMarks();
    document.body.removeAttribute('data-dragging');
    swallowUntil = Date.now() + SWALLOW_MS;
    var vd = commit && d ? verdict(view, d.src, targetAt(x, y)) : { kind: 'none' };
    if (vd.kind === 'ok') perform(vd);
    else if (vd.kind === 'no') say(vd.reason);
    document.dispatchEvent(new CustomEvent('d2:dragend'));
  }
  function perform(vd) {
    var a = window.d2InvActions; if (!a) return;
    switch (vd.act) {
      case 'equip': a.equip(vd.id, vd.slot, vd.target); break;
      case 'unequip': a.unequip(vd.slot, vd.target); break;
      case 'stash': a.stash(vd.id); break;
      case 'take': a.take(vd.id); break;
      case 'trash': a.trashConfirm(vd.key); break;
    }
  }

  // ---- events ----
  function onDown(e) {
    if (drag || (e.button !== undefined && e.button !== 0) || e.isPrimary === false) return;
    var src = srcOf(e.target); if (!src) return;
    var p = { src: src, id: e.pointerId, x: e.clientX, y: e.clientY, x0: e.clientX, y0: e.clientY, touch: e.pointerType === 'touch', timer: 0 };
    if (p.touch) p.timer = setTimeout(function () { if (pend === p) start(p); }, HOLD_MS);
    pend = p;
  }
  function onMove(e) {
    if (drag) { if (e.pointerId === drag.id) move(e.clientX, e.clientY); return; }
    if (!pend || e.pointerId !== pend.id) return;
    pend.x = e.clientX; pend.y = e.clientY;
    var far = Math.abs(e.clientX - pend.x0) > MOVE_PX || Math.abs(e.clientY - pend.y0) > MOVE_PX;
    if (!far) return;
    if (pend.touch) { clearTimeout(pend.timer); pend = null; return; } // the finger moved before the hold: the page scrolls
    start(pend);
  }
  function onUp(e) {
    if (drag) { if (e.pointerId === drag.id) finish(e.clientX, e.clientY, true); return; }
    if (pend && e.pointerId === pend.id) { clearTimeout(pend.timer); pend = null; }
  }
  function onCancel(e) {
    if (drag && e.pointerId === drag.id) finish(0, 0, false);
    else if (pend && e.pointerId === pend.id) { clearTimeout(pend.timer); pend = null; }
  }
  // A click right after a drag would open the menu of the tile that was dragged; and the armed-item path below.
  function onClick(e) {
    if (Date.now() < swallowUntil) { e.stopPropagation(); e.preventDefault(); return; }
    var s = e.target && e.target.closest ? e.target.closest('.d-slot2[data-slot]') : null;
    if (!s || s.dataset.filled !== undefined || s.dataset.owner === 'merc') return;
    var armed = document.querySelector('.item.sel');
    var m = armed && armed.dataset && /^item-(\d+)$/.exec(armed.dataset.key || '');
    var src = m ? find(view, { id: Number(m[1]) }) : null;
    if (!src || src.place !== 'pack') return;
    var vd = verdict(view, src, { type: 'slot', slot: s.dataset.slot, owner: 'hero' });
    e.stopPropagation();
    if (vd.kind === 'ok') perform(vd); else if (vd.kind === 'no') say(vd.reason);
  }
  function onKey(e) { if (e.key === 'Escape' && drag) finish(0, 0, false); }
  function init() {
    document.addEventListener('d2:inventory', function (e) { view = e.detail; });
    document.addEventListener('d2:inventory-fresh', function (e) { view = e.detail; });
    document.addEventListener('dragstart', function (e) { if (srcOf(e.target)) e.preventDefault(); }); // no native image drag
    document.addEventListener('pointerdown', onDown);
    document.addEventListener('pointermove', onMove);
    document.addEventListener('pointerup', onUp);
    document.addEventListener('pointercancel', onCancel);
    document.addEventListener('keydown', onKey);
    document.addEventListener('click', onClick, true);
    // a long press on a touch screen opens the context menu of some browsers: not while a drag runs
    document.addEventListener('contextmenu', function (e) { if (drag || (pend && pend.touch)) e.preventDefault(); });
  }

  if (typeof window !== 'undefined' && window.__dragTest) {
    window.__dragTest({ find: find, verdict: verdict, init: init, setView: function (v) { view = v; }, state: function () { return { drag: drag, pend: pend, ghost: ghost, swallowUntil: swallowUntil }; }, srcOf: srcOf, targetAt: targetAt });
    return;
  }
  init();
})();
