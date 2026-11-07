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
    }).catch(function () { say('the answer route is not reachable'); }).then(function (j) { refresh(); return j; });
  }

  function renderThoughts() {
    var box = $('thoughts'); clear(box);
    if (!th.thoughts.length) box.appendChild(el('p', 'hint', 'No open question. The hero carries on.'));
    th.thoughts.forEach(function (t) {
      var d = el('div', 'thought ' + t.kind); d.dataset.id = t.id;
      // headline = what happened; then the urgency, what happens on no answer, the buttons (plain phrase + id), the rule footnote
      d.appendChild(el('div', 'q', t.kind === 'confirm' ? t.learn : (t.headline || t.question)));
      if (t.kind !== 'confirm' && t.headline && t.headline !== t.question) d.appendChild(el('div', 'ask', t.question));
      if (t.whyNow) d.appendChild(el('div', 'urg', t.whyNow));
      else if (t.urgency) d.appendChild(el('div', 'urg', t.urgency));
      if (t.ifNothing) d.appendChild(el('div', 'ifnothing', t.ifNothing));
      else if (t.meanwhile) d.appendChild(el('div', 'meanwhile', t.meanwhile));
      var cs = el('div', 'choices');
      t.choices.forEach(function (c) {
        var b = el('button', 'btn' + (c.id === 'yes' ? ' primary' : ''), ''); b.type = 'button';
        b.appendChild(el('span', 'lbl', c.label));
        if (c.does) b.appendChild(el('small', 'does', c.does));
        b.title = c.id;
        b.addEventListener('click', function () {
          if (t.kind === 'confirm') post('/api/confirm', { thought: t.id, yes: c.id === 'yes' });
          else post('/api/answer', { thought: t.id, answer: c.id });
        });
        cs.appendChild(b);
      });
      d.appendChild(cs);
      if (t.rule) d.appendChild(el('small', 'rulenote', t.rule));
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

  // ---- Orders: Hero now, map, queue, sheet (docs/research/orders-ux-design.md). All text via textContent. ----
  var om = null, ord = { orders: [] }, ordAt = 0, mapList = false, pend = null, sheetFrom = null, toastTimer = 0;
  var DURS = [['10 min', 10], ['30 min', 30], ['1 h', 60], ['Until I cancel', 0]];

  function nodeById(id) { for (var i = 0; i < om.nodes.length; i++) if (om.nodes[i].id === id) return om.nodes[i]; return null; }
  function nodeByOrder(name) {
    for (var i = 0; i < om.nodes.length; i++) if (om.nodes[i].order && om.nodes[i].order.toLowerCase() === String(name).toLowerCase()) return om.nodes[i];
    return null;
  }
  function heroNode() {
    for (var i = 0; i < om.nodes.length; i++) if (om.nodes[i].status.here) return om.nodes[i];
    return null;
  }
  // The route is the shortest path over the map edges from where the hero is (else the town) to the node. It is the
  // schematic path, not a promise: the bot's own navigator picks the real way (U).
  function path(toId) {
    var from = heroNode(), start = from ? from.id : om.town;
    var adj = {};
    om.edges.forEach(function (e) { (adj[e.from] = adj[e.from] || []).push(e.to); (adj[e.to] = adj[e.to] || []).push(e.from); });
    var prev = {}; prev[start] = null; var q = [start];
    while (q.length) {
      var c = q.shift();
      if (c === toId) break;
      (adj[c] || []).forEach(function (n) { if (!(n in prev)) { prev[n] = c; q.push(n); } });
    }
    if (!(toId in prev)) return [toId];
    var p = []; for (var x = toId; x !== null; x = prev[x]) p.unshift(x);
    return p;
  }
  function crumbs(ids, here) {
    var d = el('div', 'crumbs'); d.setAttribute('aria-label', 'Route');
    var hi = here ? ids.indexOf(here.id) : -1;
    ids.forEach(function (id, i) {
      if (i) d.appendChild(el('span', 'muted', '›'));
      var n = nodeById(id), st = hi < 0 ? '' : i < hi ? ' done' : i === hi ? ' now' : '';
      d.appendChild(el('span', 's' + st, (st === ' done' ? '✓ ' : st === ' now' ? '▶ ' : '') + (n ? n.name : id)));
    });
    return d;
  }
  function stWords(n) {
    var s = n.status, w = [];
    if (s.here) w.push('◆ here');
    if (s.order) w.push('▶ order');
    if (s.queued) w.push('#' + s.queued);
    if (s.hard) w.push('⚠ hard');
    if (s.cleared) w.push('✓ cleared');
    return w;
  }
  function activeOrder() { for (var i = 0; i < ord.orders.length; i++) if (ord.orders[i].state === 'active') return ord.orders[i]; return null; }
  function lastClosed() {
    for (var i = ord.orders.length - 1; i >= 0; i--) if (/^(done|failed|cancelled)$/.test(ord.orders[i].state)) return ord.orders[i];
    return null;
  }
  function toast(m) {
    var t = $('toast'); t.textContent = m; t.hidden = false;
    clearTimeout(toastTimer); toastTimer = setTimeout(function () { t.hidden = true; }, 3000);
  }
  function elapsedSec(o) { return Math.max(0, ord.now - o.startedAt + (performance.now() - ordAt) / 1000 * (th.speed || 1)); }

  function renderNow() {
    var n = $('now'); clear(n);
    if (!om) return;
    var h = el('div', 'card-head'), t = el('h2', '', 'Hero now'); t.id = 't-now'; h.appendChild(t); n.appendChild(h);
    var o = activeOrder(), here = heroNode();
    var act = om.hero.activity ? om.hero.activity.charAt(0).toUpperCase() + om.hero.activity.slice(1) : '';
    if (!o) {
      var named = here ? 'In ' + here.name + '.' : 'Position unknown.';
      n.appendChild(el('p', '', 'Following its own plan. ' + named + (act ? ' ' + act + '.' : '') + ' No order. Tap an area on the map to send the hero.'));
      var lc = lastClosed();
      if (lc) n.appendChild(el('div', 'last', 'Last order: ' + lc.text + ' - ' + lc.state + (lc.note ? ' (' + lc.note + ')' : '') + '.'));
      var qk = el('div', 'quick'), hb = el('button', 'btn', 'Rest in town'); hb.type = 'button';
      hb.addEventListener('click', function () { send({ kind: 'home' }); });
      qk.appendChild(hb); n.appendChild(qk);
      return;
    }
    n.appendChild(el('p', '', '')).appendChild(el('b', '', o.text));
    if (act) n.appendChild(el('div', 'muted', act));
    var el0 = elapsedSec(o), eta = '', pct = null;
    if (o.forever) eta = 'runs until you cancel';
    else if (o.minutes) { pct = Math.min(100, Math.round(el0 / (o.minutes * 60) * 100)); eta = 'about ' + Math.max(0, Math.ceil(o.minutes - el0 / 60)) + ' min left'; }
    if (pct !== null) {
      var bar = el('div', 'bar progress'), fill = el('i'); fill.style.width = pct + '%';
      bar.setAttribute('role', 'progressbar'); bar.setAttribute('aria-valuemin', 0); bar.setAttribute('aria-valuemax', 100);
      bar.setAttribute('aria-valuenow', pct); bar.setAttribute('aria-label', 'Order progress');
      bar.appendChild(fill); n.appendChild(bar);
      n.appendChild(el('div', '', pct + '% - ' + eta));
    } else n.appendChild(el('div', '', eta));
    n.appendChild(el('div', 'muted', (o.kills || 0) + ' kills, +' + (o.xpGained || 0) + ' xp since the order'));
    var target = o.kind === 'home' ? nodeById(om.town) : o.kind === 'go' ? nodeByOrder(o.area) : here;
    if (target) n.appendChild(crumbs(path(target.id), here));
    var qk2 = el('div', 'quick');
    var cb = el('button', 'btn danger', 'Cancel order'); cb.type = 'button';
    cb.addEventListener('click', function () { post('/api/orders/cancel', { id: o.id }).then(function (j) { if (j && j.ok) toast('Cancelled: ' + o.text); }); });
    var hm = el('button', 'btn', 'Come home'); hm.type = 'button';
    hm.setAttribute('aria-label', 'Cancel the order and rest in town');
    hm.addEventListener('click', function () { send({ kind: 'home' }); });
    qk2.appendChild(cb); qk2.appendChild(hm); n.appendChild(qk2);
  }

  function renderQueue() {
    var n = $('queue'); clear(n);
    if (!om) return;
    var qd = ord.orders.filter(function (x) { return x.state === 'queued'; });
    var h = el('div', 'card-head'), t = el('h2', '', 'Up next (' + qd.length + ')'); t.id = 't-q'; h.appendChild(t); n.appendChild(h);
    if (!qd.length) { n.appendChild(el('p', 'muted', 'Nothing queued. "Go now" replaces the current order; a real queue (Add to queue) comes in a later update.')); return; }
    var ul = el('ol', 'q-list');
    qd.forEach(function (x, i) {
      var li = el('li'); li.appendChild(el('span', 'badge', String(i + 1)));
      li.appendChild(el('span', 't2', x.text));
      var c = el('button', 'btn small danger', '✕'); c.type = 'button'; c.setAttribute('aria-label', 'Cancel: ' + x.text);
      c.addEventListener('click', function () { post('/api/orders/cancel', { id: x.id }); });
      li.appendChild(c); ul.appendChild(li);
    });
    n.appendChild(ul);
  }

  var SVGNS = $('svg').namespaceURI; // read from the page's own <svg>: no URL literal in the script
  function sv(tag, at) { var e = document.createElementNS(SVGNS, tag); for (var k in at) e.setAttribute(k, at[k]); return e; }
  function nodeLabel(n) {
    var w = stWords(n);
    var l = n.name + (n.kind === 'dungeon' ? ', dungeon' : '') + (w.length ? ', ' + w.join(', ') : '');
    if (n.status.hard) l += ', too hard now: ' + n.status.hardWhy;
    if (!n.orderable) l += ', not orderable yet';
    return l;
  }
  function renderMap() {
    var svg = $('svg'), ml = $('maplist'); clear(svg); clear(ml);
    if (!om) return;
    $('mapnote').textContent = om.populated ? 'Act 1 only today. Acts 2 to 5 appear here from data once their areas can be ordered.' : 'No map data for this act yet.';
    $('svg').setAttribute('viewBox', om.box.x + ' ' + om.box.y + ' ' + om.box.w + ' ' + om.box.h);
    var by = {}; om.nodes.forEach(function (n) { by[n.id] = n; });
    om.edges.forEach(function (e) {
      var a = by[e.from], b = by[e.to]; if (!a || !b) return;
      svg.appendChild(sv('line', { x1: a.x, y1: a.y, x2: b.x, y2: b.y, 'class': 'edge' + (e.dungeon ? ' dg' : '') }));
    });
    om.nodes.forEach(function (n) {
      var s = n.status, cls = 'node' + (n.kind === 'town' ? ' town' : '') + (n.kind === 'dungeon' ? ' dungeon' : '') + (n.orderable ? '' : ' locked') +
        (s.here ? ' hero' : '') + (s.order ? ' current' : '') + (s.hard ? ' hard' : '');
      var g = sv('g', { 'class': cls, tabindex: 0, role: 'button', 'aria-label': nodeLabel(n), transform: 'translate(' + (n.x - 52) + ',' + (n.y - 22) + ')' });
      g.appendChild(sv('rect', { 'class': 'b', width: 104, height: 44, rx: 8 }));
      var words = stWords(n).slice(0, 2).join(' '), sub = n.sub || words;
      var t1 = sv('text', { x: 52, y: sub ? 18 : 27, 'text-anchor': 'middle' });
      if (n.name.length > 12) t1.setAttribute('font-size', '10.5');
      if (n.name.length > 16) t1.setAttribute('font-size', '9.5');
      t1.textContent = n.name; g.appendChild(t1);
      if (sub) { var t2 = sv('text', { x: 52, y: 34, 'text-anchor': 'middle', 'class': 'st' }); if (sub.length > 15) t2.setAttribute('font-size', '9.5'); t2.textContent = sub; g.appendChild(t2); }
      var act = function () { tap(n, g); };
      g.addEventListener('click', act);
      g.addEventListener('keydown', function (ev) { if (ev.key === 'Enter' || ev.key === ' ') { ev.preventDefault(); act(); } });
      svg.appendChild(g);
    });
    var ul = el('ul');
    om.nodes.forEach(function (n) {
      var li = el('li'), b = el('button', '', ''); b.type = 'button';
      b.appendChild(el('span', '', n.name + (n.kind === 'dungeon' ? ' (dungeon)' : '')));
      b.appendChild(el('span', 'muted', n.orderable ? stWords(n).join(' ') : 'not orderable yet'));
      b.setAttribute('aria-label', nodeLabel(n));
      b.addEventListener('click', function () { tap(n, b); });
      li.appendChild(b); ul.appendChild(li);
    });
    ml.appendChild(ul);
    svg.hidden = mapList; ml.hidden = !mapList;
  }

  // A redraw keeps keyboard focus: the same index of button in the same card gets it back.
  function renderOrders() {
    var ae = document.activeElement, keep = null;
    ['now', 'queue', 'maparea'].forEach(function (id) {
      var box = $(id), items = box.querySelectorAll('button, [role=button]');
      for (var i = 0; i < items.length; i++) if (items[i] === ae) keep = [id, i];
    });
    renderNow(); renderQueue(); renderMap();
    if (keep) {
      var items2 = $(keep[0]).querySelectorAll('button, [role=button]');
      if (items2[keep[1]] && $('sheet').hidden) items2[keep[1]].focus();
    }
  }

  // ---- the order sheet ----
  function summary() {
    if (pend.node.kind === 'town') return 'Summary: rest in town.';
    return 'Summary: farm ' + pend.node.name + (pend.mins ? ' for ' + pend.mins + ' min' : ' until you cancel') + '.';
  }
  function tap(n, from) {
    if (!n.orderable) { toast(n.name + ' is not orderable yet.'); return; }
    pend = { node: n, mins: 30 };
    sheetFrom = from;
    drawSheet();
  }
  function chip(txt, on, fn) {
    var b = el('button', 'chip', txt); b.type = 'button'; b.setAttribute('aria-pressed', on ? 'true' : 'false');
    b.addEventListener('click', fn); return b;
  }
  function drawSheet() {
    var n = pend.node, sh = $('sheet'); clear(sh);
    var hd = el('div', 'card-head'), t = el('h2', '', n.kind === 'town' ? 'Rest in town' : 'Send the hero to ' + n.name); t.id = 'sh-title'; hd.appendChild(t);
    var x = el('button', 'btn', 'Close'); x.type = 'button'; x.addEventListener('click', closeSheet); hd.appendChild(x); sh.appendChild(hd);
    var cur = activeOrder();
    if (n.kind === 'town') {
      sh.appendChild(el('p', '', 'The hero walks to town and rests. The current order, if any, is replaced.'));
    } else {
      if (n.range || n.why) sh.appendChild(el('p', 'muted', 'Veterans: ' + [n.range, n.why].filter(Boolean).join(' - ')));
      if (n.status.hard) {
        var wb = el('div', 'warnbox'); wb.setAttribute('role', 'alert');
        wb.appendChild(el('b', '', '⚠ Too hard for the hero now. '));
        wb.appendChild(document.createTextNode('The hero’s guard says: ' + n.status.hardWhy + '. You may still confirm: the order only asks, the hero’s own guard has the last word and will turn the order down if it is still unsafe when the hero gets there. The journal will say so.'));
        sh.appendChild(wb);
      } else if (n.advice) sh.appendChild(el('p', 'warn', '⚠ ' + n.advice));
      sh.appendChild(el('h3', 'sub-h', 'For how long'));
      var cs = el('div', 'chips');
      DURS.forEach(function (d) { cs.appendChild(chip(d[0], pend.mins === d[1], function () { pend.mins = d[1]; drawSheet(); })); });
      sh.appendChild(cs);
      sh.appendChild(el('h3', 'sub-h', 'Route'));
      sh.appendChild(crumbs(path(n.id), null));
    }
    sh.appendChild(el('p', 'muted', summary()));
    var qk = el('div', 'quick'), go = el('button', 'btn primary', n.kind === 'town' ? 'Confirm' : 'Go now'); go.type = 'button';
    go.addEventListener('click', function () { confirmOrder(); });
    qk.appendChild(go); sh.appendChild(qk);
    if (cur && n.kind !== 'town') sh.appendChild(el('p', 'muted', '"Go now" replaces the current order (' + cur.text + ').'));
    $('scrim').hidden = false; sh.hidden = false;
    var f = sh.querySelector('.chip[aria-pressed="true"]') || sh.querySelector('.btn.primary'); if (f) f.focus();
  }
  function closeSheet() {
    $('sheet').hidden = true; $('scrim').hidden = true; pend = null;
    if (sheetFrom && sheetFrom.focus) { try { sheetFrom.focus(); } catch (e) { /* node gone */ } }
  }
  function confirmOrder() {
    var n = pend.node;
    var body = n.kind === 'town' ? { kind: 'home' } : pend.mins ? { kind: 'go', area: n.order, minutes: pend.mins } : { kind: 'go', area: n.order, until: 'forever' };
    closeSheet();
    send(body);
  }
  // "Go now": the new order replaces every open one (slice 1 has no queue).
  function send(body) {
    body.replace = true;
    return post('/api/orders', body).then(function (j) { if (j && j.ok && j.order) toast('Order given: ' + j.order.text); });
  }
  function trapTab(ev) {
    if (ev.key === 'Escape' && !$('sheet').hidden) { closeSheet(); return; }
    if (ev.key !== 'Tab' || $('sheet').hidden) return;
    var f = $('sheet').querySelectorAll('button:not([disabled])');
    if (!f.length) return;
    var first = f[0], last = f[f.length - 1];
    if (ev.shiftKey && document.activeElement === first) { ev.preventDefault(); last.focus(); }
    else if (!ev.shiftKey && document.activeElement === last) { ev.preventDefault(); first.focus(); }
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

  // ---- Currencies (d.idle mode only): header strip and the spend sheet shell. Nothing here buys anything yet. ----
  function renderCurrency(c) {
    var strip = $('curstrip'); clear(strip);
    strip.hidden = false; $('tab-spend').hidden = !c.enabled; $('spendpane').hidden = !c.enabled;
    if (!c.enabled) { strip.appendChild(el('span', 'off', c.text)); return; }
    if (c.error) strip.appendChild(el('span', 'off', c.error));
    c.currencies.forEach(function (r) {
      if (r.where.indexOf('header') < 0) return;
      var s = el('span', 'cur'); s.title = r.blurb;
      s.appendChild(document.createTextNode(r.name + ' '));
      s.appendChild(el('b', '', r.amount === null ? '-' : String(r.amount)));
      s.appendChild(document.createTextNode(' ' + r.label));
      strip.appendChild(s);
    });
    $('spendtext').textContent = c.text;
    var ul = $('spendlist'); clear(ul);
    if (!c.spend.length) ul.appendChild(el('li', 'empty', 'Nothing to buy yet.'));
    c.spend.forEach(function (u) {
      var li = el('li', 'spendrow'), m = el('div', 'sp-main');
      m.appendChild(el('div', 'sp-name', u.name + (u.tier ? ' (' + u.tier + ')' : '') + '  level ' + u.level + ' of ' + u.maxLevel));
      m.appendChild(el('div', 'sp-effect', u.effect));
      if (u.why) m.appendChild(el('div', 'sp-why', u.why));
      li.appendChild(m);
      li.appendChild(el('span', 'sp-cost', u.cost === null ? '-' : u.cost + ' ' + u.currency));
      var b = el('button', 'btn small', 'Buy'); b.type = 'button'; b.disabled = !u.enabled;
      li.appendChild(b); ul.appendChild(li);
    });
  }

  function refresh() {
    if (pending) return; pending = true;
    var reqs = [get('/api/thoughts'), get('/api/journal?limit=100'), get('/api/rules'), get('/api/uptime'), get('/api/orders'), get('/api/orders/map')];
    if (cfg.currency) reqs.push(get('/api/currency'));
    Promise.all(reqs)
      .then(function (a) {
        if (a[6]) renderCurrency(a[6]);
        th = a[0]; thoughtsAt = performance.now();
        renderThoughts(); renderFeed(th.feed); renderJournal(a[1]); renderRules(a[2]); renderUptime(a[3]); ord = a[4]; ordAt = performance.now(); om = a[5]; renderOrders();
      }).catch(function () { say('the viewer is not reachable'); })
      .then(function () { pending = false; });
  }

  $('revokeall').addEventListener('click', function () {
    if (window.confirm('Revoke every rule? The hero will ask again.')) post('/api/revoke-all', { confirm: true });
  });
  $('scrim').addEventListener('click', closeSheet);
  document.addEventListener('keydown', trapTab);
  $('viewtoggle').addEventListener('click', function () {
    mapList = !mapList; this.setAttribute('aria-pressed', mapList ? 'true' : 'false'); this.textContent = mapList ? 'Map view' : 'List view'; renderMap();
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
  // #orders, #thoughts, ... opens that tab (a link, a screenshot); the default is Orders.
  var hv = location.hash.slice(1), hb = document.querySelector('#views button[data-view="' + hv.replace(/[^a-z]/g, '') + '"]');
  if (hb && !hb.hidden) hb.click();
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

  // Game video: an MJPEG <img>. It shows only when the server offers a stream, and reconnects when the stream ends or
  // is refused (viewer limit, capture restart). The token, if any, rides along from this page's own address.
  function video(url) {
    var shot = $('shot'), t = new URLSearchParams(location.search).get('t'), tries = 0;
    $('gamepane').hidden = false; $('tab-game').hidden = false;
    function connect() {
      var q = [];
      if (t) q.push('t=' + encodeURIComponent(t));
      q.push('r=' + Date.now());
      shot.src = url + '?' + q.join('&');
    }
    shot.addEventListener('load', function () { tries = 0; shot.hidden = false; $('noframe').hidden = true; });
    shot.addEventListener('error', function () {
      shot.hidden = true; $('noframe').hidden = false;
      tries++; setTimeout(connect, Math.min(15000, 1000 * tries));
    });
    $('noframe').hidden = false;
    connect();
  }

  get('/api/config').then(function (c) {
    cfg = c;
    answerBase = c.answerBase || (c.answerPort ? location.protocol + '//' + location.hostname + ':' + c.answerPort : '');
    if (c.frameUrl) video(c.frameUrl);
    refresh(); live(); setInterval(tickCountdown, 250);
    setInterval(function () { if (activeOrder()) { if (!pending) refresh(); } }, 5000);
  }).catch(function () { say('the viewer is not reachable'); });
})();
