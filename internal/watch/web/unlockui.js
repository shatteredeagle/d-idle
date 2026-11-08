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

  // Identify reveal: a face-down card carries no quality or colour; the flip asks the answer route and then shows both.
  var shown = {}, modalCard = '';
  function beep(cue) {
    var AC = window.AudioContext || window.webkitAudioContext;
    if (!AC || !$('revealsound').checked || !cue) return;
    try {
      var ac = new AC(), t = ac.currentTime;
      cue.forEach(function (s) {
        var o = ac.createOscillator(), g = ac.createGain();
        o.frequency.value = s[0]; g.gain.value = 0.05; o.connect(g); g.connect(ac.destination);
        o.start(t); o.stop(t + s[1] / 1000); t += s[1] / 1000;
      });
    } catch (e) { /* no audio */ }
  }
  function flip(id) {
    return fetch(answerBase + '/api/reveal/flip', { method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id: id }) }).then(function (r) { return r.json(); }).then(function (j) {
      if (j.card) { beep(j.card.cue); showCard(j.card); }
      tick(true);
    }).catch(function () {});
  }
  function showCard(c) {
    var sh = $('revealsheet'); sh.className = 'sheet revealsheet' + (c.colour ? ' ' + c.colour : '');
    $('rv-title').textContent = c.revealed ? c.label + ': ' + c.name : (shown.face || 'Unidentified item');
    $('rv-text').textContent = c.revealed ? c.text : (shown.faceText || '');
    $('rv-note').textContent = c.revealed ? ((c.affixes || []).join('; ') || c.affixNote || '') : '';
    $('rv-flip').hidden = c.revealed; modalCard = c.revealed ? '' : c.id;
    sh.hidden = false; (c.revealed ? $('rv-close') : $('rv-flip')).focus();
  }
  $('rv-flip').addEventListener('click', function () { if (modalCard) flip(modalCard); });
  $('rv-close').addEventListener('click', function () { $('revealsheet').hidden = true; modalCard = ''; });
  document.addEventListener('keydown', function (e) { if (e.key === 'Escape' && !$('revealsheet').hidden) $('rv-close').click(); });
  var offered = {};
  function reveal(open) {
    if (!open.idreveal) { $('tab-cards').hidden = true; $('cardspane').hidden = true; return Promise.resolve(); }
    return get('/api/reveal').then(function (v) {
      $('tab-cards').hidden = false; $('cardspane').hidden = false;
      shown.face = v.face; shown.faceText = v.faceText;
      var ul = $('cardlist'); clear(ul);
      (v.cards || []).slice().reverse().forEach(function (c) {
        var li = el('li', ''), b = el('button', 'btn small', c.revealed ? 'Show' : 'Flip');
        b.type = 'button'; b.addEventListener('click', function () { showCard(c); });
        li.appendChild(b);
        if (c.revealed) li.appendChild(el('span', c.colour, c.label + ': ' + c.name));
        else li.appendChild(el('span', 'cface', (v.face || 'Unidentified item') + ' #' + c.n));
        ul.appendChild(li);
        if (!c.revealed && !offered[c.id] && $('revealsheet').hidden) { offered[c.id] = true; showCard(c); }
      });
    });
  }

  function tick(now) {
    clearTimeout(timer);
    get('/api/unlocks').then(function (u) {
      fails = 0;
      var open = {};
      (u.open || []).forEach(function (s) { open[s.id] = true; });
      silhouettes(u);
      return Promise.all([chronicle(open), voice(open), quests(open), reveal(open)]);
    }).catch(function () { fails++; }).then(function () {
      if (fails < 5) timer = setTimeout(tick, 4000);
    });
  }
  tick();
})();
