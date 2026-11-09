// Gear condition (command centre; docs/repair.md): one chip per worn slot (current/max, amber under the warn percent,
// red and BROKEN at zero), a one-line summary (worst slot, broken count, repair estimate) and Repair now or Cancel,
// from GET /api/repair. Replaces the strip that sat under the top band (gear.js).
export function summary(v) {
  const gear = v.gear || [];
  const broken = gear.filter((g) => g.broken).length;
  let worst = null;
  gear.forEach((g) => { if (!worst || g.pct < worst.pct) worst = g; });
  const bits = [];
  if (broken) bits.push(broken + ' broken');
  if (worst && !worst.broken) bits.push('worst ' + worst.slot + ' ' + worst.pct + '%');
  if (v.estimate) bits.push('repair about ' + v.estimate + ' gold');
  return bits.join(', ') || 'all fine';
}

function reqText(q) {
  if (!q) return '';
  if (q.state === 'requested') return 'Repair requested: done at the next town visit.' + (q.note ? ' (' + q.note + ')' : '');
  if (q.state === 'done') return 'Repair done.';
  return '';
}

export function mount(root, ctx) {
  const chips = ctx.el('div', 'd-strip');
  const note = ctx.el('p', 'd-hint');
  const act = ctx.ui.btn('Repair now', { size: 'sm', key: 'repair' });
  ctx.actions.append(act);
  ctx.body.append(chips, note);
  let pending = false;
  act.addEventListener('click', () => {
    act.disabled = true;
    ctx.post(pending ? '/api/repair/cancel' : '/api/repair/request', {}).then((j) => {
      if (j && !j.ok) ctx.toast('warn', j.error || 'repair request failed');
      act.disabled = false;
    });
  });
  return {
    update(v) {
      const gear = (v && v.gear) || [];
      if (!gear.length) { ctx.hide(true); return; }
      ctx.hide(false);
      ctx.meta.textContent = summary(v);
      ctx.ui.clear(chips);
      gear.forEach((g) => {
        const c = ctx.ui.chip(g.slot + ' ', g.broken ? 'bad' : g.warn ? 'warn' : '');
        c.appendChild(ctx.el('b', '', g.dur + '/' + g.max));
        if (g.broken) c.appendChild(ctx.el('span', '', 'BROKEN'));
        c.title = g.name + ' ' + g.pct + '%';
        chips.appendChild(c);
      });
      pending = !!(v.request && v.request.state === 'requested');
      act.textContent = pending ? 'Cancel repair' : 'Repair now';
      note.textContent = [reqText(v.request), v.worst || ''].filter(Boolean).join(' ');
      note.hidden = !note.textContent;
    },
  };
}
