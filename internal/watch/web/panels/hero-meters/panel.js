// Life and mana in the top band (the approved mockup's status tile): who and where in one line, two thin meters with
// their numbers, from GET /api/hero. A value the bot has not reported shows as a dash, never as zero.
export function level(cur, max) {
  if (!max) return '';
  const p = (100 * cur) / max;
  return p < 25 ? 'bad' : p < 50 ? 'warn' : '';
}

export function mount(root, ctx) {
  const where = ctx.el('span', 'where');
  const life = ctx.ui.meter('life', 'life'), mana = ctx.ui.meter('mana', 'mana');
  life.root.classList.add('thin');
  mana.root.classList.add('thin');
  life.root.dataset.key = 'life';
  mana.root.dataset.key = 'mana';
  root.append(where, life.root, mana.root);
  return {
    update(h) {
      if (!h || !h.known) { ctx.hide(true); return; }
      ctx.hide(false);
      where.textContent = [h.name, h.class ? h.class + (h.level ? ' lvl ' + h.level : '') : '', h.area].filter(Boolean).join(', ');
      life.set(h.maxLife ? (100 * h.life) / h.maxLife : 0, h.maxLife ? h.life + '/' + h.maxLife : '-', level(h.life, h.maxLife));
      mana.set(h.maxMana ? (100 * h.mana) / h.maxMana : 0, h.maxMana ? h.mana + '/' + h.maxMana : '-');
    },
  };
}
