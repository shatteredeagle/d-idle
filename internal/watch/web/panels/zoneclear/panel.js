// The Zone clear panel (docs/farmloop.md, "Dashboard"): two settings of how a zone is farmed. "Explored" is the share of a
// zone's roam coverage the hero visits before it calls the zone explored and moves on (bot.exploredPct, 30 to 100).
// "Farm reset" is the share at which it leaves a cleared farm zone for a fresh game and comes back (bot.farmResetPct,
// 50 to 100). From GET /api/zoneclear (watch.zoneclear/1); Save posts the two values
// to POST /api/zoneclear, Reset puts all back to their defaults. The bot reads the knobs at its next zone decision, so
// nothing needs a restart. Not in the default layout: it is opt-in (the layout editor lists it); minimum slot size m.
// textContent only.
const KEYS = { explored: 'bot.exploredPct', pct: 'bot.farmResetPct', cool: 'bot.farmResetCooldownMin' };

// byKey indexes the knob rows of a reply.
export function byKey(v) {
  const m = {};
  ((v && v.knobs) || []).forEach((k) => { m[k.key] = k; });
  return m;
}

// state is the two controls' values from a reply (null while a knob is missing).
export function state(v) {
  const m = byKey(v);
  if (!m[KEYS.explored] || !m[KEYS.pct]) return null;
  return { explored: m[KEYS.explored].value, pct: m[KEYS.pct].value };
}

// source is the one word of where the values come from: the strongest of the knobs ("dashboard" > "config" > "default").
export function source(v) {
  const rank = { default: 0, config: 1, dashboard: 2 };
  let best = 'default';
  ((v && v.knobs) || []).forEach((k) => { if ((rank[k.source] || 0) > rank[best]) best = k.source; });
  return best;
}

// problem is the client-side check; the server's range check is the only rule left, so there is none here.
export function problem() { return ''; }

// body is what Save posts.
export function body(s) {
  return { exploredPct: s.explored, farmResetPct: s.pct };
}

export function same(a, b) { return !!a && !!b && a.explored === b.explored && a.pct === b.pct; }

export function mount(root, ctx) {
  const head = ctx.el('div', 'zc-head');
  const title = ctx.el('span', '', 'Zone clear'), badge = ctx.el('em', '', '');
  head.append(title, badge);
  const line = ctx.el('div', 'zc-line', 'Clear a farm zone to the first share, then leave and rejoin for a fresh one at the second.');

  function slider(id, label, min, max) {
    const row = ctx.el('div', 'zc-row');
    const lab = ctx.el('label', 'zc-lab', label);
    lab.htmlFor = 'zc-' + id;
    const input = ctx.el('input', 'zc-range');
    input.type = 'range'; input.id = 'zc-' + id; input.min = String(min); input.max = String(max); input.step = '1';
    input.dataset.key = id;
    const val = ctx.el('output', 'zc-val', '');
    val.htmlFor = input.id;
    row.append(lab, input, val);
    return { row, input, val, lab };
  }
  const ex = slider('explored', 'Explored', 30, 100);
  ex.input.setAttribute('aria-label', 'Explored share to move on, percent');
  const rs = slider('pct', 'Fresh game at', 50, 100);
  rs.input.setAttribute('aria-label', 'Farm reset share, percent');
  const acts = ctx.el('div', 'zc-acts');
  const save = ctx.el('button', 'd-btn', 'Save'), reset = ctx.el('button', 'd-btn', 'Reset to default');
  save.type = reset.type = 'button'; save.dataset.variant = 'primary'; save.dataset.key = 'save'; reset.dataset.key = 'reset';
  acts.append(save, reset);
  const msg = ctx.el('div', 'zc-msg');
  msg.setAttribute('aria-live', 'polite');
  // the settings scroll inside the box the slot gives (a short rail cell holds the head and part of them); Save stays reachable
  const sc = ctx.el('div', 'zc-scroll');
  sc.dataset.scroll = '1'; sc.tabIndex = 0;
  sc.setAttribute('aria-label', 'Zone clear settings');
  sc.append(line, ex.row, rs.row, acts, msg);
  ctx.body.append(head, sc);

  let live = null, cur = null, busy = false;
  function draw() {
    if (!cur) return;
    ex.input.value = String(cur.explored); ex.val.textContent = cur.explored + '%';
    rs.input.value = String(cur.pct); rs.val.textContent = cur.pct + '%';
    const p = problem(cur);
    msg.textContent = p;
    save.disabled = busy || !!p || same(cur, state(live));
    reset.disabled = busy;
  }
  function take(v) { live = v; cur = state(v); badge.textContent = cur ? 'set by ' + source(v) : ''; draw(); }
  ex.input.addEventListener('input', () => { cur.explored = +ex.input.value; draw(); });
  rs.input.addEventListener('input', () => { cur.pct = +rs.input.value; draw(); });
  function send(b, done) {
    busy = true; draw();
    ctx.post('/api/zoneclear', b).then((j) => {
      busy = false;
      if (j && j.ok && j.view) { take(j.view); ctx.toast('good', done); return; }
      msg.textContent = (j && j.error) || 'could not save';
      draw();
    });
  }
  save.addEventListener('click', () => { if (cur && !problem(cur)) send(body(cur), 'Zone clear saved'); });
  reset.addEventListener('click', () => send({ reset: true }, 'Zone clear back to the defaults'));
  return {
    update(v) {
      if (!v || !v.knobs) { ctx.state('error'); return; }
      if (!busy && (!cur || same(cur, state(live)))) take(v); else live = v; // an edit in progress is not overwritten by the poll
      ctx.state('ready');
    },
  };
}
