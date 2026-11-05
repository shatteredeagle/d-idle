// Watch screen. All text goes through textContent: the server sends plain text and nothing is ever parsed as HTML.
(function () {
  'use strict';
  var $ = function (id) { return document.getElementById(id); };
  var cfg = {}, answerBase = '', thoughtsAt = 0, th = { thoughts: [], speed: 1 };
  var pending = false, whyOpen = {};

  function el(tag, cls, text) {
    var e = document.createElement(tag);
    if (cls) e.className = cls;
    if (text !== undefined) e.textContent = text;
    return e;
  }
  function clear(n) { while (n.firstChild) n.removeChild(n.firstChild); }
  function mmss(s) {
    var neg = s < 0; s = Math.abs(Math.floor(s));
    var m = Math.floor(s / 60), r = s % 60;
    return (neg ? '-' : '') + m + ':' + (r < 10 ? '0' : '') + r;
  }
  function say(t) { $('msg').textContent = t || ''; $('msgbar').hidden = !t; }

  function get(path) {
    return fetch(path, { cache: 'no-store' }).then(function (r) { return r.json(); });
  }
  // All writes go to the answer route (another port), never to this origin.
  function post(path, body) {
    return fetch(answerBase + path, {
      method: 'POST', credentials: 'include',
      headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body)
    }).then(function (r) {
      return r.json().then(function (j) { if (!j.ok) say(j.error || ('error ' + r.status)); else say(''); return j; });
    }).catch(function () { say('the answer route is not reachable'); }).then(refresh);
  }

  function renderThoughts() {
    var box = $('thoughts'); clear(box);
    if (!th.thoughts.length) box.appendChild(el('p', 'hint', 'No open question. The hero carries on.'));
    th.thoughts.forEach(function (t) {
      var d = el('div', 'thought ' + t.kind); d.dataset.id = t.id;
      d.appendChild(el('div', 'q', t.kind === 'confirm' ? t.learn : t.question));
      d.appendChild(el('div', 'ctx', t.context));
      if (t.urgency) d.appendChild(el('div', 'urg', t.urgency));
      if (t.meanwhile) d.appendChild(el('div', 'meanwhile', t.meanwhile));
      var cs = el('div', 'choices');
      t.choices.forEach(function (c) {
        var b = el('button', 'btn' + (c.id === 'yes' ? ' primary' : ''), c.label); b.type = 'button';
        b.addEventListener('click', function () {
          if (t.kind === 'confirm') post('/api/confirm', { thought: t.id, yes: c.id === 'yes' });
          else post('/api/answer', { thought: t.id, answer: c.id });
        });
        cs.appendChild(b);
      });
      d.appendChild(cs);
      var bar = el('div', 'bar'), fill = el('i'); bar.appendChild(fill); d.appendChild(bar);
      var left = el('small', 'left'); d.appendChild(left);
      box.appendChild(d);
    });
    tickCountdown();
  }
  function tickCountdown() {
    var dt = (performance.now() - thoughtsAt) / 1000 * (th.speed || 1);
    var nodes = $('thoughts').querySelectorAll('.thought');
    th.thoughts.forEach(function (t, i) {
      var n = nodes[i]; if (!n) return;
      var left = Math.max(0, t.remainingSec - dt);
      n.querySelector('i').style.width = Math.min(100, left / (th.fadeSec || 45) * 100) + '%';
      n.querySelector('.left').textContent = left > 0 ? 'fades in ' + Math.ceil(left) + ' s' : 'fading';
      n.classList.toggle('fading', left <= 0);
      n.querySelectorAll('button').forEach(function (b) { b.disabled = left <= 0; });
    });
  }

  function revokeBtn(key) {
    var b = el('button', 'btn small danger', 'Revoke'); b.type = 'button';
    b.addEventListener('click', function () { post('/api/revoke', { key: key }); });
    return b;
  }
  function renderFeed(items) {
    var ul = $('feed'); clear(ul);
    items.slice().reverse().forEach(function (f) {
      var li = el('li', 'k-' + f.kind);
      li.appendChild(el('span', 't', mmss(f.t)));
      li.appendChild(el('span', '', f.text));
      (f.rules || []).forEach(function (r) {
        var s = el('div', 'rule'); s.appendChild(el('small', '', r.context + ': ' + r.label + (r.live ? ' ' : ' (revoked) ')));
        if (r.live) s.appendChild(revokeBtn(r.key));
        li.appendChild(s);
      });
      ul.appendChild(li);
    });
  }

  function renderJournal(j) {
    var ul = $('journal'); clear(ul);
    j.lines.slice().reverse().forEach(function (l) {
      var li = el('li', 'k-' + l.kind);
      li.appendChild(el('span', 't', mmss(l.t)));
      li.appendChild(el('span', '', l.text));
      if (l.why) {
        var a = el('a', '', ' why did you do that?'); a.href = '#'; a.setAttribute('role', 'button');
        a.addEventListener('click', function (ev) { ev.preventDefault(); toggleWhy(l.id); });
        li.appendChild(a);
        if (whyOpen[l.id]) li.appendChild(whyBox(whyOpen[l.id]));
      }
      (l.rules || []).forEach(function (r) {
        var s = el('div', 'rule'); s.appendChild(el('small', '', r.context + ': ' + r.label + (r.live ? ' ' : ' (revoked) ')));
        if (r.live) s.appendChild(revokeBtn(r.key));
        li.appendChild(s);
      });
      ul.appendChild(li);
    });
  }
  function whyBox(w) {
    var d = el('div', 'why', w.text || w.error || '');
    if (w.revoke) d.appendChild(revokeBtn(w.key));
    return d;
  }
  function toggleWhy(id) {
    if (whyOpen[id]) { delete whyOpen[id]; refresh(); return; }
    get('/api/why?line=' + encodeURIComponent(id)).then(function (w) { whyOpen[id] = w; refresh(); });
  }

  function renderRules(r) {
    var ul = $('rules'); clear(ul);
    if (!r.rules.length) ul.appendChild(el('li', 'hint', 'No rule learned yet.'));
    r.rules.forEach(function (x) {
      var li = el('li', 'rule' + (x.revoked ? ' revoked' : ''));
      li.appendChild(el('span', 'txt', x.text));
      li.appendChild(el('small', '', ' [' + x.risk + ']' + (x.revoked ? ' revoked' : '')));
      if (!x.revoked) { li.appendChild(document.createTextNode(' ')); li.appendChild(revokeBtn(x.key)); }
      ul.appendChild(li);
    });
    $('revokeall').disabled = r.live === 0;
  }

  function renderOrders(o) {
    var sel = $('oarea');
    if (!sel.options.length) o.areas.forEach(function (a) { var op = el('option', '', a); op.value = a; sel.appendChild(op); });
    var ul = $('orders'); clear(ul);
    o.orders.slice().reverse().forEach(function (x) {
      var li = el('li'); li.appendChild(el('span', '', x.text + ' - ' + x.state + (x.note ? ' (' + x.note + ')' : '') + ' '));
      if (x.state === 'queued' || x.state === 'active') {
        var b = el('button', 'btn small', 'Cancel'); b.type = 'button';
        b.addEventListener('click', function () { post('/api/orders/cancel', { id: x.id }); });
        li.appendChild(b);
      }
      ul.appendChild(li);
    });
  }

  function renderUptime(u) {
    $('state').textContent = u.label; $('state').className = 'tag ' + u.state;
    $('uptime').textContent = u.uptimePct === null ? '-' : u.uptimePct.toFixed(1) + '%';
    $('window').textContent = u.windowPct === null ? '' : '(last ' + Math.round(u.windowSec / 60) + ' min ' + u.windowPct.toFixed(1) + '%)';
    $('dot').className = 'dot ' + (/^(death|recovery|stall)$/.test(u.state) ? 'bad' : u.state === 'idle' ? 'idle' : 'ok');
    $('deaths').textContent = u.deaths ? u.deaths + ' death' + (u.deaths > 1 ? 's' : '') : '';
    $('deaths').hidden = !u.deaths;
    $('clock').textContent = 'time ' + mmss(u.now);
    $('uptime').title = u.text;
  }

  function refresh() {
    if (pending) return; pending = true;
    Promise.all([get('/api/thoughts'), get('/api/journal?limit=100'), get('/api/rules'), get('/api/uptime'), get('/api/orders')])
      .then(function (a) {
        th = a[0]; thoughtsAt = performance.now();
        renderThoughts(); renderFeed(th.feed); renderJournal(a[1]); renderRules(a[2]); renderUptime(a[3]); renderOrders(a[4]);
      }).catch(function () { say('the viewer is not reachable'); })
      .then(function () { pending = false; });
  }

  $('revokeall').addEventListener('click', function () {
    if (window.confirm('Revoke every rule? The hero will ask again.')) post('/api/revoke-all', { confirm: true });
  });
  $('orderform').addEventListener('submit', function (ev) {
    ev.preventDefault();
    var kind = $('okind').value;
    post('/api/orders', { kind: kind, area: kind === 'go' ? $('oarea').value : '', minutes: kind === 'home' ? 0 : parseInt($('omin').value, 10) || 0 });
  });
  $('okind').addEventListener('change', function () {
    $('oarea').disabled = $('okind').value !== 'go'; $('omin').disabled = $('okind').value === 'home';
  });

  // Phone tabs (the desktop grid shows every panel) and the theme toggle, same as the original dashboard.
  Array.prototype.forEach.call(document.querySelectorAll('#views button'), function (b) {
    b.addEventListener('click', function () {
      $('main').dataset.view = b.dataset.view;
      Array.prototype.forEach.call(document.querySelectorAll('#views button'), function (o) {
        if (o === b) o.setAttribute('aria-current', 'page'); else o.removeAttribute('aria-current');
      });
    });
  });
  var theme = 'auto';
  try { theme = localStorage.getItem('d2theme') || 'auto'; } catch (e) { /* storage blocked */ }
  function applyTheme(t) {
    if (t === 'light' || t === 'dark') document.documentElement.dataset.theme = t; else delete document.documentElement.dataset.theme;
    $('btn-theme').textContent = 'Theme: ' + (t === 'light' || t === 'dark' ? t : 'auto');
  }
  $('btn-theme').addEventListener('click', function () {
    theme = { auto: 'light', light: 'dark', dark: 'auto' }[theme] || 'auto';
    try { localStorage.setItem('d2theme', theme); } catch (e) { /* ignore */ }
    applyTheme(theme);
  });
  applyTheme(theme);

  function live() {
    var last = 0;
    function poll() { get('/api/uptime').then(function (u) { if (u.seq !== last) { last = u.seq; refresh(); } }).catch(function () {}); }
    if (window.EventSource) {
      var es = new EventSource('/api/events');
      es.addEventListener('change', refresh);
      es.onerror = function () { setTimeout(poll, 1000); };
    } else setInterval(poll, 2000);
    setInterval(poll, 5000); // safety net: a missed event costs at most 5 s
  }

  get('/api/config').then(function (c) {
    cfg = c;
    answerBase = c.answerBase || (c.answerPort ? location.protocol + '//' + location.hostname + ':' + c.answerPort : '');
    if (c.frameUrl) { $('shot').src = c.frameUrl; $('shot').hidden = false; $('noframe').hidden = true; }
    refresh(); live(); setInterval(tickCountdown, 250);
  }).catch(function () { say('the viewer is not reachable'); });
})();
