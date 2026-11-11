// Journey (docs/campaign.md section 6): the Act 1 campaign spine as a path of chapters (done, current, next, open,
// waiting, skipped, held), the Up next block (next chapter, its readiness band, the level meter with an ETA, Go now) and
// a detail block for the tapped chapter (reward, readiness, Go now, Skip, Hold, Ask or Auto), from GET /api/campaign.
// Every button posts one request to /api/campaign/intent. Nothing is ever locked: Go now is offered on every open chapter.
const BAND = { too_early: 'too early', ready: 'ready at the floor', comfortable: 'comfortable' };
const LEVEL = { too_early: 'bad', ready: 'warn', comfortable: '' };

export function bandText(c) {
  if (!c || !c.band) return '';
  return BAND[c.band] + ' (floor ' + c.floor + ', recommended ' + c.recommended + ')';
}

export function etaText(m) {
  if (!m) return '';
  const pct = Math.round((m.pct || 0) * 100);
  let s = 'level ' + m.from + ' to ' + m.to + ': ' + pct + '%';
  if (m.etaSeconds) s += ', about ' + Math.max(1, Math.round(m.etaSeconds / 60)) + ' min';
  return s;
}

// upNext is the chapter the decision is about, or the first open one.
export function upNext(v) {
  const cs = (v && v.chapters) || [];
  const id = v && v.decision && v.decision.chapter;
  return cs.find((c) => c.id === id) || cs.find((c) => c.state === 'open') || null;
}

// actions are the buttons of one chapter: [label, request body].
export function actions(c, v) {
  if (!c || c.state === 'done') return [];
  const out = [];
  if (c.asked) out.push(['Go now', { chapter: c.id, answer: true }], ['Not yet', { chapter: c.id, answer: false }]);
  else if (c.state !== 'current') out.push(['Go now', { chapter: c.id, goNow: true }]);
  if (c.state === 'current' && v && v.decision && v.decision.why && v.decision.why.indexOf('your order') === 0) out.push(['Cancel', { chapter: c.id, goNow: false }]);
  if (c.state === 'skipped' || c.state === 'held') out.push(['Resume', { chapter: c.id, intent: 'auto' }]);
  else {
    if (c.optional) out.push(['Skip', { chapter: c.id, intent: 'skip' }]);
    out.push(['Hold', { chapter: c.id, intent: 'hold' }]);
    out.push(c.intent === 'ask' ? ['Auto', { chapter: c.id, intent: 'auto' }] : ['Ask first', { chapter: c.id, intent: 'ask' }]);
  }
  return out;
}

export function mount(root, ctx) {
  const path = ctx.el('ol', 'path');
  path.setAttribute('aria-label', 'Act 1 chapters');
  const up = ctx.el('div', 'up');
  up.dataset.role = 'upnext';
  const upTitle = ctx.el('p', 'ttl');
  const upWhy = ctx.el('p', 'd-hint');
  const bar = ctx.ui.meter('level', 'level');
  const upBtns = ctx.el('div', 'btns');
  up.append(ctx.el('p', 'sub', 'Up next'), upTitle, upWhy, bar.root, upBtns);
  const det = ctx.el('div', 'det');
  det.dataset.role = 'detail';
  const detTitle = ctx.el('p', 'ttl');
  const detText = ctx.el('p', 'd-hint');
  const detBand = ctx.el('p', 'd-hint');
  const detBtns = ctx.el('div', 'btns');
  det.append(detTitle, detText, detBand, detBtns);
  ctx.body.append(path, up, det);
  let sel = '', last = null;
  const send = (body, b) => {
    b.disabled = true;
    ctx.post('/api/campaign/intent', body).then((j) => {
      if (j && !j.ok) ctx.toast('warn', j.error || 'the journey did not take it');
      b.disabled = false;
    });
  };
  const buttons = (box, c) => {
    ctx.ui.clear(box);
    actions(c, last).forEach(([label, body]) => {
      const b = ctx.ui.btn(label, { size: 'sm', key: c.id + ':' + label });
      b.addEventListener('click', () => send(body, b));
      box.appendChild(b);
    });
  };
  const render = (v) => {
    const cs = v.chapters || [];
    ctx.ui.clear(path);
    cs.forEach((c, i) => {
      const li = ctx.el('li', 'step');
      li.dataset.key = c.id;
      li.dataset.state = c.state;
      if (c.band) li.dataset.band = c.band;
      if (c.id === sel) li.dataset.sel = '1';
      const b = ctx.el('button', 'pip', String(i + 1));
      b.type = 'button';
      b.title = c.title + ': ' + c.state + (c.band ? ', ' + BAND[c.band] : '');
      b.setAttribute('aria-label', b.title);
      b.addEventListener('click', () => { sel = sel === c.id ? '' : c.id; render(last); });
      li.append(b, ctx.el('span', 'lbl', c.title));
      path.appendChild(li);
    });
    const done = cs.filter((c) => c.state === 'done' || c.state === 'skipped').length;
    ctx.meta.textContent = done + ' of ' + cs.length + (v.mode && v.mode !== 'guided' ? ' (' + v.mode + ')' : '');
    const n = upNext(v), d = v.decision;
    up.hidden = !n;
    if (n) {
      up.dataset.state = n.state;
      upTitle.textContent = n.title + (n.band ? ': ' + BAND[n.band] : '');
      upWhy.textContent = (d && d.chapter === n.id && d.why) || n.why;
      bar.root.hidden = !v.meter;
      if (v.meter) bar.set(Math.round((v.meter.pct || 0) * 100), etaText(v.meter), LEVEL[n.band] || '');
      buttons(upBtns, n);
    }
    const c = cs.find((x) => x.id === sel);
    det.hidden = !c;
    if (c) {
      detTitle.textContent = c.title + ' (' + c.area + ')';
      detText.textContent = 'Reward: ' + c.reward + '. ' + c.why;
      detBand.textContent = [bandText(c)].concat(c.reasons || []).filter(Boolean).join('; ');
      buttons(detBtns, c);
    }
  };
  return {
    update(v) {
      if (!v || !v.chapters || !v.chapters.length) { ctx.hide(true); return; }
      ctx.hide(false);
      last = v;
      render(v);
    },
  };
}
