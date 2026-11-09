// Tally (command centre; docs/tallystrip.md): the balance, this trip's gain and what it buys next, from GET
// /api/tally. The long strip text that crowded the top band lives here now, as a hint.
export function mount(root, ctx) {
  const bal = ctx.ui.stat('banked');
  const trip = ctx.ui.stat('this trip');
  const stats = ctx.el('div', 'd-stats');
  stats.append(bal.root, trip.root);
  const next = ctx.el('p', 'd-hint');
  ctx.body.append(stats, next);
  return {
    update(t) {
      if (!t || !t.enabled) { ctx.hide(true); return; }
      ctx.hide(false);
      bal.set(String(t.balance), t.sealed ? t.sealed + ' sealed' : '');
      const e = t.expedition || t.last;
      trip.set(e ? (e.net >= 0 ? '+' : '') + e.net : '-', t.expedition ? 'under way' : t.last ? 'last trip' : 'no trip yet');
      const n = (t.next || [])[0];
      next.textContent = t.error || (n ? 'Next: ' + (n.name || n.id) + (n.cost ? ' for ' + n.cost + ' tally' : '') + (n.affordable ? ', affordable now' : '') : t.strip || '');
    },
  };
}
