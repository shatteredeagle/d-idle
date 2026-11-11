// Game video (stage band), shown only when the server offers a stream (/api/config frameUrl). The stream is read with
// fetch and split into its JPEG parts here, so the page knows when each frame arrives: no frame for 3 s while the tab
// is visible tears the stream down and reconnects (1 s, 2 s, 4 s ... 10 s), showing "reconnecting" instead of a frozen
// picture and "stream down" with a Retry button after five failures in a row. A tab hidden for 10 s closes the stream
// (the browser throttles it anyway); visible, focus, pageshow and online reopen it at once. window.d2video.reconnect()
// lets the desktop app reload it after a sleep. Browsers without streaming fetch keep the plain MJPEG <img>.
// The token, if any, rides along from the page address. Collapse is remembered, full screen asks the panel.
// The picture always fits its box: it keeps 4:3, scales to the width and the height the cell gives and sits centred.
export function backoff(tries) { return Math.min(10000, 1000 * Math.pow(2, Math.max(0, tries - 1))); }

// partParser splits a multipart MJPEG byte stream (parts carrying Content-Length, as d2dview and the watch relay
// send) and calls onPart(bytes, type) for each complete part. push takes Uint8Array chunks of any size.
export function partParser(onPart) {
  let buf = new Uint8Array(0);
  function headerEnd() {
    for (let i = 0; i + 3 < buf.length; i++) {
      if (buf[i] === 13 && buf[i + 1] === 10 && buf[i + 2] === 13 && buf[i + 3] === 10) return i;
    }
    return -1;
  }
  return {
    push(chunk) {
      const next = new Uint8Array(buf.length + chunk.length);
      next.set(buf); next.set(chunk, buf.length);
      buf = next;
      for (;;) {
        const h = headerEnd();
        if (h < 0) {
          if (buf.length > 65536) throw new Error('stream: no part header');
          return;
        }
        let text = '';
        for (let i = 0; i < h; i++) text += String.fromCharCode(buf[i]);
        const len = /content-length:\s*(\d+)/i.exec(text);
        if (!len) throw new Error('stream: part without Content-Length');
        const n = Number(len[1]), start = h + 4;
        if (buf.length < start + n) return;
        const type = (/content-type:\s*([^\r\n;]+)/i.exec(text) || [])[1] || 'image/jpeg';
        onPart(buf.slice(start, start + n), type.trim());
        buf = buf.slice(start + n);
      }
    },
  };
}

// videoController is the connection logic without the DOM: o.open(onFrame, onEnd) starts a stream and returns its
// close function; o.later(fn, ms) / o.cancel(id) are the timers; o.state(name) reports connecting, live,
// reconnecting, down or paused; o.now() is the clock. stall 0 turns the watchdog off (the <img> fallback cannot see
// frames).
export function videoController(o) {
  const stall = o.stall === undefined ? 3000 : o.stall, hideAfter = o.hideAfter || 10000, downAfter = o.downAfter || 5;
  let close = null, gen = 0, tries = 0, stallT = 0, retryT = 0, hideT = 0, hidden = false, last = 0, state = '';
  const stamps = [];
  const set = (s) => { if (s !== state) { state = s; o.state(s); } };
  const cancel = (t) => { if (t) o.cancel(t); return 0; };
  function arm() {
    stallT = cancel(stallT);
    if (stall && !hidden && close) stallT = o.later(() => { stallT = 0; fail(); }, stall);
  }
  function drop() {
    gen++;
    stallT = cancel(stallT);
    if (close) { const c = close; close = null; c(); }
  }
  function fail() {
    drop();
    if (hidden) { set('paused'); return; }
    tries++;
    set(tries >= downAfter ? 'down' : 'reconnecting');
    retryT = cancel(retryT);
    retryT = o.later(() => { retryT = 0; start(); }, backoff(tries));
  }
  function frame(g) {
    if (g !== gen) return;
    tries = 0; last = o.now();
    stamps.push(last);
    while (stamps.length && last - stamps[0] > 2000) stamps.shift();
    set('live');
    arm();
  }
  function start() {
    if (close || hidden) return;
    retryT = cancel(retryT);
    if (state !== 'down') set(tries ? 'reconnecting' : 'connecting');
    const g = ++gen;
    close = o.open(() => frame(g), () => { if (g === gen) fail(); });
    arm();
  }
  function reconnect() {
    drop(); tries = 0; retryT = cancel(retryT); hidden = false; hideT = cancel(hideT);
    set('connecting'); start();
  }
  return {
    start, reconnect,
    get state() { return state; },
    // fps over the last 2 s of frames
    fps() { const n = stamps.length; return n < 2 ? 0 : (1000 * (n - 1)) / Math.max(1, stamps[n - 1] - stamps[0]); },
    // focus, pageshow, online: a stream that still delivers frames is left alone
    wake() { if (hidden) return; if (state === 'live' && (!stall || o.now() - last < stall)) return; reconnect(); },
    visibility(isHidden) {
      if (isHidden) {
        hidden = true;
        stallT = cancel(stallT); // a hidden tab is throttled: its silence is no stall
        if (!hideT) hideT = o.later(() => { hideT = 0; drop(); retryT = cancel(retryT); set('paused'); }, hideAfter);
        return;
      }
      hidden = false;
      hideT = cancel(hideT);
      if (!close) { tries = 0; start(); } else arm();
    },
    stop() { drop(); retryT = cancel(retryT); hideT = cancel(hideT); },
  };
}

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
  const streaming = typeof fetch === 'function' && typeof ReadableStream === 'function' && typeof AbortController === 'function' &&
    typeof URL === 'function' && typeof URL.createObjectURL === 'function' && typeof Blob === 'function';
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
    const ac = new AbortController();
    (async () => {
      try {
        const r = await fetch(src(), { signal: ac.signal, cache: 'no-store', credentials: 'same-origin' });
        if (!r.ok || !r.body) throw new Error('stream answered ' + r.status);
        const p = partParser((bytes, type) => { show(bytes, type); onFrame(); });
        const rd = r.body.getReader();
        for (;;) {
          const { done, value } = await rd.read();
          if (done) break;
          p.push(value);
        }
        onEnd();
      } catch (e) {
        if (!ac.signal.aborted) onEnd(e);
      }
    })();
    return () => ac.abort();
  }
  // <img> fallback: the browser plays the MJPEG itself; 'error' (refused, ended) is the only end signal
  let imgEnd = null, imgFrame = null;
  function openImg(onFrame, onEnd) {
    imgFrame = onFrame; imgEnd = onEnd;
    shot.src = src();
    return () => { imgEnd = imgFrame = null; };
  }
  shot.addEventListener('load', () => {
    if (streaming) decoded(); else if (imgFrame) imgFrame();
    shot.hidden = false; wait.hidden = true;
  });
  shot.addEventListener('error', () => {
    if (streaming) { decoded(); return; }
    shot.hidden = true; wait.hidden = false;
    if (imgEnd) imgEnd();
  });

  const timers = new Map();
  let tick = 0;
  const ctl = videoController({
    stall: streaming ? 3000 : 0,
    open: streaming ? openFetch : openImg,
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
  window.d2video = { reconnect: () => ctl.reconnect(), state: () => ctl.state, fps: () => ctl.fps() };
  ctl.start();
  return {};
}
