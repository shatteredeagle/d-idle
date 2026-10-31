import { h, clear } from '../lib.js';

// Failsafe sentinel (cmd/d2sentinel): per-instance health, what the rules are timing, recent actions.
export default {
  id: 'sentinel',
  span: 12,
  render(el, d, ctx) {
    clear(el);
    ctx.aside.textContent = d.mode || '';
    const bad = (d.instances || []).some((i) => i.health !== 'ok');
    if (bad) el.append(h('div', { class: 'pill hot', text: d.summary }));
    const rows = (d.instances || []).map((i) => h('tr', {},
      h('td', { text: i.id + (i.character ? ' ' + i.character : '') }),
      h('td', { class: i.health === 'ok' ? '' : 'neg', text: i.health + (i.acting ? ' ACTING ' + i.acting : '') }),
      h('td', { text: i.line || '' })));
    el.append(h('div', { class: 'tablewrap' }, h('table', {}, h('tbody', {}, rows))));
    const rec = (d.recent || []).slice(0, 12).map((r) => h('tr', {},
      h('td', { text: (r.time || '').slice(11, 19) }),
      h('td', { text: r.instance || '' }),
      h('td', { text: r.kind }),
      h('td', { text: (r.rule || '') + ' > ' + (r.action || '') }),
      h('td', { class: 'muted', text: (r.result && r.result.detail) || r.evidence || '' })));
    if (rec.length) el.append(h('h3', { text: 'Recent failsafe events' }), h('div', { class: 'tablewrap' }, h('table', {}, h('tbody', {}, rec))));
  },
};
