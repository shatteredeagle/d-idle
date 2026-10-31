import { h, clear, num, compact, dur, bar } from '../lib.js';

export default {
  id: 'overview',
  span: 12,
  render(el, d, ctx) {
    clear(el);
    ctx.aside.textContent = d.inGame ? (d.inTown ? 'in town' : 'in the field') : 'not in a game';
    const r = d.rates;
    const five = r[0], hour = r[1], sess = r[2];
    const mark = (rs) => (rs.partial ? '*' : '');
    const g = d.goal || {};

    const act = h('div', {},
      h('div', { class: 'act' }, d.activity || 'idle', d.detail ? h('small', { text: d.detail }) : null),
      h('div', { class: 'muted', text: d.areaName ? 'Area: ' + d.areaName : 'No area' }));

    const goal = h('div', {},
      h('div', { class: 'muted', text: 'Goal' }),
      h('div', { text: g.name || 'none set' }),
      g.name ? bar('', g.progress, 1, Math.round((g.progress || 0) * 100) + '%') : null,
      g.detail ? h('div', { class: 'muted', text: g.detail }) : null);

    const lvl = h('div', {},
      h('div', { class: 'muted', text: 'Level ' + (d.level || '-') }),
      d.expNext
        ? bar('', d.exp - d.expThisLevel, d.expNext - d.expThisLevel, `${num(d.exp)} / ${num(d.expNext)}`)
        : h('div', { text: num(d.exp) + ' xp' }),
      h('div', { class: 'muted', text: 'Next level in ' + (d.etaSecs >= 0 ? dur(d.etaSecs) : 'unknown') }));

    const xph = h('div', {},
      h('div', { class: 'muted', text: 'XP per hour (5 min)' }),
      h('div', { class: 'rate-hero' }, num(five.xp), h('small', { text: mark(five) ? ' /h (partial window)' : ' /h' })));

    const row = (label, key, f = num) => h('tr', {}, h('td', { text: label }),
      h('td', { class: 'num', text: f(five[key]) + mark(five) }),
      h('td', { class: 'num', text: f(hour[key]) + mark(hour) }),
      h('td', { class: 'num', text: f(sess[key]) }));
    const table = h('div', { class: 'tablewrap' }, h('table', {},
      h('thead', {}, h('tr', {}, h('th', { text: 'Per hour' }), h('th', { text: '5 min' }), h('th', { text: '1 h' }), h('th', { text: 'Session' }))),
      h('tbody', {}, row('XP', 'xp'), row('Kills', 'kills'), row('Gold', 'gold'), row('Drops', 'drops'))),
      h('div', { class: 'muted', style: { fontSize: '12px' }, text: '* window not yet full: rate uses elapsed time' }));

    const tile = (k, v, s) => h('div', { class: 'tile' }, h('div', { class: 'k', text: k }), h('div', { class: 'v', text: v }), s ? h('div', { class: 's', text: s }) : null);
    const pb = d.potionsBy || [0, 0, 0];
    const tiles = h('div', { class: 'tiles' },
      tile('Kills', num(d.totalKills), 'this session'),
      tile('XP gained', compact(d.totalXP), `${d.levelUps} level-up${d.levelUps === 1 ? '' : 's'}`),
      tile('Gold gained', compact(d.totalGold)),
      tile('Deaths', num(d.deaths)),
      tile('Potions used', num(d.potions), `${pb[0]} hp, ${pb[1]} mp, ${pb[2]} rejuv`),
      tile('Runs', num(d.runs), d.runs ? 'avg ' + dur(d.avgRunSecs) : 'none finished'),
      tile('Current run', d.curRun ? dur(d.curRunSecs) : '-', d.curRun || ''),
      tile('Session', dur(d.sessionSecs)));

    el.append(h('div', { class: 'hero' }, act, goal, lvl, xph), h('h3', { text: 'Rates' }), table, tiles);
  },
};
