// Gear condition strip (docs/repair.md): one compact line under the header, per worn slot current/max durability, amber
// under the warn percent, red when broken, and the Repair now button. Polls GET /api/repair; hidden until the bot answers.
(function () {
  'use strict';
  var top = document.querySelector('header.top');
  if (!top || document.getElementById('gearstrip')) return;
  var box = document.createElement('div');
  box.id = 'gearstrip'; box.setAttribute('role', 'status'); box.setAttribute('aria-label', 'Gear condition'); box.hidden = true;
  top.insertAdjacentElement('afterend', box);

  function el(t, c, x) { var e = document.createElement(t); if (c) e.className = c; if (x !== undefined) e.textContent = x; return e; }
  function base() {
    return fetch('/api/config', { cache: 'no-store' }).then(function (r) { return r.json(); }).then(function (c) {
      return c.answerBase ? c.answerBase : location.protocol + '//' + location.hostname + ':' + c.answerPort;
    });
  }
  // The last answer to a click stays on screen until the next click: an error must not vanish with the next redraw.
  var feedback = '', busy = false;
  function post(path) {
    return base().then(function (u) {
      return fetch(u + path, { method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' }, body: '{}' });
    }).then(function (r) {
      return r.json().catch(function () { return {}; }).then(function (j) {
        if (!r.ok || !j.ok) throw new Error((j && j.error) || ('the answer route said HTTP ' + r.status));
        return j;
      });
    });
  }
  function reqText(q) {
    if (!q) return '';
    if (q.state === 'requested') {
      if (q.short) return 'Repair requested, waiting for gold: need ' + q.short + ' more (' + q.need + ' in all).';
      return 'Repair requested: done at the next town visit.' + (q.note ? ' (' + q.note + ')' : '');
    }
    if (q.state === 'done') return 'Repair done.';
    return '';
  }
  function draw(v) {
    box.textContent = '';
    var gear = v.gear || [];
    if (!gear.length) { box.hidden = true; return; }
    box.hidden = false;
    box.appendChild(el('b', '', 'Gear'));
    gear.forEach(function (g) {
      var s = el('span', 'gslot' + (g.broken ? ' broken' : g.warn ? ' warn' : ''));
      s.appendChild(el('span', 'gname', g.slot));
      s.appendChild(el('span', 'gdur', g.dur + '/' + g.max));
      if (g.broken) s.appendChild(el('span', 'gstate', 'BROKEN'));
      s.title = g.name + ' ' + g.pct + '%';
      box.appendChild(s);
    });
    if (v.estimate) box.appendChild(el('span', 'gnote', 'repair costs at least ' + v.estimate + ' gold'));
    var pending = v.request && v.request.state === 'requested';
    var tb = v.trip && v.trip.button; // schema 2 (docs/repair.md, the repair trip): the button and the line come from the bot
    var cancel = tb ? tb.action === 'cancel' : pending;
    var b = el('button', 'btn small', tb ? tb.label : (pending ? 'Cancel repair' : 'Repair now')); b.type = 'button';
    if (tb && !tb.enabled) { b.disabled = true; b.title = tb.reason || ''; }
    b.addEventListener('click', function () {
      if (busy || (tb && !tb.enabled)) return;
      busy = true; b.disabled = true; feedback = 'Sending...';
      post(cancel ? '/api/repair/cancel' : '/api/repair/request').then(function () {
        feedback = cancel ? 'Repair request cancelled.' : 'Repair requested: the hero goes at the next safe moment.';
      }, function (e) {
        feedback = 'Not sent: ' + e.message;
      }).then(function () { busy = false; load(); });
    });
    box.appendChild(b);
    var t = v.trip ? v.trip.line : reqText(v.request); if (t) box.appendChild(el('span', 'gnote', t));
    if (feedback) box.appendChild(el('span', 'gnote gfeedback', feedback));
    if (v.worst) box.appendChild(el('span', 'gnote', v.worst));
  }
  function load() {
    fetch('/api/repair', { cache: 'no-store', credentials: 'same-origin' }).then(function (r) {
      if (!r.ok) throw new Error(String(r.status));
      return r.json();
    }).then(draw, function (e) { box.hidden = false; box.textContent = ''; box.appendChild(el('span', 'gnote', 'Gear status unavailable: ' + e.message)); });
  }
  load(); setInterval(load, 5000);
})();
