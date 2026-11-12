// The first-minutes card (docs/onboarding.md): the welcome card, the first-30-minutes checklist, the Help entries and the
// reconnect banners, from GET /api/onboarding. Every word comes from data/onboarding/en.json through the reply; the panel
// only places it. It hides itself when nothing is left to show. The banners come from the failure path of the shell's
// poll (update(null, {status, fails})), the video state (window.d2video) and the absence the reply carries.
const RESTART_SHOW_MS = 20000;
const DOWN_AFTER = 2; // failed polls in a row before the page says it lost the bot (one blip is not news)

// fmtDuration: 7200 -> "2h 0m", 8040 -> "2h 14m", 90000 -> "1d 1h", 30 -> "under a minute".
export function fmtDuration(sec) {
  const s = Math.max(0, Math.round(sec || 0));
  if (s < 60) return 'under a minute';
  const m = Math.floor(s / 60), h = Math.floor(m / 60), d = Math.floor(h / 24);
  if (d > 0) return d + 'd ' + (h % 24) + 'h';
  if (h > 0) return h + 'h ' + (m % 60) + 'm';
  return m + 'm';
}

export function fill(tpl, vars) {
  return String(tpl || '').replace(/\{(\w+)\}/g, (a, k) => (k in vars ? String(vars[k]) : a));
}

// banners(): the banners to show, most urgent first. Pure: the DOM is built from this list.
//   texts = the reply's texts.reconnect; st = { down, restarted, awaySec (null: none), stream ('' | 'down' | 'reconnecting') }
export function banners(texts, st) {
  const t = texts || {};
  const out = [];
  if (st.down) out.push({ key: 'down', sev: 'warn', text: t.down || '', hint: t.downHint || '' });
  if (st.restarted) out.push({ key: 'restarted', sev: 'info', text: t.restarted || '' });
  if (st.awaySec != null) {
    out.push({ key: 'away', sev: 'info', text: fill(t.away, { duration: fmtDuration(st.awaySec) }), link: { label: t.awayLink || 'Return screen', href: '/watch/return' }, dismiss: true });
  }
  if (st.stream && !st.down) out.push({ key: 'stream', sev: 'warn', text: fill(t.stream, { state: st.stream }), action: { label: t.streamLink || 'Retry now', id: 'retry' } });
  return out;
}

export function mount(root, ctx) {
  const el = ctx.el;
  const bannerBox = el('div', 'banners');
  const welcomeBox = el('div', 'welcome');
  const helpBox = el('div', 'help');
  const artBox = el('div', 'art');
  const listBox = el('div', 'list');
  [bannerBox, welcomeBox, helpBox, artBox, listBox].forEach((b) => { b.hidden = true; }); // an empty grid box would still take its gap
  ctx.body.append(bannerBox, welcomeBox, helpBox, artBox, listBox);

  let view = null;          // the last good reply (with local changes applied before the next poll confirms them)
  let helpOpen = false;
  let conn = { fails: 0, status: 0 };
  let lastBoot = '';
  let restartedUntil = 0;
  let awaySec = null, awayDismissed = false, awayLatched = false;
  const sigs = { banner: '', welcome: '', help: '', art: '', list: '' };

  function btn(label, key, handler, variant) {
    const b = el('button', 'd-btn', label);
    b.type = 'button'; b.dataset.size = 'sm'; b.dataset.key = key;
    if (variant) b.dataset.variant = variant;
    b.addEventListener('click', handler);
    return b;
  }
  function link(label, href, key, handler) {
    const a = el('a', 'd-btn', label);
    a.href = href; a.dataset.size = 'sm'; a.dataset.key = key;
    if (handler) a.addEventListener('click', handler);
    return a;
  }
  // a POST that fails or is refused leaves the page as the server has it; the next poll shows the truth
  function act(action, step) {
    const body = step ? { action, step } : { action };
    return ctx.post('/api/onboarding', body);
  }
  function tick(id) {
    if (view) { const s = view.steps.find((x) => x.id === id); if (s) s.done = true; recount(); }
    paint();
    return act('tick', id);
  }
  function recount() {
    view.done = view.steps.filter((s) => s.done).length;
    view.allDone = view.done === view.steps.length;
  }
  function set(box, key, sig, build) {
    if (sigs[key] === sig) return;
    sigs[key] = sig;
    const kids = build();
    box.hidden = kids.length === 0;
    box.replaceChildren(...kids);
  }

  // The reconnect words are kept from the last good reply, so a page opened while the bot is down still says something
  // (a first visit with the bot down gets the built-in line).
  const saved = ctx.store('reconnect');
  function reconnectTexts() {
    if (view && view.texts && view.texts.reconnect) return view.texts.reconnect;
    try { const s = saved.get(); if (s) return JSON.parse(s); } catch (e) { /* unreadable: use the built-in line */ }
    return { down: 'Reconnecting to your hero...' };
  }

  function paintBanners() {
    const vt = typeof window !== 'undefined' && window.d2video;
    let stream = '';
    try { const s = vt && vt.state && vt.state(); if (s === 'down' || s === 'reconnecting') stream = s; } catch (e) { /* no video hook */ }
    const offline = typeof navigator !== 'undefined' && navigator.onLine === false;
    const list = banners(reconnectTexts(), {
      down: conn.fails >= DOWN_AFTER || offline,
      restarted: Date.now() < restartedUntil,
      awaySec: awayLatched && !awayDismissed ? awaySec : null,
      stream,
    });
    set(bannerBox, 'banner', JSON.stringify(list), () => list.map((b) => {
      const n = el('div', 'd-note');
      n.dataset.sev = b.sev; n.dataset.banner = b.key;
      n.setAttribute('role', b.key === 'down' ? 'alert' : 'status');
      n.append(el('span', 'msg', b.text));
      if (b.hint) n.append(el('span', 'hint', b.hint));
      if (b.link) n.append(link(b.link.label, b.link.href, 'banner-' + b.key));
      if (b.action) n.append(btn(b.action.label, 'banner-' + b.key, () => { try { window.d2video.reconnect(); } catch (e) { /* none */ } }));
      if (b.dismiss) n.append(btn('Dismiss', 'banner-' + b.key + '-x', () => { awayDismissed = true; paint(); }, 'ghost'));
      return n;
    }));
    return list.length > 0;
  }

  function paintWelcome() {
    const w = view && view.texts && view.texts.welcome;
    const on = !!(view && view.showWelcome && w);
    set(welcomeBox, 'welcome', on ? JSON.stringify(w) : '', () => {
      if (!on) return [];
      const card = el('section', 'welcome-card');
      card.setAttribute('aria-label', w.title);
      const lines = el('ul', 'lines');
      (w.lines || []).forEach((l) => lines.append(el('li', '', l)));
      const row = el('div', 'btnrow');
      (w.buttons || []).forEach((b) => row.append(link(b.label, b.href, 'welcome-' + b.id, b.href.startsWith('#') ? () => tick(b.id) : null)));
      row.append(btn(w.dismiss || 'Hide', 'welcome-dismiss', () => {
        view.showWelcome = false; paint(); act('dismiss_welcome');
      }, 'ghost'));
      card.append(el('h3', '', w.title), lines, row);
      return [card];
    });
    return on;
  }

  function paintHelp() {
    const h = view && view.texts && view.texts.help;
    const on = !!(helpOpen && h);
    set(helpBox, 'help', on ? JSON.stringify([h, view.showWelcome, view.showList]) : '', () => {
      if (!on) return [];
      const box = el('div', 'help-card');
      box.setAttribute('role', 'group');
      box.setAttribute('aria-label', h.label);
      box.append(btn(h.showWelcome, 'help-welcome', () => { view.showWelcome = true; helpOpen = false; paint(); act('show_welcome'); }));
      box.append(btn(h.showChecklist, 'help-list', () => { view.showList = true; view.collapsed = false; helpOpen = false; paint(); act('show_list'); }));
      box.append(btn('Close', 'help-close', () => { helpOpen = false; paint(); }, 'ghost'));
      return [box];
    });
    return on;
  }

  function paintArt() {
    const a = view && view.art, t = view && view.texts && view.texts.art;
    const on = !!(a && t && view.showList);
    set(artBox, 'art', on ? JSON.stringify(a) : '', () => {
      if (!on) return [];
      const n = el('p', 'd-note', fill(t.preparing, { done: a.done, total: a.total }));
      n.dataset.sev = 'info'; n.setAttribute('role', 'status');
      return [n];
    });
    return on;
  }

  function paintList() {
    const c = view && view.texts && view.texts.checklist;
    const on = !!(view && view.showList && c);
    const sig = on ? JSON.stringify([view.steps.map((s) => [s.id, s.label, s.done]), view.collapsed, view.allDone]) : '';
    set(listBox, 'list', sig, () => {
      if (!on) return [];
      const card = el('section', 'check');
      card.setAttribute('aria-label', c.title);
      const head = el('div', 'check-head');
      head.append(el('h3', '', c.title), el('span', 'meta', fill(c.progress, { done: view.done, total: view.steps.length })));
      head.append(btn(view.collapsed ? c.expand : c.collapse, 'check-fold', () => {
        view.collapsed = !view.collapsed; paint(); act(view.collapsed ? 'collapse' : 'expand');
      }, 'ghost'));
      card.append(head);
      if (view.allDone) card.append(el('p', 'd-note', c.done));
      if (!view.collapsed) {
        const ol = el('ol', 'steps');
        view.steps.forEach((s) => {
          const li = el('li', 'step');
          li.dataset.done = s.done ? '1' : '0';
          const mark = el('span', 'tick', s.done ? 'done' : 'to do');
          const label = s.href ? link(s.label, s.href, 'step-' + s.id, s.tickOnOpen ? () => tick(s.id) : null) : el('span', 'name', s.label);
          label.classList.add('name');
          li.append(mark, label);
          if (s.hint) li.append(el('span', 'hint', s.hint));
          if (!s.done) li.append(btn(c.manual, 'tick-' + s.id, () => tick(s.id), 'ghost'));
          ol.append(li);
        });
        card.append(ol);
      }
      return [card];
    });
    if (view && view.steps) ctx.meta.textContent = view.showList ? fill((c || {}).progress, { done: view.done, total: view.steps.length }) : '';
    return on;
  }

  function paint() {
    const shown = [paintBanners(), paintWelcome(), paintHelp(), paintArt(), paintList()];
    ctx.hide(!shown.some(Boolean));
  }

  // the video state lives outside the reply, so a slow timer re-reads it while the tab is visible
  function watchStream() {
    if (typeof document === 'undefined' || !document.hidden) paintBanners() && ctx.hide(false);
    ctx.later(watchStream, 2000);
  }
  ctx.later(watchStream, 2000);

  // the Help button of the page header (web/shell.js) sends this event, so the panel can open its Help entries even
  // when it is otherwise hidden
  if (typeof window !== 'undefined' && window.addEventListener) {
    window.addEventListener('d2help', () => {
      helpOpen = !helpOpen;
      paint();
      if (helpOpen && root.scrollIntoView) root.scrollIntoView({ block: 'nearest' });
    });
  }

  return {
    update(v, info) {
      if (!v) {
        const status = (info && info.status) || 0;
        if (status === 404 && !view) { ctx.hide(true); return; } // the route is off: nothing to show
        conn = { fails: (info && info.fails) || 1, status };
        paint();
        return;
      }
      conn = { fails: 0, status: 0 };
      if (lastBoot && v.boot && v.boot !== lastBoot) restartedUntil = Date.now() + RESTART_SHOW_MS;
      lastBoot = v.boot || lastBoot;
      if (!awayLatched && v.away && v.away.absent) { awayLatched = true; awaySec = v.away.seconds; }
      if (!view && v.texts && v.texts.reconnect) saved.set(JSON.stringify(v.texts.reconnect));
      view = v;
      paint();
      if (restartedUntil > Date.now()) ctx.later(paint, RESTART_SHOW_MS + 50);
    },
  };
}
