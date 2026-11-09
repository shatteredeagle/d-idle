// Alerts and pings (docs/unlock-alerts.md). Polls GET /api/alerts, shows toasts and a notification centre, and can raise
// browser notifications while this tab is open. Background push (tab closed) needs a service worker and web push and is
// NOT built. The browser asks for notification permission only when the button is pressed. All text goes through textContent.
(function () {
  'use strict';
  var $ = function (id) { return document.getElementById(id); };
  if (!$('alertspane')) return;
  function el(tag, cls, text) { var e = document.createElement(tag); if (cls) e.className = cls; if (text !== undefined) e.textContent = text; return e; }
  function clear(n) { while (n.firstChild) n.removeChild(n.firstChild); }
  var CUES = { info: [[520, 90]], good: [[520, 90], [780, 120]], warn: [[400, 140], [400, 140]], alarm: [[300, 200], [220, 320]] };
  var seq = 0, all = [], unseen = 0, fails = 0, timer = 0, answerBase = '', first = true;
  var KEY = 'd2lod.alerts.sound';

  function soundOn() { return $('alertsound').checked; }
  function beep(sev) {
    var AC = window.AudioContext || window.webkitAudioContext, cue = CUES[sev];
    if (!AC || !cue || !soundOn()) return;
    try {
      var ac = new AC(), t = ac.currentTime;
      cue.forEach(function (s) {
        var o = ac.createOscillator(), g = ac.createGain();
        o.frequency.value = s[0]; g.gain.value = 0.05; o.connect(g); g.connect(ac.destination);
        o.start(t); o.stop(t + s[1] / 1000); t += s[1] / 1000;
      });
    } catch (e) { /* no audio */ }
  }
  try { $('alertsound').checked = localStorage.getItem(KEY) === '1'; } catch (e) { /* storage off */ }
  $('alertsound').addEventListener('change', function () { try { localStorage.setItem(KEY, soundOn() ? '1' : '0'); } catch (e) { /* storage off */ } });

  function toast(a) {
    var t = el('div', 'toast ' + a.severity), box = $('toasts');
    t.appendChild(el('span', '', a.text));
    var x = el('button', 'btn ghost small', 'Dismiss'); x.type = 'button';
    x.addEventListener('click', function () { if (t.parentNode) t.parentNode.removeChild(t); });
    t.appendChild(x); box.appendChild(t);
    while (box.children.length > 4) box.removeChild(box.firstChild);
    setTimeout(function () { if (t.parentNode) t.parentNode.removeChild(t); }, a.severity === 'alarm' ? 20000 : 8000);
  }

  function browserNote(a) {
    if (!('Notification' in window) || Notification.permission !== 'granted' || !document.hidden) return;
    try { new Notification('d.idle', { body: a.text, tag: 'd2lod-' + a.seq }); } catch (e) { /* not allowed here */ }
  }

  function paintNotify() {
    var b = $('alertnotify'), note = $('alertnotifynote');
    if (!('Notification' in window)) { b.hidden = true; note.textContent = 'This browser has no desktop notifications.'; return; }
    var p = Notification.permission;
    b.hidden = p !== 'default';
    note.textContent = p === 'granted' ? 'Desktop notifications are on while this tab is open. Closing the tab stops them (no background push).'
      : p === 'denied' ? 'Desktop notifications are blocked in the browser settings for this site.'
      : 'Works while this tab is open. The browser asks only when you press the button. Background push is not built.';
  }
  $('alertnotify').addEventListener('click', function () {
    Notification.requestPermission().then(paintNotify, paintNotify);
  });

  function paintList() {
    var ul = $('alertlist'); clear(ul);
    all.slice().reverse().forEach(function (a) {
      var li = el('li', 'alert ' + a.severity);
      li.appendChild(el('b', '', a.severity + ' '));
      li.appendChild(el('span', '', a.text));
      ul.appendChild(li);
    });
    var badge = $('alertbadge'); badge.hidden = unseen === 0; badge.textContent = String(unseen);
  }

  function paintState(v) {
    $('alertsstate').textContent = v.open ? '(on)' : '(death, low life and gear only)';
    var tz = $('alertsteaser'); clear(tz);
    tz.hidden = v.open;
    if (!v.open) {
      tz.appendChild(el('span', '', (v.teaser || 'Locked.') + (v.price ? ' ' + v.price + ' tally.' : '') + ' '));
      if (v.price) {
        var b = el('button', 'btn small', 'Buy for ' + v.price + ' tally'); b.type = 'button';
        b.addEventListener('click', function () {
          fetch(answerBase + '/api/unlocks/buy', { method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ id: 'alerts' }) }).then(function () { tick(true); }).catch(function () {});
        });
        tz.appendChild(b);
      }
    }
    $('alertconsent').textContent = v.outbound ? v.outbound.consent : '';
  }

  function tick() {
    clearTimeout(timer);
    fetch('/api/alerts?since=' + seq, { cache: 'no-store', credentials: 'same-origin' }).then(function (r) {
      if (!r.ok) throw new Error(String(r.status));
      return r.json();
    }).then(function (v) {
      fails = 0;
      $('tab-alerts').hidden = false; $('alertspane').hidden = false;
      paintState(v); paintNotify();
      var shown = $('alertspane').offsetParent !== null;
      (v.alerts || []).forEach(function (a) {
        all.push(a); if (all.length > 100) all.shift();
        if (!first) { toast(a); beep(a.severity); browserNote(a); if (!shown) unseen++; }
      });
      if (first) unseen = 0;
      first = false; seq = v.seq; paintList();
    }).catch(function () { fails++; }).then(function () {
      if (fails < 5) timer = setTimeout(tick, 4000);
    });
  }
  $('tab-alerts').addEventListener('click', function () { unseen = 0; paintList(); });
  fetch('/api/config', { credentials: 'same-origin' }).then(function (r) { return r.json(); }).then(function (c) {
    answerBase = c.answerBase || (c.answerPort ? location.protocol + '//' + location.hostname + ':' + c.answerPort : '');
  }).catch(function () {}).then(tick);
})();
