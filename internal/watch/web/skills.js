// Skills page (docs/skilltrees.md): the hero's class skill trees from GET /api/skills. Read only. Text through textContent
// only; the only inline style writes are the tooltip transform (CSSOM, allowed under style-src 'self'); tile placement is
// by class names. Icons are the user's own game art decoded at request time (/watch/art/skill-<class>-<cel>.png, see
// docs/paperdoll-art-build.md); a missing icon leaves a text tile. The tooltip is one element, refilled on hover or focus.
(function () {
  'use strict';
  var $ = function (id) { return document.getElementById(id); };
  var view = null, sig = '', tip = null, openId = null;

  function el(t, c, x) { var e = document.createElement(t); if (c) e.className = c; if (x !== undefined) e.textContent = x; return e; }
  function clear(n) { while (n.firstChild) n.removeChild(n.firstChild); }
  function say(t) { var m = $('msg'); m.textContent = t || ''; m.hidden = !t; }
  function num(x) { return Math.round(x * 100) / 100; }
  function abbr(n) { return n.split(/\s+/).map(function (w) { return w.charAt(0); }).join('').slice(0, 3).toUpperCase(); }

  // lines returns [{k, t}] for one node: the same facts the page tiles show plus what the tables give.
  function lines(n) {
    var L = [], add = function (k, t) { if (t) L.push({ k: k, t: t }); };
    add('desc', n.short);
    add('desc', n.long && n.long !== n.short ? n.long : '');
    add('type', n.passive ? 'Passive' : (n.aura ? 'Aura' : ''));
    add('req', 'Required level: ' + n.reqLevel);
    (n.prereqs || []).forEach(function (p) { add(p.have ? 'req' : 'bad', 'Requires: ' + p.name + (p.have ? '' : ' (not learned)')); });
    var cur = 'Skill level: ' + n.base + (n.bonus ? ' (+' + n.bonus + ' from items, level ' + n.level + ')' : '') + ' of ' + n.maxLevel;
    add(n.base || n.bonus ? 'good' : 'dim', cur);
    if (n.manaNow !== undefined) add('base', 'Mana cost now: ' + num(n.manaNow));
    if (n.manaNext !== undefined) add('base', 'Mana cost at level ' + (n.level + 1) + ': ' + num(n.manaNext));
    if (n.synergy && n.synergy.length) add('dim', 'Synergies: ' + n.synergy.join(', '));
    if (!n.canAdd && n.why && n.base === 0) add('bad', 'Cannot learn yet: ' + n.why);
    return L;
  }

  function tipEl() {
    if (tip) return tip;
    tip = el('div', 'inv-tip sk-tip'); tip.id = 'sk-tip'; tip.setAttribute('role', 'tooltip'); tip.hidden = true;
    document.body.appendChild(tip);
    return tip;
  }
  function showTip(tile, n) {
    var t = tipEl();
    clear(t);
    t.appendChild(el('div', 'tip-name q-normal', n.name));
    lines(n).forEach(function (l) { t.appendChild(el('div', 'tip-line', l.t)).setAttribute('data-k', l.k); });
    t.hidden = false; openId = n.id;
    tile.setAttribute('aria-describedby', 'sk-tip');
    var r = tile.getBoundingClientRect(), w = t.offsetWidth, h = t.offsetHeight, vw = document.documentElement.clientWidth, vh = window.innerHeight, m = 8;
    var x = r.right + m; if (x + w > vw - m) x = r.left - m - w; if (x < m) { x = Math.max(m, Math.min(vw - w - m, r.left)); }
    var y = Math.max(m, Math.min(vh - h - m, r.top));
    if (x < r.right && x + w > r.left) y = r.bottom + m + h < vh ? r.bottom + m : Math.max(m, r.top - m - h); // overlaps the tile: below or above
    t.style.transform = 'translate(' + Math.round(x) + 'px,' + Math.round(y) + 'px)';
  }
  function hideTip() {
    if (tip) tip.hidden = true;
    openId = null;
    var d = document.querySelectorAll('[aria-describedby="sk-tip"]');
    for (var i = 0; i < d.length; i++) d[i].removeAttribute('aria-describedby');
  }

  function tile(n, allocOn) {
    var d = el('div', 'sk-tile sk-r' + n.row + ' sk-c' + n.col);
    d.tabIndex = 0; d.setAttribute('role', 'group'); d.setAttribute('data-state', n.state); d.setAttribute('data-id', n.id);
    d.setAttribute('aria-label', n.name + ', level ' + n.base + (n.bonus ? ' plus ' + n.bonus : '') + ' of ' + n.maxLevel + (n.state === 'locked' ? ', locked' : ''));
    if (allocOn && n.canAdd) d.setAttribute('data-act', '1');
    var ab = el('span', 'sk-ab', n.icon ? '' : abbr(n.name));
    if (n.icon) {
      var img = new Image(); img.alt = ''; img.decoding = 'async'; img.loading = 'lazy'; img.src = '/watch/art/' + n.icon + '.png';
      img.addEventListener('error', function () { img.remove(); ab.textContent = abbr(n.name); });
      d.appendChild(img);
    }
    d.appendChild(ab);
    if (n.level > 0) d.appendChild(el('span', 'sk-lv', n.bonus ? n.base + '+' + n.bonus : String(n.base)));
    d.addEventListener('mouseenter', function () { showTip(d, n); });
    d.addEventListener('mouseleave', hideTip);
    d.addEventListener('focus', function () { showTip(d, n); });
    d.addEventListener('blur', hideTip);
    d.addEventListener('click', function (e) {
      if (d.getAttribute('data-act') === '1') { allocate(n.id); return; }
      if (e.pointerType === 'touch' || e.pointerType === 'pen') { if (openId === n.id) hideTip(); else showTip(d, n); }
    });
    return d;
  }

  // The allocation control is disabled today: /api/skills says allocate:false and the route answers 501.
  function allocate(id) {
    fetch('/api/config', { cache: 'no-store' }).then(function (r) { return r.json(); }).then(function (c) {
      var u = c.answerBase ? c.answerBase : location.protocol + '//' + location.hostname + ':' + c.answerPort;
      return fetch(u + '/api/skills/allocate', { method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ skill: id, points: 1 }) });
    }).then(function (r) { return r.json().then(function (j) { if (!r.ok) throw new Error(j.error || ('error ' + r.status)); say(''); load(true); }); })
      .catch(function (e) { say(e.message); });
  }

  var SVGNS = 'http://www.w3.org/2000/svg';
  // arrows draws one line per prerequisite, from the centre bottom of the required tile to the centre top of the skill.
  function arrows(board, tree) {
    var old = board.querySelector('.sk-arrows'); if (old) old.remove();
    var svg = document.createElementNS(SVGNS, 'svg'); svg.setAttribute('class', 'sk-arrows'); svg.setAttribute('aria-hidden', 'true');
    var br = board.getBoundingClientRect(), pos = {};
    var tiles = board.querySelectorAll('.sk-tile');
    for (var i = 0; i < tiles.length; i++) { var r = tiles[i].getBoundingClientRect(); pos[tiles[i].getAttribute('data-id')] = { cx: r.left - br.left + r.width / 2, top: r.top - br.top, bot: r.bottom - br.top }; }
    tree.nodes.forEach(function (n) {
      (n.prereqs || []).forEach(function (p) {
        var a = pos[p.id], b = pos[n.id]; if (!a || !b) return; // a prerequisite in another tree has no arrow here
        var ln = document.createElementNS(SVGNS, 'line'), have = p.have ? '1' : '0';
        ln.setAttribute('x1', a.cx); ln.setAttribute('y1', a.bot); ln.setAttribute('x2', b.cx); ln.setAttribute('y2', b.top - 3); ln.setAttribute('data-have', have);
        svg.appendChild(ln);
        var hd = document.createElementNS(SVGNS, 'polygon'); hd.setAttribute('data-have', have);
        hd.setAttribute('points', (b.cx - 4) + ',' + (b.top - 7) + ' ' + (b.cx + 4) + ',' + (b.top - 7) + ' ' + b.cx + ',' + (b.top - 1));
        svg.appendChild(hd);
      });
    });
    board.insertBefore(svg, board.firstChild);
  }

  function render(v) {
    var host = $('trees'); clear(host); hideTip();
    var meta = $('summeta');
    if (!v.available) {
      meta.textContent = '';
      host.appendChild(el('p', 'd-hint', 'No skill trees yet: the hero or its class is not known.'));
      return;
    }
    meta.textContent = v.class + ', level ' + v.level + ' | unspent points: ' + v.unspent + ' | spent: ' + v.spent;
    v.trees.forEach(function (tr, i) {
      var card = el('section', 'd-card'), head = el('div', 'd-card-head'), h = el('h2', '', 'Tree ' + (i + 1));
      h.id = 'tree-' + tr.tab; card.setAttribute('aria-labelledby', h.id);
      head.appendChild(h); head.appendChild(el('span', 'meta', tr.spent + ' point' + (tr.spent === 1 ? '' : 's') + ' spent'));
      card.appendChild(head);
      var board = el('div', 'sk-board'); board.setAttribute('role', 'group'); board.setAttribute('aria-label', 'Skills of tree ' + (i + 1));
      tr.nodes.forEach(function (n) { board.appendChild(tile(n, v.allocate)); });
      card.appendChild(board); host.appendChild(card);
      arrows(board, tr);
      board._tree = tr;
    });
  }
  function redraw() {
    var bs = document.querySelectorAll('.sk-board');
    for (var i = 0; i < bs.length; i++) if (bs[i]._tree) arrows(bs[i], bs[i]._tree);
  }

  function load(force) {
    return fetch('/api/skills', { cache: 'no-store' }).then(function (r) {
      if (!r.ok) throw new Error(r.status === 503 ? 'The skills page is off or the hero is not known yet.' : 'Could not read the skill trees (' + r.status + ').');
      return r.text();
    }).then(function (txt) {
      say('');
      if (txt === sig && !force) return; // unchanged: no rewrite, no reflow, an open tooltip stays
      sig = txt; view = JSON.parse(txt); render(view);
    }).catch(function (e) { say(e.message); });
  }
  document.addEventListener('keydown', function (e) { if (e.key === 'Escape') hideTip(); });
  document.addEventListener('click', function (e) { if (!e.target.closest || !e.target.closest('.sk-tile')) hideTip(); });
  window.addEventListener('resize', redraw);
  load(); setInterval(load, 5000);
})();
