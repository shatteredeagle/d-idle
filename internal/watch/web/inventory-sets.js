// Gear sets of the inventory page (docs/gearsets.md), inside the paper doll card: save what the hero (or the mercenary)
// wears now under a name, rename, delete, and Apply. Apply shows a PREVIEW first (which item is where: worn, pack, stash,
// worn in another slot, missing, blocked, and the steps in the order they will run) and only the second click queues the
// steps as ordinary Equip, Take off and Take out requests in the queue list, where each can be cancelled (cancelling one
// ends the rest). Nothing here is automatic: the bot never saves, chooses or applies a set (docs/gear-policy.md).
// Locked until the ladder row `gearsets` opens (the view says why: sets.open, sets.why). Text through textContent only.
(function () {
  'use strict';
  var LABEL = { head: 'helm', neck: 'amulet', torso: 'armour', rhand: 'weapon', lhand: 'shield', rring: 'right ring', lring: 'left ring', belt: 'belt', feet: 'boots', gloves: 'gloves', mhead: 'helm', mtorso: 'armour', mrhand: 'weapon', mlhand: 'shield' };
  var WORD = { worn: 'worn', pack: 'in the backpack', stash: 'in the stash', elsewhere: 'worn in another slot', missing: 'missing', blocked: 'cannot be worn' };
  var OPS = { equip: 'Equip', unequip: 'Take off', take: 'Take out of the stash' };
  var view = null, host = null, drawn = '', pv = null, pvId = '', renameId = '', delId = '', note = '', nameDraft = '', busy = false;

  function el(t, c, x) { var e = document.createElement(t); if (c) e.className = c; if (x !== undefined) e.textContent = x; return e; }
  function clear(n) { while (n.firstChild) n.removeChild(n.firstChild); }
  function api() { return window.d2InvActions; }

  // ---- network ----
  function post(path, body) {
    var a = api(); if (!a || !a.base) return Promise.reject(new Error('the page is not ready'));
    return a.base().then(function (u) {
      return fetch(u + path, { method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
    }).then(function (r) { return r.json().then(function (j) { if (!r.ok) throw new Error(j.error || ('error ' + r.status)); return j; }); });
  }
  function act(path, body, ok) {
    if (busy) return Promise.resolve();
    busy = true;
    return post(path, body).then(function (j) { note = ok || ''; return j; }, function (e) { note = e.message; })
      .then(function () { busy = false; return api().refresh(); }).then(function () { busy = false; draw(true); }, function () { busy = false; draw(true); });
  }
  function loadPreview(id) {
    pvId = id; pv = null; note = ''; draw(true);
    fetch('/api/inventory/setpreview?id=' + encodeURIComponent(id), { cache: 'no-store' }).then(function (r) {
      return r.json().then(function (j) { if (!r.ok) throw new Error(j.error || ('error ' + r.status)); return j; });
    }).then(function (j) { if (pvId === id) { pv = j; draw(true); } }, function (e) { if (pvId === id) { note = e.message; pvId = ''; draw(true); } });
  }

  // ---- view ----
  function slotText(s) { return (LABEL[s.slot] || s.slot) + ': ' + s.name; }
  function previewNode(set) {
    var box = el('div', 'sets-preview'); box.setAttribute('role', 'group'); box.setAttribute('aria-label', 'Preview of ' + set.name);
    if (!pv) { box.appendChild(el('p', 'd-hint', 'Looking where the items are...')); return box; }
    var ul = el('ul', 'sets-rows');
    pv.slots.forEach(function (s) {
      var li = el('li'); li.dataset.status = s.status;
      li.appendChild(el('span', 'sl', LABEL[s.slot] || s.slot));
      li.appendChild(el('span', 'nm', s.name));
      li.appendChild(el('span', 'st', (WORD[s.status] || s.status) + (s.from ? ' (' + (LABEL[s.from] || s.from) + ')' : '')));
      if (s.why) li.appendChild(el('span', 'why', s.why));
      ul.appendChild(li);
    });
    box.appendChild(ul);
    if (pv.steps && pv.steps.length) {
      box.appendChild(el('p', 'd-muted', 'Steps, in this order:'));
      var ol = el('ol', 'sets-steps');
      pv.steps.forEach(function (st) { ol.appendChild(el('li', '', (OPS[st.op] || st.op) + ' ' + st.name + (st.slot ? ' (' + (LABEL[st.slot] || st.slot) + ')' : '') + (st.note ? ': ' + st.note : ''))); });
      box.appendChild(ol);
    }
    if (pv.missing) box.appendChild(el('p', 'd-note', pv.missing + ' item' + (pv.missing === 1 ? ' is' : 's are') + ' not found and skipped. Nothing is put in its place.'));
    var bar = el('div', 'd-btngroup');
    var go = el('button', 'act', 'Apply now'); go.type = 'button'; go.dataset.act = 'apply-go';
    if (!pv.ready) { go.setAttribute('aria-disabled', 'true'); go.dataset.off = ''; }
    go.addEventListener('click', function () {
      if (!pv.ready) { note = pv.why || 'nothing to apply'; draw(true); return; }
      var id = set.id; pvId = ''; pv = null;
      act('/api/inventory/sets/apply', { setId: id }, 'Queued. The steps are in the list below; the bot does them at a safe moment, and you can cancel them there.');
    });
    var no = el('button', 'act', 'Close'); no.type = 'button'; no.dataset.act = 'apply-close';
    no.addEventListener('click', function () { pvId = ''; pv = null; draw(true); });
    bar.appendChild(go); bar.appendChild(no); box.appendChild(bar);
    if (!pv.ready && pv.why) box.appendChild(el('p', 'd-hint sets-why', pv.why));
    return box;
  }
  function row(set) {
    var li = el('li', 'sets-set'); li.dataset.set = set.id; li.dataset.target = set.target;
    var head = el('div', 'sets-head');
    if (renameId === set.id) {
      var inp = el('input', 'sets-name'); inp.setAttribute('type', 'text'); inp.setAttribute('maxlength', '24'); inp.setAttribute('aria-label', 'New name'); inp.value = set.name;
      var okb = el('button', 'act', 'Save name'); okb.type = 'button'; okb.dataset.act = 'rename-ok';
      okb.addEventListener('click', function () { var id = set.id; renameId = ''; act('/api/inventory/sets/rename', { setId: id, name: inp.value }, 'Renamed.'); });
      var cb = el('button', 'act', 'Cancel'); cb.type = 'button'; cb.dataset.act = 'rename-no';
      cb.addEventListener('click', function () { renameId = ''; draw(true); });
      head.appendChild(inp); head.appendChild(okb); head.appendChild(cb);
    } else {
      head.appendChild(el('span', 'nm', set.name));
      head.appendChild(el('span', 'meta', (set.target === 'merc' ? 'mercenary' : 'hero') + ', ' + set.slots.length + ' item' + (set.slots.length === 1 ? '' : 's')));
    }
    li.appendChild(head);
    var items = el('p', 'sets-items d-muted', set.slots.map(slotText).join(', ')); li.appendChild(items);
    var bar = el('div', 'd-btngroup');
    function btn(label, key, fn, off) { var b = el('button', 'act', label); b.type = 'button'; b.dataset.act = key; if (off) { b.setAttribute('aria-disabled', 'true'); b.dataset.off = ''; } b.addEventListener('click', fn); bar.appendChild(b); return b; }
    if (set.applying) {
      btn('Cancel apply', 'apply-cancel', function () { act('/api/inventory/sets/cancel', { group: set.applying }, 'Cancelled.'); });
    } else {
      btn('Apply...', 'apply', function () { if (pvId === set.id) { pvId = ''; pv = null; draw(true); } else loadPreview(set.id); });
    }
    btn('Rename', 'rename', function () { renameId = set.id; delId = ''; draw(true); });
    var dl = btn(delId === set.id ? 'Confirm: delete it' : 'Delete', 'delete', function () {
      if (delId !== set.id) { delId = set.id; draw(true); return; }
      delId = ''; if (pvId === set.id) { pvId = ''; pv = null; } act('/api/inventory/sets/delete', { setId: set.id }, 'Deleted.');
    });
    if (delId === set.id) { dl.dataset.danger = ''; dl.dataset.confirm = ''; }
    li.appendChild(bar);
    if (pvId === set.id) li.appendChild(previewNode(set));
    return li;
  }
  function saveBar(s) {
    var bar = el('div', 'sets-save');
    var lab = el('label', 'sets-lab', 'Name'); lab.setAttribute('for', 'sets-new');
    var inp = el('input', 'sets-name'); inp.id = 'sets-new'; inp.setAttribute('type', 'text'); inp.setAttribute('maxlength', '24'); inp.setAttribute('placeholder', 'e.g. Fire damage'); inp.value = nameDraft;
    inp.addEventListener('input', function () { nameDraft = inp.value; });
    function save(target, text) {
      var b = el('button', 'act', text); b.type = 'button'; b.dataset.act = 'save-' + target;
      b.addEventListener('click', function () {
        var nm = inp.value; nameDraft = '';
        act('/api/inventory/sets/save', { name: nm, target: target }, 'Saved.').then(function () { if (/^Saved/.test(note) === false) nameDraft = nm; });
      });
      return b;
    }
    bar.appendChild(lab); bar.appendChild(inp); bar.appendChild(save('hero', 'Save what the hero wears'));
    if (view.merc && view.merc.owned) bar.appendChild(save('merc', 'Save what the mercenary wears'));
    return bar;
  }
  function fill(s) {
    clear(host);
    var h = el('div', 'sets-title'); h.appendChild(el('h3', '', 'Gear sets'));
    if (s.open) h.appendChild(el('span', 'meta', (s.sets || []).length + ' of ' + s.max));
    host.appendChild(h);
    if (!s.open) {
      var lk = el('div', 'd-locked sets-locked'); lk.dataset.state = 'locked';
      lk.appendChild(el('p', 'd-hint teaser', s.why || 'Locked.'));
      host.appendChild(lk);
      return;
    }
    host.appendChild(el('p', 'd-hint', 'Save what you wear now and put it back on later. A set goes on only when you click Apply; the bot never changes your gear. An item that is gone is skipped, never replaced by another.'));
    host.appendChild(saveBar(s));
    if ((s.sets || []).length) { var ul = el('ul', 'sets-list'); s.sets.forEach(function (x) { ul.appendChild(row(x)); }); host.appendChild(ul); }
    else host.appendChild(el('p', 'd-empty', 'No sets saved yet.'));
    var n = el('p', 'd-hint sets-note', note); n.setAttribute('role', 'status'); host.appendChild(n);
  }
  function draw(force) {
    if (!host) { host = document.getElementById('gearsets'); if (!host) return; }
    var s = view && view.sets;
    if (!s) { host.hidden = true; drawn = ''; return; }
    host.hidden = false;
    var sig = JSON.stringify([s, view.merc && view.merc.owned, pvId, pv, renameId, delId, note]);
    if (!force && sig === drawn) return;
    // a name being typed (new set or rename) survives a redraw by the poll: keep the focused field's text
    var a = document.activeElement, keep = a && a.id === 'sets-new' ? a.value : null;
    drawn = sig;
    fill(s);
    if (keep !== null) { var n = document.getElementById('sets-new'); if (n) { n.value = keep; nameDraft = keep; } }
  }
  function onView(v) { view = v; draw(false); }
  function init() {
    document.addEventListener('d2:inventory', function (e) { onView(e.detail); });
    document.addEventListener('d2:inventory-fresh', function (e) { onView(e.detail); });
  }
  if (typeof window !== 'undefined' && window.__setsTest) {
    window.__setsTest({ init: init, onView: onView, draw: draw, state: function () { return { host: host, pvId: pvId, note: note }; }, setPreview: function (id, p) { pvId = id; pv = p; } });
    return;
  }
  init();
})();
