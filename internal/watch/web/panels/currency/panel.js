// Marks (command centre; docs/currency.md): the currencies the header strip showed, as chips, from GET /api/currency.
// Hidden unless the bot runs in d.idle mode (the route is absent or off); the spend list stays in the Spend card.
export function mount(root, ctx) {
  const strip = ctx.el('div', 'd-strip');
  ctx.body.append(strip);
  return {
    update(c) {
      if (!c || !c.enabled) { ctx.hide(true); return; }
      ctx.hide(false);
      ctx.ui.clear(strip);
      (c.currencies || []).forEach((r) => {
        const chip = ctx.ui.chip(r.name + ' ');
        chip.appendChild(ctx.el('b', '', r.amount === null ? '-' : String(r.amount)));
        chip.appendChild(ctx.el('span', 'd-muted', ' ' + r.label));
        chip.title = r.blurb || '';
        strip.appendChild(chip);
      });
      ctx.meta.textContent = c.error || '';
    },
  };
}
