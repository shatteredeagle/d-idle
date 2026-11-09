// Doing now (command centre): what the hero does this moment in one line, the order under way with its progress, and
// the next orders in the queue, from GET /api/hero.
export function mount(root, ctx) {
  const now = ctx.el('p', 'now');
  now.setAttribute('role', 'status');
  now.setAttribute('aria-live', 'polite');
  const goal = ctx.el('div', 'goal');
  const goalText = ctx.el('p', 'gtext');
  const bar = ctx.ui.meter('order', 'order');
  const goalNote = ctx.el('p', 'd-hint');
  goal.append(goalText, bar.root, goalNote);
  const nextH = ctx.el('p', 'sub', 'Up next');
  const next = ctx.el('ol', 'd-list');
  ctx.body.append(now, goal, nextH, next);
  return {
    update(h) {
      if (!h) { ctx.hide(true); return; }
      ctx.hide(false);
      const line = h.activity || h.stateLabel || 'Waiting for the hero.';
      now.textContent = line.charAt(0).toUpperCase() + line.slice(1);
      ctx.meta.textContent = h.area || '';
      const g = h.goal;
      if (g) {
        goal.hidden = false;
        goalText.textContent = g.text;
        bar.root.hidden = g.pct === null;
        if (g.pct !== null) bar.set(g.pct, Math.round(g.pct) + '%');
        const bits = [];
        if (g.leftMin !== null) bits.push('about ' + g.leftMin + ' min left');
        if (g.kills) bits.push(g.kills + ' kills');
        if (g.xp) bits.push('+' + g.xp + ' xp');
        goalNote.textContent = bits.length ? bits.join(', ') + ' since the order.' : 'Order under way.';
      } else {
        goal.hidden = false;
        goalText.textContent = 'Following its own plan';
        bar.root.hidden = true;
        goalNote.textContent = 'No order from you. Tap an area on the map below to send the hero.';
      }
      ctx.ui.clear(next);
      (h.queue || []).forEach((t, i) => {
        const li = ctx.el('li');
        li.append(ctx.ui.badge(String(i + 1)), ctx.el('span', 'txt', t));
        next.appendChild(li);
      });
      if (!next.firstChild) {
        const li = ctx.el('li');
        li.append(ctx.el('span'), ctx.el('span', 'txt d-muted', 'Nothing queued.'));
        next.appendChild(li);
      }
    },
  };
}
