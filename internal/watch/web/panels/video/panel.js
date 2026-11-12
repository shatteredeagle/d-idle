import { videoController, openStream, streamingSupported, registerVideo } from '../../mjpeg.js';

// Game video (stage band), shown only when the server offers a stream (/api/config frameUrl). The stream is read by
// web/mjpeg.js (fetch, split into JPEG parts), so the page knows when each frame arrives: no frame for 3 s while the tab
// is visible tears the stream down and reconnects (1 s, 2 s, 4 s ... 10 s), showing "reconnecting" instead of a frozen
// picture and "stream down" with a Retry button after five failures in a row. A tab hidden for 10 s closes the stream
// (the browser throttles it anyway); visible, focus, pageshow and online reopen it at once. window.d2video.reconnect()
// lets the desktop app reload it after a sleep. Browsers without streaming fetch keep the plain MJPEG <img>.
// The token, if any, rides along from the page address. Collapse is remembered, full screen asks the panel.
// The picture always fits its box: it keeps 4:3, scales to the width and the height the cell gives and sits centred.
export function mount(root, ctx) {
  const url = ctx.config && ctx.config.frameUrl;
  if (!url) { ctx.hide(true); return {}; }
  root.dataset.aspect = '4/3'; // the layout editor keeps the cell in the picture's shape (panel version 2)
  const frame = ctx.el('div', 'frame');
  const shot = ctx.el('img', 'shot');
  shot.alt = 'live game window';
  shot.width = 800; shot.height = 600;
  shot.hidden = true;
  const wait = ctx.el('p', 'd-empty', 'Waiting for the game picture...');
  const badge = ctx.el('span', 'badge', '');
  badge.hidden = true;
  badge.setAttribute('role', 'status');
  const retry = ctx.el('button', 'd-btn', 'Retry');
  retry.type = 'button'; retry.dataset.size = 'sm'; retry.hidden = true;
  frame.append(shot, wait, badge, retry);
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

  function src() {
    const q = [];
    if (ctx.token) q.push('t=' + encodeURIComponent(ctx.token));
    q.push('r=' + Date.now());
    return url + '?' + q.join('&');
  }

  // fetch path: each part becomes an object URL on the <img>; while one decodes only the newest waits (latest wins)
  let streaming = streamingSupported(), blobErrs = 0;
  let shown = '', busy = false, pending = null;
  function show(bytes, type) {
    if (busy) { pending = [bytes, type]; return; }
    busy = true;
    const u = URL.createObjectURL(new Blob([bytes], { type }));
    if (shown) URL.revokeObjectURL(shown);
    shown = u;
    shot.src = u;
  }
  function decoded() {
    busy = false;
    if (pending) { const p = pending; pending = null; show(p[0], p[1]); }
  }
  function openFetch(onFrame, onEnd) {
    return openStream(src(), (bytes, type) => { show(bytes, type); onFrame(); }, onEnd);
  }
  // <img> fallback: the browser plays the MJPEG itself; 'error' (refused, ended) is the only end signal
  let imgEnd = null, imgFrame = null;
  function openImg(onFrame, onEnd) {
    imgFrame = onFrame; imgEnd = onEnd;
    shot.src = src();
    return () => { imgEnd = imgFrame = null; };
  }
  shot.addEventListener('load', () => {
    if (streaming) { blobErrs = 0; decoded(); } else if (imgFrame) imgFrame();
    shot.hidden = false; wait.hidden = true;
  });
  shot.addEventListener('error', () => {
    if (streaming) {
      decoded();
      // frames that never decode (a page policy without blob: images): fall back to the browser's own MJPEG <img>
      if (++blobErrs >= 3) { streaming = false; ctl.reconnect(); }
      return;
    }
    shot.hidden = true; wait.hidden = false;
    if (imgEnd) imgEnd();
  });

  const timers = new Map();
  let tick = 0;
  const ctl = videoController({
    stall: () => (streaming ? 3000 : 0),
    open: (onFrame, onEnd) => (streaming ? openFetch(onFrame, onEnd) : openImg(onFrame, onEnd)),
    later(fn, ms) { const id = ++tick; timers.set(id, true); ctx.later(() => { if (timers.delete(id)) fn(); }, ms); return id; },
    cancel(id) { timers.delete(id); },
    now: () => Date.now(),
    state(st) {
      ctx.state(st === 'live' ? 'ready' : st);
      badge.hidden = st === 'live' || st === 'connecting';
      badge.textContent = st === 'down' ? 'stream down' : st === 'paused' ? 'paused (tab hidden)' : st === 'reconnecting' ? 'reconnecting...' : '';
      retry.hidden = st !== 'down';
    },
  });
  retry.addEventListener('click', () => ctl.reconnect());
  // a tiny rate line in the card head, only with ?debug in the page address
  if (typeof location !== 'undefined' && /[?&]debug\b/.test(location.search || '')) {
    const rate = () => { ctx.meta.textContent = 'stream: ' + Math.round(ctl.fps()) + ' fps'; ctx.later(rate, 1000); };
    rate();
  }
  if (typeof document.addEventListener === 'function') {
    document.addEventListener('visibilitychange', () => ctl.visibility(document.visibilityState === 'hidden'));
  }
  if (typeof window.addEventListener === 'function') {
    window.addEventListener('focus', () => ctl.wake());
    window.addEventListener('online', () => ctl.wake());
    window.addEventListener('pageshow', (e) => { if (e && e.persisted) ctl.reconnect(); else ctl.wake(); });
  }
  registerVideo(ctl);
  ctl.start();
  return {};
}
