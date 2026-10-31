import { h, clear, num, bar } from '../lib.js';

export default {
  id: 'character',
  span: 5,
  render(el, d, ctx) {
    clear(el);
    ctx.aside.textContent = d.difficulty || '';
    const skill = (s) => h('tr', {},
      h('td', {}, s.name,
        s.id === d.leftSkill ? h('span', { class: 'sel', title: 'Selected left skill', text: 'L' }) : null,
        s.id === d.rightSkill ? h('span', { class: 'sel', title: 'Selected right skill', text: 'R' }) : null),
      h('td', { class: 'num', text: s.level }),
      h('td', { class: 'hk', text: s.hotkey || '' }));
    const res = (name, v, c) => h('span', { style: { '--rc': c }, class: v < 0 ? 'neg' : '', text: `${name} ${v}%` });
    const R = d.resists || {};
    const skills = (d.skills || []).slice().sort((a, b) => b.level - a.level);
    el.append(
      h('div', { class: 'row' }, h('b', { text: d.name || '(no name)' }), h('span', { class: 'muted', text: `Level ${d.level} ${d.class || ''}` }),
        d.statPoints ? h('span', { class: 'pill hot', text: d.statPoints + ' stat pts' }) : null,
        d.skillPoints ? h('span', { class: 'pill hot', text: d.skillPoints + ' skill pts' }) : null),
      h('div', { class: 'vitals' },
        bar('life', d.life, d.maxLife, `Life ${num(d.life)} / ${num(d.maxLife)}`),
        bar('mana', d.mana, d.maxMana, `Mana ${num(d.mana)} / ${num(d.maxMana)}`),
        d.maxStamina ? bar('stam', d.stamina, d.maxStamina, `Stamina ${num(d.stamina)} / ${num(d.maxStamina)}`) : null,
        d.expNext ? bar('', d.exp - d.expPrev, d.expNext - d.expPrev, `XP ${num(d.exp)} / ${num(d.expNext)}`) : null),
      h('div', { class: 'stats' },
        ...[['Str', d.str], ['Dex', d.dex], ['Vit', d.vit], ['Ene', d.ene]].map(([k, v]) => h('div', {}, h('b', { text: v }), h('span', { text: k })))),
      h('h3', { text: 'Resistances' }),
      h('div', { class: 'res' }, res('Fire', R.fire, 'var(--life)'), res('Cold', R.cold, 'var(--mana)'), res('Light', R.light, 'var(--warn)'), res('Poison', R.poison, 'var(--stam)')),
      h('h3', { text: 'Skills' }),
      skills.length
        ? h('div', { class: 'tablewrap' }, h('table', {}, h('thead', {}, h('tr', {}, h('th', { text: 'Skill' }), h('th', { text: 'Lvl' }), h('th', { text: 'Key' }))), h('tbody', {}, skills.map(skill))))
        : h('p', { class: 'empty-note', text: 'No skills allocated yet.' }));
  },
};
