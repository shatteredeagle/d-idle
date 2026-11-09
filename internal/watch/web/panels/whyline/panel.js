// The whyline (status band): the hero's own reason for what it does, from GET /api/whyline. Hidden while the route or a
// line is missing (docs/whyline.md); a new line flashes once (data-changed), the text is held by the server.
export function mount(root, ctx) {
  const p = ctx.el('p', 'd-ticker');
  p.setAttribute('role', 'status');
  p.setAttribute('aria-live', 'polite');
  p.setAttribute('aria-atomic', 'true');
  const text = ctx.el('span', 'txt');
  p.append(ctx.el('b', '', 'Why'), text);
  root.append(p);
  ctx.hide(true);
  return {
    update(v) {
      const t = v && v.text ? v.text : '';
      ctx.hide(!t);
      if (!t || text.textContent === t) return;
      text.textContent = t;
      p.removeAttribute('data-changed');
      void p.offsetWidth; // restart the flash
      p.dataset.changed = '';
    },
  };
}
