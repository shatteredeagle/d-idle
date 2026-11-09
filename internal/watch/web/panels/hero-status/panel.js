// Hero status (status band): state word with its dot, uptime, deaths and the game clock, from GET /api/uptime.
// Life, mana and the gear word wait for their data (docs/ui-design-system.md section 13, item 6).
function mmss(s) {
  const neg = s < 0;
  s = Math.abs(Math.floor(s));
  const m = Math.floor(s / 60), r = s % 60;
  return (neg ? '-' : '') + m + ':' + (r < 10 ? '0' : '') + r;
}

export function level(state) {
  return /^(death|recovery|stall)$/.test(state) ? 'bad' : state === 'idle' ? 'warn' : 'ok';
}

export function mount(root, ctx) {
  const st = ctx.el('span', 'state');
  const dot = ctx.el('span', 'dot');
  dot.setAttribute('aria-hidden', 'true');
  const word = ctx.el('span', 'word', 'connecting');
  st.append(dot, word);
  st.setAttribute('role', 'status');
  st.setAttribute('aria-live', 'polite');
  const up = ctx.el('span', 'up');
  const upVal = ctx.el('b', 'd-num', '-');
  const win = ctx.el('small', 'win');
  up.append('uptime ', upVal, ' ', win);
  const deaths = ctx.el('span', 'd-badge');
  deaths.dataset.tone = 'bad';
  deaths.hidden = true;
  const clock = ctx.el('span', 'clock d-num');
  root.append(st, up, deaths, clock);
  return {
    update(u) {
      if (!u) { word.textContent = 'not reachable'; dot.dataset.level = 'bad'; return; }
      word.textContent = u.label;
      st.dataset.state = u.state;
      dot.dataset.level = level(u.state);
      upVal.textContent = u.uptimePct === null ? '-' : u.uptimePct.toFixed(1) + '%';
      up.title = u.text || '';
      win.textContent = u.windowPct === null ? '' : '(last ' + Math.round(u.windowSec / 60) + ' min ' + u.windowPct.toFixed(1) + '%)';
      deaths.textContent = u.deaths ? u.deaths + ' death' + (u.deaths > 1 ? 's' : '') : '';
      deaths.hidden = !u.deaths;
      clock.textContent = 'time ' + mmss(u.now);
    },
  };
}
