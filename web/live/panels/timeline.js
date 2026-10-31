import { h, clear, clock, dur } from '../lib.js';

const KINDS = [
  ['kill', 'Kills', 'x'], ['drop', 'Drops', '+'], ['death', 'Deaths', '!'], ['levelup', 'Level-ups', '^'],
  ['area', 'Areas', '>'], ['potion', 'Potions', 'o'], ['error', 'Errors', '#'], ['activity', 'Activity', '~'], ['run', 'Runs', '@'], ['info', 'Info', 'i'],
];
const GLYPH = Object.fromEntries(KINDS.map(([k, , g]) => [k, g]));
const STATE_COLORS = {
  fighting: 'var(--bad)', combat: 'var(--bad)', walking: 'var(--c1)', exploring: 'var(--c1)', town: 'var(--accent)', shopping: 'var(--accent)',
  looting: 'var(--q-unique)', resting: 'var(--stam)', healing: 'var(--stam)', loading: 'var(--muted)', idle: 'var(--muted)', dead: 'var(--life)', chicken: 'var(--warn)',
};
const MAX_ROWS = 400;

function mk(el) {
  const off = new Set();
  try { JSON.parse(localStorage.getItem('d2live-tl-off') || '[]').forEach((k) => off.add(k)); } catch (e) { /* none */ }
  const st = { off, last: 0, counts: {}, chips: h('div', { class: 'chips', role: 'group', 'aria-label': 'Event filters' }), bar: h('div'), log: h('ul', { class: 'evlog', tabindex: 0, 'aria-label': 'Event log, newest first' }) };
  const chipBtns = {};
  for (const [k, label] of KINDS) {
    const b = h('button', { type: 'button', 'aria-pressed': String(!off.has(k)), onclick: () => {
      if (off.has(k)) off.delete(k); else off.add(k);
      b.setAttribute('aria-pressed', String(!off.has(k)));
      try { localStorage.setItem('d2live-tl-off', JSON.stringify([...off])); } catch (e) { /* ignore */ }
      applyFilter(st);
    } }, label + ' ');
    b.append(h('span', { class: 'muted', text: '0' }));
    chipBtns[k] = b; st.chips.append(b);
  }
  st.chipBtns = chipBtns;
  el.append(st.chips, h('h3', { text: 'Activity' }), st.bar, h('h3', { text: 'Events' }), st.log);
  return st;
}
function applyFilter(st) { for (const li of st.log.children) li.hidden = st.off.has(li.dataset.k); }
function row(st, e) {
  const li = h('li', { 'data-k': e.kind },
    h('time', { text: clock(new Date(e.time).getTime()) }),
    h('span', { class: `kind k-${e.kind}`, 'data-g': GLYPH[e.kind] || '.', text: e.kind }),
    h('span', { text: e.text + (e.extra ? ` (${e.extra})` : '') }));
  li.hidden = st.off.has(e.kind);
  return li;
}
function count(st, kind, n) {
  st.counts[kind] = (st.counts[kind] || 0) + n;
  const b = st.chipBtns[kind]; if (b) b.lastChild.textContent = st.counts[kind];
}

export default {
  id: 'timeline',
  span: 5,
  render(el, d) {
    const st = el._t || (el._t = (clear(el), mk(el)));
    clear(st.log); st.counts = {};
    for (const k of Object.keys(st.chipBtns)) st.chipBtns[k].lastChild.textContent = '0';
    for (const e of d.events) { count(st, e.kind, 1); st.last = Math.max(st.last, e.seq); }
    for (const e of d.events.slice().reverse()) st.log.append(row(st, e));
    // activity bar
    clear(st.bar);
    const segs = d.segments;
    if (!segs.length) { st.bar.append(h('p', { class: 'empty-note', text: 'No activity recorded yet.' })); return; }
    const t0 = new Date(segs[0].start).getTime(), now = d.now;
    const total = Math.max(1, now - t0);
    const bar = h('div', { class: 'tlbar', role: 'img', 'aria-label': 'Activity timeline: ' + segs.map((s) => s.state).slice(-6).join(', then ') });
    const seen = new Set();
    segs.forEach((s) => {
      const a = new Date(s.start).getTime(), b = s.end ? new Date(s.end).getTime() : now;
      const i = h('i', { title: `${s.state}${s.detail ? ' - ' + s.detail : ''} (${dur((b - a) / 1000)})` });
      i.style.flexGrow = Math.max(1, b - a); i.style.flexBasis = '0';
      i.style.setProperty('--sc', STATE_COLORS[s.state] || 'var(--c3)');
      bar.append(i); seen.add(s.state);
    });
    const legend = h('div', { class: 'legend' }, ...[...seen].map((s) => h('span', {}, h('i', { style: { width: '10px', height: '10px', borderRadius: '2px', display: 'inline-block', background: STATE_COLORS[s] || 'var(--c3)' } }), s)));
    st.bar.append(bar, h('div', { class: 'tlaxis' }, h('span', { text: clock(t0, false) }), h('span', { text: dur(total / 1000) + ' shown' }), h('span', { text: 'now' })), legend);
  },
  append(el, e) {
    const st = el._t; if (!st || e.seq <= st.last) return;
    st.last = e.seq; count(st, e.kind, 1);
    st.log.prepend(row(st, e));
    while (st.log.children.length > MAX_ROWS) st.log.lastChild.remove();
  },
};
