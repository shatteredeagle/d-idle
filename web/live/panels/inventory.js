import { h, clear, num, tip, setStyle, QUALITY, qclass } from '../lib.js';

function label(it, small) {
  return small ? (it.code || it.name || '?').slice(0, 4).toUpperCase() : (it.name || it.code || '?');
}
function describe(it) {
  return `${it.name} (${QUALITY[it.quality] || 'normal'}${it.ethereal ? ', ethereal' : ''}${it.sockets ? `, ${it.sockets} sockets` : ''})`;
}
function tipBody(it) {
  return h('div', { class: qclass(it.quality) },
    h('div', { class: 't1 qt', text: it.name }),
    h('div', { class: 't2', text: [QUALITY[it.quality] || 'Normal', it.ethereal ? 'Ethereal' : '', it.level ? 'ilvl ' + it.level : '', it.identified === false ? 'Unidentified' : ''].filter(Boolean).join(' / ') }),
    it.sockets ? h('div', { class: 't2', text: `Sockets: ${it.filled || 0} / ${it.sockets}` }) : null,
    ...(it.props || []).map((p) => h('div', { class: 'p', text: p })));
}
function itemEl(it, extra) {
  const small = (it.w || 1) * (it.h || 1) === 1;
  const el = h('button', { class: `item ${qclass(it.quality)}${it.ethereal ? ' eth' : ''}`, type: 'button', 'aria-label': describe(it) },
    label(it, small), it.sockets ? h('span', { class: 'soc', 'aria-hidden': 'true', text: '◦'.repeat(it.sockets) }) : null);
  setStyle(el, extra || {});
  tip(el, () => tipBody(it));
  return el;
}
function itemGrid(cols, rows, items, flipY) {
  const g = h('div', { class: 'igrid', style: { '--cols': cols } });
  for (let i = 0; i < cols * rows; i++) g.append(h('div', { class: 'empty' }));
  for (const it of items || []) {
    const w = it.w || 1, hh = it.h || 1;
    const row = flipY ? rows - it.y - hh + 1 : it.y + 1;
    g.append(itemEl(it, { gridColumn: `${it.x + 1} / span ${w}`, gridRow: `${row} / span ${hh}` }));
  }
  return g;
}

export default {
  id: 'inventory',
  span: 7,
  render(el, d, ctx) {
    clear(el);
    ctx.aside.textContent = `${num(d.gold)} gold carried, ${num(d.stashGold)} in stash`;
    const doll = h('div', { class: 'doll' });
    for (const s of ['head', 'neck', 'rhand', 'torso', 'lhand', 'gloves', 'belt', 'feet', 'rring', 'lring']) {
      const it = d.equipment && d.equipment[s];
      const cell = h('div', { class: 'slot', style: { gridArea: s } }, it ? null : s.replace(/^[rl]/, (c) => c + ' ').trim());
      if (it) { const b = itemEl(it, { width: '100%', height: '100%' }); cell.append(b); cell.style.border = '0'; }
      doll.append(cell);
    }
    const beltRows = d.beltRows || 1;
    const belt = h('div', { class: 'igrid', style: { '--cols': 4 } });
    for (let i = 0; i < 4 * beltRows; i++) belt.append(h('div', { class: 'empty' }));
    for (const it of d.belt || []) belt.append(itemEl(it, { gridColumn: it.x + 1, gridRow: beltRows - it.y }));
    el.append(h('div', { class: 'inv' },
      h('div', { class: 'invcol' }, h('h3', { text: 'Equipment' }), doll),
      h('div', { class: 'invcol' },
        h('h3', { text: 'Belt' }), belt,
        h('h3', { text: 'Inventory' }), itemGrid(10, 4, d.inventory)),
      h('div', { class: 'invcol' }, h('details', { open: true }, h('summary', { text: 'Stash' }), itemGrid(6, 8, d.stash)))));
  },
};
