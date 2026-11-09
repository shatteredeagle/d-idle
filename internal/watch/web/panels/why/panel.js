// The Why card (command centre): the reason now, what would change it, what comes next (GET /api/hero, why). It
// replaces the single readout that sat in the top band.
export function mount(root, ctx) {
  const rows = [['Now', 'now'], ['Would change it', 'change'], ['Next', 'next']].map(([label, key]) => {
    const dt = ctx.el('dt', '', label), dd = ctx.el('dd', '');
    return { key, dt, dd };
  });
  const dl = ctx.el('dl', 'lines');
  rows.forEach((r) => dl.append(r.dt, r.dd));
  ctx.body.append(dl);
  return {
    update(h) {
      const w = h && h.why;
      if (!w || !w.now) { ctx.hide(true); return; }
      ctx.hide(false);
      rows.forEach((r) => {
        const t = r.key === 'next' ? (w.next || '').replace(/^Next: /, '') : w[r.key] || '';
        r.dd.textContent = t;
        r.dt.hidden = r.dd.hidden = !t;
      });
    },
  };
}
