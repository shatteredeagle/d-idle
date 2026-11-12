// The journey chip of the top band (docs/campaign.md section 6): the current chapter, its readiness band and the next
// chapter in one badge, from GET /api/campaign. It links to the Journey panel; with the route off it is not shown.
const BAND = { too_early: 'too early', ready: 'ready', comfortable: 'comfortable' };
const TONE = { too_early: 'bad', ready: 'warn', comfortable: 'good' };

// line is the chip text: "Find Deckard Cain: comfortable, next: The Countess".
export function line(v) {
  const cs = (v && v.chapters) || [];
  const id = v && v.decision && v.decision.chapter;
  const cur = cs.find((c) => c.id === id) || cs.find((c) => c.state === 'open');
  if (!cur) return { text: v && v.decision && v.decision.action === 'done' ? 'Act 1 journey done' : '', tone: '' };
  const after = cs.slice(cs.indexOf(cur) + 1).find((c) => c.state !== 'done' && c.state !== 'skipped' && c.state !== 'held');
  let text = cur.title + (cur.band ? ': ' + BAND[cur.band] : '');
  if (after) text += ', next: ' + after.title;
  return { text, tone: TONE[cur.band] || '' };
}

export function mount(root, ctx) {
  const a = ctx.el('a', 'd-badge');
  a.href = '#p-journey-title'; // the Journey panel's heading (shell.js gives every panel title an id)
  a.dataset.key = 'journey';
  root.appendChild(a);
  return {
    update(v) {
      const l = line(v);
      if (!l.text) { ctx.hide(true); return; }
      ctx.hide(false);
      a.textContent = l.text;
      if (l.tone) a.dataset.tone = l.tone; else delete a.dataset.tone;
      a.title = (v.decision && v.decision.why) || '';
    },
  };
}
