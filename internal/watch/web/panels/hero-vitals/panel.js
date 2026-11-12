// Hero (command centre): the one hero tile. Who the hero is, life, mana and experience with numbers, gold on hand and in
// the stash, the four resistances and the readiness light, all from GET /api/hero. Merged from the old Hero and Hero stats
// tiles (the latter, panel id hero-detail, is an alias of this one: web/shell.json). Each number is shown once. A value
// the bot has not reported shows as a dash, never as zero; the stats block is null until the bot reports it and the tile
// then shows the basics alone.
const num = (n) => (n === null || n === undefined ? '-' : Number(n).toLocaleString('en-US').replace(/,/g, ' '));
const commas = (n) => Number(n).toLocaleString('en-US');

export function lifeLevel(cur, max) {
  if (!max) return '';
  const p = (100 * cur) / max;
  return p < 25 ? 'bad' : p < 50 ? 'warn' : '';
}

// xpText is the experience figure under the bar: "79,451 / 90,180 (40%)"; null while the hero's experience is not reported.
export function xpText(h) {
  if (!h || !Number.isFinite(h.xp) || !h.xpNext || h.xp < 0) return null;
  const pct = Number.isFinite(h.xpPct) ? ' (' + Math.floor(h.xpPct) + '%)' : '';
  return commas(h.xp) + ' / ' + commas(h.xpNext) + pct;
}

// rateText is the experience rate; "measuring" while the first minutes build it up.
export function rateText(h) {
  return Number.isFinite(h && h.xpPerHour) && h.xpPerHour > 0 ? commas(Math.round(h.xpPerHour)) + ' xp/h' : 'xp/h measuring';
}

export const LIGHTS = { green: ['ok', 'ready'], amber: ['warn', 'risky'], red: ['bad', 'not ready'] };

export function resistLevel(v) {
  return v < 0 ? 'bad' : ''; // a low positive number is normal for a young hero; only a negative one is marked
}

export function mount(root, ctx) {
  const who = ctx.el('p', 'who');
  const life = ctx.ui.meter('life', 'life'), mana = ctx.ui.meter('mana', 'mana'), xp = ctx.ui.meter('xp', 'xp');
  const bars = ctx.el('div', 'bars');
  bars.append(life.root, mana.root, xp.root);
  const rate = ctx.el('p', 'd-hint rate');
  const gold = ctx.ui.stat('gold'), stash = ctx.ui.stat('stash');
  const stats = ctx.el('div', 'd-stats');
  stats.append(gold.root, stash.root);
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
  ctx.body.append(who, bars, rate, stats, res, ready);
  return {
    update(h) {
      if (!h) { ctx.hide(true); return; }
      ctx.hide(false);
      if (!h.known) { who.textContent = 'Waiting for the first look at the hero.'; return; }
      who.textContent = [h.name, h.class, h.level ? 'level ' + h.level : '', h.difficulty].filter(Boolean).join(' · ');
      life.set(h.maxLife ? (100 * h.life) / h.maxLife : 0, h.maxLife ? h.life + '/' + h.maxLife : '-', lifeLevel(h.life, h.maxLife));
      mana.set(h.maxMana ? (100 * h.mana) / h.maxMana : 0, h.maxMana ? h.mana + '/' + h.maxMana : '-');
      const xt = xpText(h);
      xp.set(Number.isFinite(h.xpPct) ? h.xpPct : 0, xt || '-');
      rate.textContent = xt ? rateText(h) : '';
      rate.hidden = !xt;
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
