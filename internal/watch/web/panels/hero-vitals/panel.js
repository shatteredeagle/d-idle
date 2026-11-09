// Hero vitals (command centre): name, class, level, life, mana and experience with numbers, gold on hand and in the
// stash, from GET /api/hero. A value the bot has not reported shows as a dash, never as zero.
const num = (n) => (n === null || n === undefined ? '-' : Number(n).toLocaleString('en-US').replace(/,/g, ' '));

export function lifeLevel(cur, max) {
  if (!max) return '';
  const p = (100 * cur) / max;
  return p < 25 ? 'bad' : p < 50 ? 'warn' : '';
}

export function mount(root, ctx) {
  const who = ctx.el('p', 'who');
  const life = ctx.ui.meter('life', 'life'), mana = ctx.ui.meter('mana', 'mana'), xp = ctx.ui.meter('xp', 'xp');
  const bars = ctx.el('div', 'bars');
  bars.append(life.root, mana.root, xp.root);
  const gold = ctx.ui.stat('gold'), stash = ctx.ui.stat('stash');
  const stats = ctx.el('div', 'd-stats');
  stats.append(gold.root, stash.root);
  ctx.body.append(who, bars, stats);
  return {
    update(h) {
      if (!h) { ctx.hide(true); return; }
      ctx.hide(false);
      if (!h.known) { who.textContent = 'Waiting for the first look at the hero.'; return; }
      who.textContent = [h.name, h.class, h.level ? 'level ' + h.level : '', h.difficulty].filter(Boolean).join(' · ');
      life.set(h.maxLife ? (100 * h.life) / h.maxLife : 0, h.maxLife ? h.life + '/' + h.maxLife : '-', lifeLevel(h.life, h.maxLife));
      mana.set(h.maxMana ? (100 * h.mana) / h.maxMana : 0, h.maxMana ? h.mana + '/' + h.maxMana : '-');
      xp.set(h.xpPct === null ? 0 : h.xpPct, h.xpPct === null ? '-' : h.xpPct.toFixed(1) + '%');
      xp.root.title = h.xpNext ? num(h.xp) + ' of ' + num(h.xpNext) + ' experience for the next level' : '';
      gold.set(num(h.gold), h.gold === null ? 'not reported' : 'on the hero');
      stash.set(num(h.stashGold), h.stashGold === null ? 'not reported' : 'in the stash');
    },
  };
}
