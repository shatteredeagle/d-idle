// Merc paper doll of the inventory page (docs/research/merc-gear.md section 3, TASK-287). Read only: it draws the
// merc block of GET /api/inventory (schema 3) under the hero's doll and has no actions yet (stage B and C).
// It listens for the d2:inventory event that inventory.js dispatches after every paint, so it never touches
// inventory.js. A reply without a merc block (schema 2, or a backend that cannot read a merc) leaves the card hidden.
// Text through textContent only; no inline styles. States: none, ready, dead, unknown (see invdesk.MercView).
(function () {
  'use strict';
  var KIND = { rogue: 'Rogue Scout', desert: 'Desert Guard', ironwolf: 'Iron Wolf', barbarian: 'Barbarian' };
  var SLOTS = ['mhead', 'mtorso', 'mrhand', 'mlhand'];
  var LABEL = { mhead: 'helm', mtorso: 'armour', mrhand: 'weapon', mlhand: 'shield' };
  // what the weapon hand takes per kind (S: SME advice and the d2d slot rule; the Barbarian and javelin cases are U)
  var WEAPON = { rogue: 'bow', desert: 'spear or polearm', ironwolf: 'one-handed sword', barbarian: 'sword' };
  var QUAL = /^[a-z]{1,12}$/;
  var SLOTART = { mhead: 'helm', mtorso: 'armor', mrhand: 'weapon', mlhand: 'weapon' }; // existing /watch/art ghost keys
  var ART_NAME = /^[a-z0-9_]{1,16}$/;
  var host = null, fold = null, body = null, drawn = '', lastView = null, art = { on: false, probed: false };

  function el(t, c, x) { var e = document.createElement(t); if (c) e.className = c; if (x !== undefined) e.textContent = x; return e; }
  function clear(n) { while (n.firstChild) n.removeChild(n.firstChild); }
  function name(it) { return (it.name || it.code) + (it.ethereal ? ' (eth)' : '') + (it.identified ? '' : ' (unidentified)'); }
  function kindName(m) { return KIND[m.kind] || 'Mercenary'; }
  function slotLabel(m, s) { return s === 'mrhand' && WEAPON[m.kind] ? 'weapon: ' + WEAPON[m.kind] : LABEL[s]; }

  // game art is an optional layer (docs/paperdoll-art-build.md): an image that fails to load goes away and the drawn
  // slot shows again. The probe loads the same panel frame the hero doll needs, so a missing art cache changes nothing.
  function artImg(host, cls, key) {
    var i = el('img', cls); i.setAttribute('alt', ''); i.setAttribute('src', '/watch/art/' + key + '.png');
    host.dataset.shown = '';
    i.addEventListener('error', function () { host.removeChild(i); delete host.dataset.shown; });
    host.appendChild(i);
  }
  function artProbe() {
    if (art.probed || /(^|[?&])mode=overlay(&|$)/.test(location.search) || typeof Image !== 'function') return;
    art.probed = true;
    var i = new Image();
    i.onload = function () { art.on = true; drawn = ''; if (lastView) render(lastView); };
    i.src = '/watch/art/panel-inv.png';
  }

  function slot(m, s) {
    var it = m.equipment && m.equipment[s];
    var legal = !m.slots || !m.slots.length || m.slots.indexOf(s) >= 0; // unknown kind: do not grey anything
    var b = el('div', 'd-slot2'); b.dataset.slot = s; b.dataset.owner = 'merc';
    if (art.on && legal && it && it.art && ART_NAME.test(it.art)) artImg(b, 'ico', 'item-' + it.art);
    else if (art.on && legal && !it) artImg(b, 'ghost', 'slot-' + SLOTART[s]);
    b.appendChild(el('span', 'sil', LABEL[s]));
    if (!legal) {
      b.dataset.state = 'na';
      var why = 'No ' + LABEL[s] + ' for a ' + kindName(m);
      b.title = why; b.setAttribute('aria-label', why);
      b.appendChild(el('span', 'd-badge', 'n/a'));
      return b;
    }
    if (it) {
      b.dataset.filled = '';
      if (it.quality && QUAL.test(it.quality)) b.classList.add('q-' + it.quality);
      b.appendChild(el('span', 'nm', name(it)));
      var tip = name(it);
      if (it.maxDur) tip += ', durability ' + (it.dur || 0) + '/' + it.maxDur;
      if (it.level) tip += ', item level ' + it.level;
      b.title = tip;
      b.setAttribute('aria-label', LABEL[s] + ': ' + name(it));
      if (it.maxDur && !(it.dur > 0)) b.appendChild(el('span', 'broken', 'Broken'));
    } else {
      b.title = slotLabel(m, s) + ': empty';
      b.setAttribute('aria-label', LABEL[s] + ': empty');
    }
    return b;
  }

  function header(m) {
    var h = el('div', 'inv-merc-head');
    h.appendChild(el('h3', '', kindName(m)));
    var bits = [];
    if (m.level) bits.push('level ' + m.level);
    if (m.str || m.dex) bits.push('Str ' + (m.str || 0) + ', Dex ' + (m.dex || 0));
    if (m.owned && m.state !== 'dead') bits.push('life ' + (m.lifePct || 0) + '%');
    if (bits.length) h.appendChild(el('span', 'meta', bits.join(', ')));
    return h;
  }

  function fill(m) {
    clear(body);
    if (m.state === 'none') {
      body.appendChild(el('p', 'd-empty', 'No mercenary hired'));
      body.appendChild(el('p', 'd-hint', 'Hire one in Act 1 (free after Blood Raven) or Act 2.'));
      return;
    }
    body.appendChild(header(m));
    if (m.state === 'dead') { var n = el('p', 'd-note', m.note || 'Mercenary is dead.'); n.dataset.sev = 'warn'; body.appendChild(n); }
    if (m.state === 'unknown') body.appendChild(el('p', 'd-hint', m.note || 'Equipment unreadable on this backend.'));
    if (m.state !== 'unknown') {
      var doll = el('div', 'd-doll'); doll.dataset.owner = 'merc'; doll.dataset.state = m.state;
      doll.setAttribute('role', 'group'); doll.setAttribute('aria-label', 'Mercenary paper doll');
      SLOTS.forEach(function (s) { doll.appendChild(slot(m, s)); });
      body.appendChild(doll);
      body.appendChild(el('p', 'd-hint', 'Read only: the bot makes no equipment decision for your mercenary, and equipping him from this page is not built yet.'));
    }
  }

  function render(v) {
    if (!host) { host = document.getElementById('merc-doll'); if (!host) return; }
    lastView = v;
    var m = v && v.merc;
    if (!m) { host.hidden = true; drawn = ''; return; }
    var sig = JSON.stringify(m); if (sig === drawn) return; // unchanged: leave the DOM, the fold and scroll alone
    drawn = sig;
    if (!fold) {
      fold = el('details', 'inv-merc-fold'); fold.dataset.remember = 'merc-doll';
      var sum = el('summary', '', 'Mercenary'); fold.appendChild(sum);
      body = el('div', 'inv-merc-body'); fold.appendChild(body);
      host.appendChild(fold);
      fold.open = m.state !== 'none'; // collapsed while nobody is hired; the player's own toggle is kept afterwards
    }
    host.hidden = false;
    fill(m);
    artProbe();
  }

  if (typeof window !== 'undefined' && window.__mercTest) {
    window.__mercTest({ render: render, setArt: function (on) { art.on = !!on; art.probed = true; drawn = ''; },
      reset: function () { host = fold = body = null; drawn = ''; lastView = null; art = { on: false, probed: false }; } });
    return;
  }
  document.addEventListener('d2:inventory', function (e) { render(e.detail); });
})();
