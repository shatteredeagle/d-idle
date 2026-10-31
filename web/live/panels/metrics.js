import { h, clear, num, compact, dur, lineChart, clock } from '../lib.js';

const CHARTS = [
  ['xp', 'XP per hour', 'var(--c1)', compact],
  ['kills', 'Kills per hour', 'var(--c2)', num],
  ['gold', 'Gold per hour', 'var(--c3)', compact],
];

function mk(el) {
  const st = { data: null, charts: {} };
  const wrap = h('div', { class: 'charts' });
  for (const [key, title, color] of CHARTS) {
    const host = h('div');
    const tbl = h('details', {}, h('summary', { text: 'Table view' }), h('div', { class: 'tablewrap' }));
    st.charts[key] = { host, tbl };
    wrap.append(h('div', { class: 'chart' }, h('h3', { text: title + ' (5 min average, per minute)' }), host, tbl));
  }
  st.areas = h('div'); st.drops = h('div');
  el.append(wrap, h('div', { class: 'cols2' }, h('div', {}, h('h3', { text: 'Per area' }), st.areas), h('div', {}, h('h3', { text: 'Recent drops' }), st.drops)));
  new ResizeObserver(() => st.data && draw(st)).observe(el);
  return st;
}

function draw(st) {
  for (const [key, title, color, fmt] of CHARTS) {
    const c = st.charts[key];
    const pts = st.data.series.map((p) => ({ x: new Date(p.t).getTime(), y: p[key] }));
    lineChart(c.host, { points: pts, label: title, color, fmtY: fmt, unit: ' /h' });
    const wrap = c.tbl.lastChild; clear(wrap);
    wrap.append(h('table', {}, h('thead', {}, h('tr', {}, h('th', { text: 'Time' }), h('th', { text: title }))),
      h('tbody', {}, pts.slice(-20).reverse().map((p) => h('tr', {}, h('td', { text: clock(p.x, false) }), h('td', { class: 'num', text: num(p.y) }))))));
  }
}

export default {
  id: 'metrics',
  span: 12,
  render(el, d) {
    const st = el._m || (el._m = (clear(el), mk(el)));
    st.data = d;
    draw(st);
    clear(st.areas);
    st.areas.append(d.areas.length ? h('div', { class: 'tablewrap' }, h('table', {},
      h('thead', {}, h('tr', {}, ...['Area', 'Visits', 'Time', 'Kills', 'Drops', 'XP', 'XP/h'].map((t) => h('th', { text: t })))),
      h('tbody', {}, d.areas.map((a) => h('tr', {}, h('td', { text: a.name || 'Area ' + a.area }), h('td', { class: 'num', text: a.visits }), h('td', { class: 'num', text: dur(a.secs) }),
        h('td', { class: 'num', text: a.kills }), h('td', { class: 'num', text: a.drops }), h('td', { class: 'num', text: compact(a.xp) }),
        h('td', { class: 'num', text: a.secs > 60 ? compact(a.xp / a.secs * 3600) : '-' }))))))
      : h('p', { class: 'empty-note', text: 'No areas visited yet.' }));
    clear(st.drops);
    const ds = d.drops.slice().reverse().slice(0, 15);
    st.drops.append(ds.length ? h('ul', { class: 'evlog' }, ds.map((e) => h('li', { 'data-k': 'drop' }, h('time', { text: clock(new Date(e.time).getTime(), false) }),
      h('span', { class: `kind q-${e.extra || 'normal'}`, style: { color: 'var(--qc)' }, text: e.extra || 'normal' }), h('span', { text: e.text.replace(/ dropped$/, '') }))))
      : h('p', { class: 'empty-note', text: 'No drops yet.' }));
  },
};
