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
  function post(path) {
    return base().then(function (u) {
      return fetch(u + path, { method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' }, body: '{}' });
    }).then(function (r) { return r.json(); });
  }
  function reqText(q) {
    if (!q) return '';
    if (q.state === 'requested') return 'Repair requested: done at the next town visit.' + (q.note ? ' (' + q.note + ')' : '');
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
    var b = el('button', 'btn small', pending ? 'Cancel repair' : 'Repair now'); b.type = 'button';
    b.addEventListener('click', function () {
      b.disabled = true;
      post(pending ? '/api/repair/cancel' : '/api/repair/request').then(function (j) {
        if (!j.ok) box.appendChild(el('span', 'gnote', j.error || 'failed'));
        else load();
      }).catch(function () { b.disabled = false; });
    });
    box.appendChild(b);
    var t = reqText(v.request); if (t) box.appendChild(el('span', 'gnote', t));
    if (v.worst) box.appendChild(el('span', 'gnote', v.worst));
  }
  function load() {
    fetch('/api/repair', { cache: 'no-store', credentials: 'same-origin' }).then(function (r) {
      if (!r.ok) throw new Error(String(r.status));
      return r.json();
    }).then(draw, function () { box.hidden = true; });
  }
  load(); setInterval(load, 5000);
})();
