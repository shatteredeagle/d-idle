// The one toast and the one busy button (card vaultperf, below). The toast (card uitoast, docs/ui-design-system.md "Toasts and severity"): a short message that goes by itself, with a Dismiss
// button or one action (Undo). Every page uses it: the shell panels, the Vault, the live screen, the alert feed.
// ONE severity scale for toast, note and chip: info, good, warn, bad. Old names map in (alarm and danger to bad, ok to good).
// ONE table of auto-dismiss times. The stack sits in #toasts (role=status, aria-live=polite; a bad toast is role=alert, announced at once).
// Text goes through textContent only. Classic script: window.dToast, or module.exports for the node test.
(function (g) {
  'use strict';
  const SEVERITIES = ['info', 'good', 'warn', 'bad'];
  const ALIAS = { alarm: 'bad', danger: 'bad', error: 'bad', ok: 'good', warning: 'warn' };
  const TIMES = { info: 6000, good: 6000, warn: 10000, bad: 20000, action: 8000 }; // ms; a toast with an action uses 'action' unless it says ms
  const MAX = 4; // the stack keeps this many, oldest first out

  const sev = (s) => { const v = ALIAS[s] || s; return SEVERITIES.includes(v) ? v : 'info'; };

  // create(doc, { host, later, clear, dismissText }): a stack on doc. later(fn, ms) and clear(handle) default to the timers.
  function create(doc, o) {
    o = o || {};
    const later = o.later || ((fn, ms) => setTimeout(fn, ms));
    const clear = o.clear || ((h) => clearTimeout(h));
    function host() {
      if (o.host) return o.host;
      const h = doc.getElementById && doc.getElementById('toasts');
      if (h) return h;
      const b = doc.createElement('div');
      b.id = 'toasts'; b.className = 'd-toasts';
      b.setAttribute('role', 'status'); b.setAttribute('aria-live', 'polite');
      doc.body.append(b);
      return b;
    }
    // show(sev, text, act): act = { label, run, ms } adds one action button that runs once; returns the toast node
    function show(s, text, act) {
      const t = doc.createElement('div');
      t.className = 'd-toast';
      t.dataset.sev = sev(s);
      t.setAttribute('role', sev(s) === 'bad' ? 'alert' : 'status');
      const msg = doc.createElement('span'); msg.textContent = text; t.append(msg);
      const b = doc.createElement('button');
      b.className = 'd-btn'; b.type = 'button'; b.textContent = act ? act.label : (o.dismissText || 'Dismiss');
      b.dataset.size = 'sm'; b.dataset.variant = 'ghost'; b.dataset.key = 'toast-dismiss';
      const box = host();
      let timer = null;
      const detach = (n) => { n.hidden = true; if (n.remove) n.remove(); else if (Array.isArray(box.children) && box.children.includes(n)) box.children.splice(box.children.indexOf(n), 1); }; // the else: a stub DOM
      const remove = () => { if (timer !== null) { clear(timer); timer = null; } detach(t); };
      b.addEventListener('click', () => { remove(); if (act && act.run) act.run(); });
      t.append(b);
      box.append(t);
      const mine = () => [...box.children].filter((n) => n.className === 'd-toast');
      while (mine().length > MAX) { const old = mine()[0]; if (old.dismiss) old.dismiss(); else detach(old); }
      timer = later(remove, (act && act.ms) || (act ? TIMES.action : TIMES[sev(s)]));
      t.dismiss = remove;
      return t;
    }
    return { show, host };
  }

  // ---- the one busy button (card vaultperf) ----
  // Whenever an action takes time, the button must answer at once: data-busy="working" (spinner, aria-busy, further taps swallowed),
  // then "done" or "failed" for a moment. State lives in a registry keyed by data-key or id, so a panel that redraws and replaces the
  // button gets the state again (a MutationObserver re-applies it). It never touches the button's text or `disabled`, so a page that
  // draws its own label keeps it. busy.run(btn, fn) is the explicit form; install() makes EVERY button whose click starts a POST or PUT
  // busy for that request's duration (a fetch wrapper tied to the click that is being handled), so no page wires its buttons one by one.
  function createBusy(doc, o) {
    o = o || {};
    const later = o.later || ((fn, ms) => setTimeout(fn, ms));
    const clear = o.clear || ((h) => clearTimeout(h));
    const DONE_MS = o.doneMs || 1200, FAIL_MS = o.failMs || 3000;
    const live = new Map(); // key -> { state, timer }
    const keyOf = (b) => (b && ((b.dataset && b.dataset.key) || b.id)) || '';
    const paint = (b, st) => {
      if (!b) return;
      if (st) { b.dataset.busy = st; b.setAttribute('aria-busy', st === 'working' ? 'true' : 'false'); } else { delete b.dataset.busy; b.removeAttribute('aria-busy'); }
    };
    const okOf = (r) => !(r === null || r === undefined || r === false || (typeof r === 'object' && (r.ok === false || r.error)));
    // set: the state of one button (and of its registry entry, so a redrawn twin gets it)
    function set(b, st, ms) {
      const k = keyOf(b);
      const e = k ? live.get(k) : null;
      if (e && e.timer !== null) { clear(e.timer); e.timer = null; }
      if (e && e.b && e.b !== b) { paint(b, ''); b = e.b; } // a redraw replaced the button: the twin carries the state
      paint(b, st);
      if (!k) { if (st && st !== 'working') later(() => { if (b.dataset.busy === st) paint(b, ''); }, ms); return; }
      if (!st) { live.delete(k); return; }
      const n = { st, timer: null, b };
      if (st !== 'working') n.timer = later(() => { if (live.get(k) === n) { live.delete(k); paint(n.b, ''); } }, ms);
      live.set(k, n);
    }
    const isWorking = (b) => !!b && b.dataset && b.dataset.busy === 'working';
    // run: fn() may return a promise (or a plain value); the outcome is judged by ok(result), default: not null/false, no ok:false, no error
    function run(b, fn, ok) {
      if (isWorking(b)) return Promise.resolve(null);
      set(b, 'working');
      let p;
      try { p = Promise.resolve(fn()); } catch (e) { p = Promise.reject(e); }
      return p.then((r) => { set(b, (ok || okOf)(r) ? 'done' : 'failed', (ok || okOf)(r) ? DONE_MS : FAIL_MS); return r; },
        (e) => { set(b, 'failed', FAIL_MS); throw e; });
    }
    // install(win): the click guard and the fetch wrapper
    function install(win) {
      if (!win || win.__dBusy) return;
      win.__dBusy = true;
      let cur = null;
      doc.addEventListener('click', (ev) => {
        const t = ev.target && ev.target.closest ? ev.target.closest('button') : null;
        if (t && isWorking(t)) { ev.stopPropagation(); ev.preventDefault(); if (ev.stopImmediatePropagation) ev.stopImmediatePropagation(); return; }
        cur = t && !(t.dataset && t.dataset.nobusy !== undefined) ? t : null;
        if (cur) later(() => { cur = null; }, 0);
      }, true);
      const raw = win.fetch;
      if (typeof raw !== 'function') return;
      win.fetch = function (input, init) {
        const m = String((init && init.method) || (input && input.method) || 'GET').toUpperCase();
        const b = cur && m !== 'GET' && m !== 'HEAD' ? cur : null;
        if (b) cur = null;
        const pr = raw.apply(this, arguments);
        if (!b) return pr;
        set(b, 'working');
        const fin = (good) => { set(b, good ? 'done' : 'failed', good ? DONE_MS : FAIL_MS); };
        pr.then((res) => {
          if (!res || res.ok === false) { fin(false); return; }
          const ct = res.headers && res.headers.get && res.headers.get('content-type');
          if (!ct || ct.indexOf('json') < 0 || !res.clone) { fin(true); return; }
          res.clone().json().then((j) => fin(okOf(j)), () => fin(true));
        }, () => fin(false));
        return pr;
      };
      if (typeof win.MutationObserver === 'function' && doc.body) {
        new win.MutationObserver(() => {
          if (!live.size) return;
          live.forEach((e, k) => {
            const nb = doc.querySelector ? doc.querySelector('[data-key="' + k + '"], #' + k) : null;
            if (nb && nb !== e.b) { e.b = nb; paint(nb, e.st); }
          });
        }).observe(doc.body, { childList: true, subtree: true });
      }
    }
    return { run, set, install, isWorking, live };
  }

  let main = null;
  const api = { createBusy, SEVERITIES, TIMES, MAX, sev, create, show(s, text, act) { main = main || create(g.document); return main.show(s, text, act); } };
  if (typeof module !== 'undefined' && module.exports) module.exports = api;
  else {
    g.dToast = api;
    if (g.document && g.document.addEventListener && !g.dBusy) { g.dBusy = createBusy(g.document); g.dBusy.install(g); }
  }
})(typeof window !== 'undefined' ? window : globalThis);
