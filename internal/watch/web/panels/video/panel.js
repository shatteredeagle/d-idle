// Game video (stage band): an MJPEG <img> shown only when the server offers a stream (/api/config frameUrl). It
// reconnects when the stream ends or is refused (viewer limit, capture restart) with a backoff of 1 s, 2 s ... capped
// at 15 s; the token, if any, rides along from the page address. Collapse is remembered, full screen asks the panel.
export function backoff(tries) { return Math.min(15000, 1000 * tries); }

export function mount(root, ctx) {
  const url = ctx.config && ctx.config.frameUrl;
  if (!url) { ctx.hide(true); return {}; }
  const frame = ctx.el('div', 'frame');
  const shot = ctx.el('img', 'shot');
  shot.alt = 'live game window';
  shot.width = 800; shot.height = 600;
  shot.hidden = true;
  const wait = ctx.el('p', 'd-empty', 'Waiting for the game picture...');
  frame.append(shot, wait);
  ctx.body.append(frame);

  const full = ctx.el('button', 'd-btn', 'Full screen');
  full.type = 'button'; full.dataset.size = 'sm'; full.dataset.key = 'full';
  full.setAttribute('aria-label', 'Full screen');
  const col = ctx.el('button', 'd-btn', 'Collapse');
  col.type = 'button'; col.dataset.size = 'sm'; col.dataset.key = 'collapse';
  ctx.actions.append(full, col);

  const saved = ctx.store('collapsed');
  function collapse(on, remember) {
    ctx.body.hidden = on;
    col.setAttribute('aria-pressed', String(on));
    col.textContent = on ? 'Expand' : 'Collapse';
    if (remember) saved.set(on ? '1' : '0');
  }
  collapse(saved.get() === '1', false);
  col.addEventListener('click', () => collapse(!ctx.body.hidden, true));
  full.addEventListener('click', () => {
    if (document.fullscreenElement) document.exitFullscreen();
    else if (root.requestFullscreen) root.requestFullscreen();
  });

  let tries = 0;
  function connect() {
    const q = [];
    if (ctx.token) q.push('t=' + encodeURIComponent(ctx.token));
    q.push('r=' + Date.now());
    shot.src = url + '?' + q.join('&');
  }
  shot.addEventListener('load', () => { tries = 0; shot.hidden = false; wait.hidden = true; ctx.state('ready'); });
  shot.addEventListener('error', () => {
    shot.hidden = true; wait.hidden = false;
    tries++;
    ctx.later(connect, backoff(tries));
  });
  connect();
  return {};
}
