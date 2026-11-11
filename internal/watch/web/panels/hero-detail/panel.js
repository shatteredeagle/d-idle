// Hero stats (the approved mockup's "Hero" tile): level and the way to the next one, gold, the four resistances and
// the readiness light for the hero's area, from GET /api/hero. The stats block is null until the bot reports it; the
// tile then shows the basics alone.
const num = (n) => (n === null || n === undefined ? '-' : Number(n).toLocaleString('en-US').replace(/,/g, ' '));
export const LIGHTS = { green: ['ok', 'ready'], amber: ['warn', 'risky'], red: ['bad', 'not ready'] };

export function resistLevel(v) {
  return v < 0 ? 'bad' : ''; // a low positive number is normal for a young hero; only a negative one is marked
}

export function mount(root, ctx) {
  const lvl = ctx.ui.stat('level'), xp = ctx.ui.stat('next level'), gold = ctx.ui.stat('gold'), stash = ctx.ui.stat('stash');
  const stats = ctx.el('div', 'd-stats');
  stats.append(lvl.root, xp.root, gold.root, stash.root);
  const res = ctx.el('div', 'd-strip res');
  res.setAttribute('aria-label', 'Resistances');
  const chips = {};
  for (const k of ['fire', 'cold', 'light', 'poison']) {
    const c = ctx.el('span', 'd-chip'), b = ctx.el('b');
    c.dataset.key = k;
    c.append(k + ' ', b);
    chips[k] = { c, b };
    res.appendChild(c);
  }
  const ready = ctx.el('p', 'ready');
  const light = ctx.ui.chip('', '');
  light.dataset.key = 'readiness';
  const why = ctx.el('span', 'd-hint');
  ready.append(light, why);
  ctx.body.append(stats, res, ready);
  return {
    update(h) {
      if (!h) { ctx.hide(true); return; }
      ctx.hide(false);
      ctx.meta.textContent = h.known ? [h.class, h.difficulty].filter(Boolean).join(' · ') : '';
      lvl.set(h.level ? String(h.level) : '-', h.name || '');
      const left = h.xpNext && h.xp <= h.xpNext ? h.xpNext - h.xp : null;
      xp.set(h.xpPct === null || h.xpPct === undefined ? '-' : h.xpPct.toFixed(1) + '%', left === null ? '' : num(left) + ' xp to go');
      gold.set(num(h.gold), h.gold === null ? 'not reported' : 'on the hero');
      stash.set(num(h.stashGold), h.stashGold === null ? 'not reported' : 'in the stash');
      const s = h.stats;
      res.hidden = !s;
      if (s) for (const k in chips) { chips[k].b.textContent = s.resists[k] + '%'; const l = resistLevel(s.resists[k]); if (l) chips[k].c.dataset.level = l; else delete chips[k].c.dataset.level; }
      const r = s && s.readiness, l = r && LIGHTS[r.light];
      ready.hidden = !l;
      if (l) {
        light.textContent = l[1] + (r.area ? ' for ' + r.area : '');
        light.dataset.level = l[0];
        why.textContent = r.reason || '';
      }
    },
  };
}
