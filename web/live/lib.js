// Shared helpers for panel modules. No dependencies.

export function h(tag, props, ...kids) {
  const el = tag.startsWith('svg:')
    ? document.createElementNS('http://www.w3.org/2000/svg', tag.slice(4))
    : document.createElement(tag);
  for (const [k, v] of Object.entries(props || {})) {
    if (v == null || v === false) continue;
    if (k === 'class') el.setAttribute('class', v);
    else if (k === 'text') el.textContent = v;
    else if (k === 'style' && typeof v === 'object') setStyle(el, v);
    else if (k.startsWith('on')) el.addEventListener(k.slice(2), v);
    else el.setAttribute(k, v === true ? '' : v);
  }
  add(el, kids);
  return el;
}
export function setStyle(el, o) {
  for (const [k, v] of Object.entries(o)) { if (k.startsWith('--')) el.style.setProperty(k, v); else el.style[k] = v; }
}
function add(el, kids) {
  for (const k of kids.flat(Infinity)) {
    if (k == null || k === false) continue;
    el.append(k instanceof Node ? k : document.createTextNode(String(k)));
  }
}
export function clear(el) { while (el.firstChild) el.removeChild(el.firstChild); return el; }

export const num = (n) => (n == null ? '-' : Math.round(n).toLocaleString('en-US'));
export function compact(n) {
  const a = Math.abs(n);
  if (a >= 1e6) return (n / 1e6).toFixed(a >= 1e7 ? 0 : 1) + 'M';
  if (a >= 1e4) return Math.round(n / 1e3) + 'k';
  if (a >= 1e3) return (n / 1e3).toFixed(1) + 'k';
  return String(Math.round(n));
}
export function dur(s) {
  if (s == null || s < 0 || !isFinite(s)) return '-';
  s = Math.round(s);
  const d = Math.floor(s / 86400), hh = Math.floor((s % 86400) / 3600), m = Math.floor((s % 3600) / 60), sec = s % 60;
  if (d) return `${d}d ${hh}h`;
  if (hh) return `${hh}h ${String(m).padStart(2, '0')}m`;
  if (m) return `${m}m ${String(sec).padStart(2, '0')}s`;
  return `${sec}s`;
}
export function clock(ms, withSec = true) {
  const d = new Date(ms);
  const p = (x) => String(x).padStart(2, '0');
  return withSec ? `${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}` : `${p(d.getHours())}:${p(d.getMinutes())}`;
}

export const QUALITY = {
  low: 'Low quality', normal: 'Normal', superior: 'Superior', magic: 'Magic', rare: 'Rare',
  unique: 'Unique', set: 'Set', crafted: 'Crafted', runeword: 'Runeword',
};
export const qclass = (q) => 'q-' + (QUALITY[q] ? q : 'normal');

export function bar(kind, value, max, label) {
  const pct = max > 0 ? Math.max(0, Math.min(100, (value / max) * 100)) : 0;
  const fill = h('i');
  fill.style.width = pct + '%';
  return h('div', { class: 'bar ' + (kind || ''), role: 'progressbar', 'aria-valuemin': 0, 'aria-valuemax': max, 'aria-valuenow': value, 'aria-label': label || kind },
    fill, h('span', { text: label != null ? label : `${num(value)} / ${num(max)}` }));
}

// --- tooltip ---------------------------------------------------------------
let tipEl;
export function tip(target, build) {
  const show = () => {
    tipEl = tipEl || document.getElementById('tip');
    clear(tipEl);
    add(tipEl, [build()]);
    tipEl.hidden = false;
    const r = target.getBoundingClientRect();
    const tw = tipEl.offsetWidth, th = tipEl.offsetHeight;
    let x = r.right + 8, y = r.top;
    if (x + tw > innerWidth - 8) x = Math.max(8, r.left - tw - 8);
    if (x < 8) x = Math.min(8, innerWidth - tw - 8);
    if (y + th > innerHeight - 8) y = Math.max(8, innerHeight - th - 8);
    tipEl.style.left = x + 'px';
    tipEl.style.top = y + 'px';
  };
  const hide = () => { if (tipEl) tipEl.hidden = true; };
  target.addEventListener('mouseenter', show);
  target.addEventListener('focus', show);
  target.addEventListener('mouseleave', hide);
  target.addEventListener('blur', hide);
  target.addEventListener('keydown', (e) => { if (e.key === 'Escape') hide(); });
}

// --- SVG line chart ----------------------------------------------------------
// opts: {points:[{x:ms,y:number}], fmtY, fmtX, color, label, height}
export function lineChart(host, opts) {
  clear(host);
  const pts = opts.points;
  const W = Math.max(200, host.clientWidth || 300), H = opts.height || 150;
  const m = { l: 44, r: 8, t: 8, b: 20 };
  const svg = h('svg:svg', { width: W, height: H, viewBox: `0 0 ${W} ${H}`, role: 'img' });
  host.append(svg);
  if (pts.length < 2) {
    svg.setAttribute('aria-label', opts.label + ': not enough data yet');
    const t = h('svg:text', { x: W / 2, y: H / 2, 'text-anchor': 'middle' });
    t.textContent = 'collecting data...';
    svg.append(t);
    return;
  }
  const x0 = pts[0].x, x1 = pts[pts.length - 1].x || x0 + 1;
  let ymax = Math.max(...pts.map((p) => p.y), 1);
  const step = niceStep(ymax / 4);
  ymax = Math.ceil(ymax / step) * step;
  const sx = (x) => m.l + ((x - x0) / Math.max(1, x1 - x0)) * (W - m.l - m.r);
  const sy = (y) => H - m.b - (y / ymax) * (H - m.t - m.b);
  const fy = opts.fmtY || compact;
  for (let v = 0; v <= ymax + 1e-9; v += step) {
    svg.append(h('svg:line', { class: 'grid-l', x1: m.l, x2: W - m.r, y1: sy(v), y2: sy(v) }));
    const t = h('svg:text', { x: m.l - 6, y: sy(v) + 4, 'text-anchor': 'end' });
    t.textContent = fy(v);
    svg.append(t);
  }
  const nx = Math.max(2, Math.min(5, Math.floor(W / 90)));
  for (let i = 0; i < nx; i++) {
    const x = x0 + ((x1 - x0) * i) / (nx - 1);
    const t = h('svg:text', { x: sx(x), y: H - 4, 'text-anchor': i === 0 ? 'start' : i === nx - 1 ? 'end' : 'middle' });
    t.textContent = clock(x, false);
    svg.append(t);
  }
  const d = pts.map((p, i) => `${i ? 'L' : 'M'}${sx(p.x).toFixed(1)},${sy(p.y).toFixed(1)}`).join('');
  svg.append(h('svg:path', { d: d + `L${sx(x1)},${sy(0)}L${sx(x0)},${sy(0)}Z`, fill: opts.color, 'fill-opacity': 0.12 }));
  svg.append(h('svg:path', { d, fill: 'none', stroke: opts.color, 'stroke-width': 2, 'stroke-linejoin': 'round' }));
  const last = pts[pts.length - 1];
  svg.setAttribute('aria-label', `${opts.label}: latest ${fy(last.y)}, peak ${fy(Math.max(...pts.map((p) => p.y)))}, over ${dur((x1 - x0) / 1000)}`);
  // hover layer
  const cross = h('svg:line', { y1: m.t, y2: H - m.b, stroke: 'currentColor', 'stroke-opacity': 0.4, visibility: 'hidden' });
  const dot = h('svg:circle', { r: 4, fill: opts.color, stroke: 'var(--surface)', 'stroke-width': 2, visibility: 'hidden' });
  const tipbox = h('div', { class: 'tip', hidden: true, style: { position: 'absolute' } });
  host.style.position = 'relative';
  host.append(tipbox);
  svg.append(cross, dot);
  const move = (ev) => {
    const r = svg.getBoundingClientRect();
    const px = ev.clientX - r.left;
    let best = 0, bd = 1e9;
    pts.forEach((p, i) => { const dd = Math.abs(sx(p.x) - px); if (dd < bd) { bd = dd; best = i; } });
    const p = pts[best];
    cross.setAttribute('x1', sx(p.x)); cross.setAttribute('x2', sx(p.x)); cross.setAttribute('visibility', 'visible');
    dot.setAttribute('cx', sx(p.x)); dot.setAttribute('cy', sy(p.y)); dot.setAttribute('visibility', 'visible');
    tipbox.textContent = `${clock(p.x, false)}  ${fy(p.y)}${opts.unit || ''}`;
    tipbox.hidden = false;
    tipbox.style.left = Math.min(W - 120, Math.max(0, sx(p.x) - 40)) + 'px';
    tipbox.style.top = '0px';
  };
  const leave = () => { cross.setAttribute('visibility', 'hidden'); dot.setAttribute('visibility', 'hidden'); tipbox.hidden = true; };
  svg.addEventListener('pointermove', move);
  svg.addEventListener('pointerleave', leave);
}
function niceStep(raw) {
  const p = Math.pow(10, Math.floor(Math.log10(raw || 1)));
  const f = raw / p;
  return (f <= 1 ? 1 : f <= 2 ? 2 : f <= 5 ? 5 : 10) * p;
}
