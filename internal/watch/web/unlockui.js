// Unlockable panels (docs/unlock-ladder.md). View only: polls GET /api/unlocks and the routes of the open features.
// A locked feature shows as a silhouette with its teaser; an open one gets its panel. All text goes through textContent.
(function () {
  'use strict';
  var $ = function (id) { return document.getElementById(id); };
  function el(tag, cls, text) { var e = document.createElement(tag); if (cls) e.className = cls; if (text !== undefined) e.textContent = text; return e; }
  function clear(n) { while (n.firstChild) n.removeChild(n.firstChild); }
  function get(path) {
    return fetch(path, { cache: 'no-store', credentials: 'same-origin' }).then(function (r) { if (!r.ok) throw new Error(String(r.status)); return r.json(); });
  }
  var timer = 0, fails = 0, chronN = 0, answerBase = '', muted = false;

  function silhouettes(u) {
    var list = $('lockedlist'); clear(list);
    var locked = (u.locked || []).filter(function (s) { return s.group === 'ui'; });
    locked.forEach(function (s) {
      var li = el('li', 'silhouette');
      li.appendChild(el('b', '', s.title));
      li.appendChild(el('span', 'muted', ' - ' + (s.teaser || 'Locked.')));
      list.appendChild(li);
    });
    $('lockedpane').hidden = locked.length === 0;
  }

  function chronicle(open) {
    if (!open.chronicle) { $('tab-chronicle').hidden = true; $('chroniclepane').hidden = true; return Promise.resolve(); }
    $('tab-chronicle').hidden = false; $('chroniclepane').hidden = false;
    return get('/api/chronicle?since=' + chronN).then(function (v) {
      var ul = $('chronlines'), stick = ul.scrollTop + ul.clientHeight >= ul.scrollHeight - 8;
      (v.lines || []).forEach(function (l) {
        ul.appendChild(el('li', 'kind-' + l.kind, l.text)); chronN = Math.max(chronN, l.n);
      });
      while (ul.children.length > 300) ul.removeChild(ul.firstChild);
      if (stick) ul.scrollTop = ul.scrollHeight;
      var tape = $('tape'), tv = v.tape || {};
      $('t-tape').hidden = tape.hidden = !tv.open;
      if (tv.open) {
        clear(tape);
        (tv.trips || []).forEach(function (t) { tape.appendChild(el('li', '', t.headline || ('Expedition ' + t.n))); });
      }
    });
  }

  function voice(open) {
    if (!open.voice) { $('voicestrip').hidden = true; $('tab-diary').hidden = true; $('diarypane').hidden = true; return Promise.resolve(); }
    return get('/api/voice').then(function (v) {
      muted = !!v.muted;
      $('tab-diary').hidden = false; $('diarypane').hidden = false;
      var b = $('voicemute');
      b.textContent = muted ? 'Unmute voice' : 'Mute voice'; b.setAttribute('aria-pressed', muted ? 'true' : 'false');
      $('voicestrip').hidden = !v.latest;
      $('voicetext').textContent = !v.latest ? '' : muted ? 'The hero is quiet. The diary still fills.' : v.latest.text;
      var ol = $('diarylist'); clear(ol);
      (v.diary || []).slice().reverse().forEach(function (p) {
        var li = el('li', ''); li.appendChild(el('b', '', p.milestone)); li.appendChild(el('span', '', p.text)); ol.appendChild(li);
      });
    });
  }
  $('voicemute').addEventListener('click', function () {
    fetch(answerBase + '/api/voice/mute', { method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ muted: !muted }) }).then(function () { tick(true); }).catch(function () {});
  });
  fetch('/api/config', { credentials: 'same-origin' }).then(function (r) { return r.json(); }).then(function (c) {
    answerBase = c.answerBase || (c.answerPort ? location.protocol + '//' + location.hostname + ':' + c.answerPort : '');
  }).catch(function () {});

  function quests(open) {
    if (!open.questlog) { $('tab-quests').hidden = true; $('questspane').hidden = true; return Promise.resolve(); }
    return get('/api/questlog').then(function (v) {
      $('tab-quests').hidden = false; $('questspane').hidden = false;
      $('questdiff').textContent = v.difficulty ? '(' + v.difficulty + ')' : '';
      var ul = $('questlist'); clear(ul);
      (v.quests || []).forEach(function (q) {
        var li = el('li', q.state), head = el('div', '');
        head.appendChild(el('span', 'qname', q.name)); head.appendChild(el('span', 'qstate', q.state));
        li.appendChild(head);
        li.appendChild(el('div', 'qobj', q.objective));
        li.appendChild(el('span', 'qwhy', q.why));
        if (q.reward) li.appendChild(el('span', 'qreward', 'Reward: ' + q.reward));
        ul.appendChild(li);
      });
      $('questmore').textContent = v.notGiven > 0 ? v.notGiven + ' more in Act 1 not given yet.' : '';
    });
  }

  function tick(now) {
    clearTimeout(timer);
    get('/api/unlocks').then(function (u) {
      fails = 0;
      var open = {};
      (u.open || []).forEach(function (s) { open[s.id] = true; });
      silhouettes(u);
      return Promise.all([chronicle(open), voice(open), quests(open)]);
    }).catch(function () { fails++; }).then(function () {
      if (fails < 5) timer = setTimeout(tick, 4000);
    });
  }
  tick();
})();
