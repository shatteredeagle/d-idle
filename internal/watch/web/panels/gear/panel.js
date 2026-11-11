// Gear condition (command centre; docs/repair.md): one chip per worn slot (current/max, amber under the warn percent,
// red and BROKEN at zero), a one-line summary (worst slot, broken count, repair estimate) and Repair now or Cancel,
// from GET /api/repair. Replaces the strip that sat under the top band (gear.js).
// Panel version 2 adds the Farm gold order (docs/goldfarm.md): a button next to Repair now and one status line, both
// from the additive `farm` field of the same reply; the button posts {on} to /api/farmgold.
// Panel version 3 (docs/repair.md, the repair trip): the repair button is the reply's `trip.button` (Repair now, Farm
// until repair, Cancel, or off with its reason) and the line under it is `trip.line`; `trip.state` goes to
// data-state of the note. A reply without `trip` (an older bot) keeps the version 2 behaviour.
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

// The button of the repair part: the trip's own, or the version 2 one for a reply without it.
export function repairButton(v) {
  const t = v && v.trip;
  if (t && t.button) return { label: t.button.label, enabled: !!t.button.enabled, action: t.button.action, reason: t.button.reason || '' };
  const pending = !!(v && v.request && v.request.state === 'requested');
  return { label: pending ? 'Cancel repair' : 'Repair now', enabled: true, action: pending ? 'cancel' : 'request', reason: '' };
}

export function repairLine(v) {
  if (v && v.trip) return [v.trip.line || '', v.worst || ''].filter(Boolean).join(' ');
  return [reqText(v && v.request), (v && v.worst) || ''].filter(Boolean).join(' ');
}

export function mount(root, ctx) {
  const chips = ctx.el('div', 'd-strip');
  const note = ctx.el('p', 'd-hint');
  const act = ctx.ui.btn('Repair now', { size: 'sm', key: 'repair' });
  const farmBtn = ctx.ui.btn('Farm gold', { size: 'sm', key: 'farmgold' });
  const farmNote = ctx.el('p', 'd-hint');
  farmNote.dataset.role = 'farmgold';
  ctx.actions.append(farmBtn, act);
  ctx.body.append(chips, note, farmNote);
  let action = 'request', farming = false;
  farmBtn.addEventListener('click', () => {
    farmBtn.disabled = true;
    ctx.post('/api/farmgold', { on: !farming }).then((j) => {
      if (j && !j.ok) ctx.toast('warn', j.error || 'farm gold order failed');
      farmBtn.disabled = false;
    });
  });
  act.addEventListener('click', () => {
    if (action === 'none') return;
    act.disabled = true;
    ctx.post(action === 'cancel' ? '/api/repair/cancel' : '/api/repair/request', {}).then((j) => {
      if (j && !j.ok) ctx.toast('warn', j.error || 'repair request failed');
      act.disabled = false;
    });
  });
  return {
    update(v) {
      const gear = (v && v.gear) || [];
      const farm = v && v.farm;
      farming = !!(farm && farm.on);
      farmBtn.hidden = !farm;
      farmBtn.textContent = farming ? 'Stop farming' : 'Farm gold';
      farmNote.textContent = farm ? farm.text || '' : '';
      farmNote.hidden = !farmNote.textContent || (!farm.on && !farm.note) || !!(farm.on && farm.for === 'repair' && v.trip); // the repair line shows the progress
      if (!gear.length && !farm) { ctx.hide(true); return; }
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
      const rb = repairButton(v);
      action = rb.action;
      act.textContent = rb.label;
      act.disabled = !rb.enabled;
      act.title = rb.reason;
      act.dataset.state = v.trip ? v.trip.state : (v.request && v.request.state) || 'idle';
      note.dataset.state = act.dataset.state;
      note.textContent = repairLine(v);
      note.hidden = !note.textContent;
    },
  };
}
