// The first-30-minutes tour (card onboardtour, docs/onboarding-tour.md): dismissible tips tied to what really happened (first launch,
// first death, first town, first Vault item, first quest hand-in), the "Now" line from /api/explain, and the Help sheet. Words are in
// /watch/onboarding.json. What a viewer has seen is kept in localStorage (d2tour) with an in-memory fallback. Text goes through textContent only.
// Classic script: window.dTour, or module.exports for the node test.
(function (g) {
  'use strict';
  const KEY = 'd2tour';
  const POLL_MS = 15000;
  const fill = (t, v) => String(t || '').replace(/\{(\w+)\}/g, (a, k) => (k in v ? String(v[k]) : a));

  // store(ls): { read() -> {seen:{id:1}, off:bool}, write(s), ok } ; ls = a Storage or null. A throw anywhere falls back to memory.
  function store(ls) {
    let mem = { seen: {}, off: false }, ok = true;
    try { if (!ls) throw new Error('none'); ls.getItem(KEY); } catch (e) { ok = false; }
    const norm = (s) => ({ seen: (s && typeof s.seen === 'object' && s.seen) || {}, off: !!(s && s.off) });
    return {
      get ok() { return ok; },
      read() {
        if (!ok) return mem;
        try { const r = ls.getItem(KEY); if (r) mem = norm(JSON.parse(r)); } catch (e) { ok = false; }
        return mem;
      },
      write(s) {
        mem = norm(s);
        if (!ok) return;
        try { ls.setItem(KEY, JSON.stringify(mem)); } catch (e) { ok = false; }
      },
    };
  }

  // triggered(snap, townAreas): the step ids whose real event is true. snap = { uptime, hero, vault, quests } (a missing part is null).
  function triggered(snap, townAreas) {
    const s = snap || {}, out = ['launch'];
    if (s.hero && s.hero.area && (townAreas || []).includes(String(s.hero.area).toLowerCase())) out.push('town');
    if (s.vault && s.vault.total > 0) out.push('vault');
    if (s.quests && (s.quests.quests || []).some((q) => q.state === 'done')) out.push('quest');
    if (s.uptime && s.uptime.deaths > 0) out.push('death');
    return out;
  }

  // nowLine(explain, doing): { kind: 'now'|'wait', text } or null. explain = the /api/explain reply.
  function nowLine(ex, doing) {
    const e = ex && ex.enabled && ex.explain;
    if (!e) return null;
    const ref = (e.refusals || [])[0];
    const d = (e.decisions || [])[(e.decisions || []).length - 1];
    if (!d && !ref) return null;
    const dd = doing || {};
    const text = d ? (dd[d.code] || dd[d.family] || dd._default || d.code) : '';
    return { text, wait: ref ? (ref.need || ref.what) : '' };
  }

  function create(doc, o) {
    const T = o.texts, st = o.store;
    const later = o.later || ((f, ms) => setTimeout(f, ms));
    const mk = (tag, cls, text) => { const e = doc.createElement(tag); if (cls) e.className = cls; if (text !== undefined) e.textContent = text; return e; };
    const btn = (label, key, fn, variant) => {
      const b = mk('button', 'd-btn', label);
      b.type = 'button'; b.dataset.size = 'sm'; b.dataset.key = key;
      if (variant) b.dataset.variant = variant;
      b.addEventListener('click', fn);
      return b;
    };
    const card = mk('section', 'd-tour');
    card.hidden = true;
    card.setAttribute('role', 'dialog');
    card.setAttribute('aria-label', T.tourButton);
    const now = mk('div', 'd-now');
    now.hidden = true;
    now.setAttribute('role', 'status');
    doc.body.append(card, now);

    const shown = [];     // step ids shown this visit, for Back
    let cur = '', pos = -1, due = [];
    const byId = (id) => T.steps.find((s) => s.id === id);

    function paintCard() {
      const s = byId(cur);
      if (!s) { card.hidden = true; card.replaceChildren(); return; }
      const row = mk('div', 'row');
      if (pos > 0) row.append(btn(T.back, 'tour-back', () => { pos--; cur = shown[pos]; paintCard(); }, 'ghost'));
      row.append(btn(T.next, 'tour-next', () => dismiss(false)));
      row.append(btn(T.skip, 'tour-skip', () => dismiss(true), 'ghost'));
      card.replaceChildren(mk('span', 'meta', fill(T.step, { n: pos + 1, total: Math.max(shown.length, due.length) })), mk('h2', '', s.title), mk('p', '', s.body), row);
      card.hidden = false;
      const nb = row.children.find((c) => c.dataset.key === 'tour-next');
      if (nb && nb.focus) nb.focus();
    }
    function dismiss(all) {
      const s = st.read();
      if (all) { s.off = true; T.steps.forEach((x) => { s.seen[x.id] = 1; }); } else s.seen[cur] = 1;
      st.write(s);
      cur = ''; pos = -1;
      if (!all) next();
      else { card.hidden = true; card.replaceChildren(); }
    }
    function next() {
      const s = st.read();
      const id = due.find((x) => !s.seen[x] && !shown.includes(x)) || due.find((x) => !s.seen[x]);
      if (s.off || !id) { card.hidden = true; card.replaceChildren(); return; }
      cur = id;
      if (!shown.includes(id)) shown.push(id);
      pos = shown.indexOf(id);
      paintCard();
    }
    // feed(snap): the page's latest reading; a step that is due and unseen comes up (one at a time)
    function feed(snap) {
      due = triggered(snap, T.townAreas);
      if (!cur) next();
    }
    function paintNow(ex) {
      const l = nowLine(ex, T.doing);
      if (!l) { now.hidden = true; now.replaceChildren(); return; }
      const kids = [];
      if (l.text) kids.push(mk('span', '', T.nowLabel + ': '), mk('b', '', l.text));
      if (l.wait) kids.push(mk('span', '', T.waitLabel + ': ' + l.wait));
      now.replaceChildren(...kids);
      now.hidden = false;
    }
    function replay() { st.write({ seen: {}, off: false }); shown.length = 0; cur = ''; next(); }

    function openHelp() {
      const h = T.help;
      const ul = mk('ul'); h.lines.forEach((l) => ul.append(mk('li', '', l)));
      const body = mk('div', 'd-help-body');
      body.append(ul);
      if (!st.ok) body.append(mk('p', 'meta', h.storeOff));
      const row = mk('div', 'row');
      let sh = null;
      row.append(btn(h.replay, 'help-replay', () => { if (sh) sh.close(); replay(); if (o.toast) o.toast('info', h.replayDone); }));
      row.append(btn(h.guide, 'help-guide', () => { if (sh) sh.close(); // the welcome card and the first-30-minutes list of the Help drawer (TASK-247)
        const ev = new o.CustomEvent('d2:nav', { cancelable: true, detail: { id: 'help' } });
        doc.dispatchEvent(ev);
        if (o.onGuide) o.onGuide();
      }));
      body.append(row);
      if (o.sheet) sh = o.sheet.open(doc, { title: h.title, panel: true, nodes: [body] });
      return sh;
    }

    return { feed, paintNow, openHelp, replay, card, now, dismiss, state: () => ({ cur, pos, due: due.slice(), shown: shown.slice() }) };
  }

  // start(win): the browser wiring. Polls the readings, then feeds them.
  async function start(win) {
    const doc = win.document;
    let T;
    try { T = await (await win.fetch('/watch/onboarding.json', { cache: 'no-store', credentials: 'same-origin' })).json(); } catch (e) { return null; }
    let ls = null; try { ls = win.localStorage; } catch (e) { ls = null; }
    const tour = create(doc, { texts: T, store: store(ls), sheet: win.dSheet, toast: (s, t) => win.dToast && win.dToast.show && win.dToast.show(s, t), CustomEvent: win.CustomEvent });
    const get = (p) => win.fetch(p, { cache: 'no-store', credentials: 'same-origin' }).then((r) => (r.ok ? r.json() : null)).catch(() => null);
    async function poll() {
      if (!doc.hidden) {
        const [uptime, hero, vault, quests, ex] = await Promise.all([get('/api/uptime'), get('/api/hero'), get('/api/vault?limit=1'), get('/api/questlog'), get('/api/explain')]);
        tour.feed({ uptime, hero, vault, quests });
        tour.paintNow(ex);
      }
      setTimeout(poll, POLL_MS);
    }
    const b = doc.getElementById('btn-tour');
    if (b) b.addEventListener('click', () => tour.openHelp());
    poll();
    return tour;
  }

  const api = { KEY, store, triggered, nowLine, create, start, fill };
  if (typeof module !== 'undefined' && module.exports) module.exports = api;
  else if (g.document) {
    g.dTour = api;
    if (g.document.getElementById && g.document.getElementById('btn-tour')) start(g);
  }
})(typeof window !== 'undefined' ? window : globalThis);
