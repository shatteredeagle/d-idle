// The status sentence of the cockpit band (J-second-attempt.md section 2): a state chip, the place and the reason on one
// line, cut with an ellipsis; a click opens a popover with what is happening now, what would change it, what is next, the
// uptime and the hero's voice line. From GET /api/glance (sentence). Only the sentence is aria-live. The margin of
// survival shows as words (safe, risky, too hard), never as a number.
const TONES = ['ok', 'warn', 'danger', 'info'];

export function tone(t) { return TONES.includes(t) ? t : 'info'; }

// text is the line as one string: the title of the button, so a cut line can still be read in full.
export function text(s) { return [s.place, s.reason].filter(Boolean).join(' · '); }

// rows are the popover's lines: label and text; a line with nothing to say is left out.
export function rows(s) {
  const w = s.why || {};
  return [
    ['Now', w.now || s.reason],
    ['Would change it', w.change],
    ['Next', (w.next || '').replace(/^Next:\s*/i, '')],
    ['Risk here', s.margin],
    ['Uptime', s.uptime],
    ['Voice', s.voice],
  ].filter((r) => r[1]);
}

export function mount(root, ctx) {
  const say = ctx.el('button', 'say');
  say.type = 'button';
  say.dataset.key = 'say';
  say.setAttribute('aria-expanded', 'false');
  say.setAttribute('aria-haspopup', 'true');
  const line = ctx.el('span', 'line');
  line.setAttribute('aria-live', 'polite');
  const word = ctx.el('span', 'word');
  const txt = ctx.el('span', 'txt');
  line.append(word, txt);
  say.append(line);
  const pop = ctx.el('div', 'pop d-bandpop');
  pop.hidden = true;
  pop.tabIndex = -1;
  pop.setAttribute('role', 'dialog');
  pop.setAttribute('aria-label', 'Why the hero does this');
  root.append(say, pop);

  const open = (on) => { pop.hidden = !on; say.setAttribute('aria-expanded', String(on)); };
  say.addEventListener('click', () => open(pop.hidden));
  root.addEventListener('keydown', (e) => { if (e.key === 'Escape' && !pop.hidden) { open(false); say.focus(); } });
  // a click or a tab outside moves the focus out of the panel: that closes it (no listener on the page)
  root.addEventListener('focusout', (e) => { if (!e.relatedTarget || !root.contains(e.relatedTarget)) open(false); });

  // the last drawing: an unchanged reply redraws nothing, so the live region is not announced again and an open popover keeps its focus
  let sig = '';
  function show(t, w, place, reason, why) {
    const next = JSON.stringify([t, w, place, reason, why]);
    if (next === sig) return;
    sig = next;
    say.dataset.tone = t;
    word.textContent = w;
    txt.textContent = '';
    if (place) txt.append(ctx.el('b', 'place', place));
    if (reason) txt.append(ctx.el('span', 'reason', (place ? ' · ' : '') + reason));
    say.title = [place, reason].filter(Boolean).join(' · ');
    pop.textContent = '';
    const dl = ctx.el('dl', 'rows');
    for (const [k, v] of why) dl.append(ctx.el('dt', '', k), ctx.el('dd', '', v));
    pop.append(dl);
    const a = ctx.el('a', 'more', 'Open the log');
    a.href = '#log';
    // the shell's drawer API (slice S6); before it exists the link is only the anchor
    a.addEventListener('click', (e) => { if (ctx.openDrawer) { e.preventDefault(); open(false); ctx.openDrawer('log'); } });
    pop.append(a);
  }
  show('info', '...', '', 'waiting for the bot', []);
  let good = false;
  return {
    update(v, err) {
      const s = v && v.sentence;
      // one slow or failed poll after good data is a blip (a busy bot, a page just restored from the back-forward cache): keep the
      // sentence, say "No signal" only when two polls in a row failed
      if (!s && good && (!err || !(err.fails >= 2))) return;
      if (s) good = true;
      if (!s) { show('warn', 'No signal', '', 'the page cannot reach the bot', []); ctx.state('error'); return; }
      show(tone(s.tone), s.word || '', s.place || '', s.reason || '', rows(s));
      ctx.state('ready');
    },
  };
}
