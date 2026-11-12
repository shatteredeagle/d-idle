// MJPEG stream reading for the game video panel (web/panels/video). A long-lived stream is not a poll, so it is not
// the shell's scheduler's job: this module reads the multipart stream with fetch and splits it into JPEG parts, so the
// page knows when each frame arrives, and owns the reconnect logic (videoController) and the one page hook
// window.d2video.reconnect() that the desktop app calls after a sleep (docs/viewer-fps.md section 8).
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
// reconnecting, down or paused; o.now() is the clock. o.stall (ms, or a function giving them) 0 turns the watchdog off (the <img> fallback cannot see
// frames).
export function videoController(o) {
  const stallMs = () => (typeof o.stall === 'function' ? o.stall() : o.stall === undefined ? 3000 : o.stall);
  const hideAfter = o.hideAfter || 10000, downAfter = o.downAfter || 5;
  let close = null, gen = 0, tries = 0, stallT = 0, retryT = 0, hideT = 0, hidden = false, last = 0, state = '';
  const stamps = [];
  const set = (s) => { if (s !== state) { state = s; o.state(s); } };
  const cancel = (t) => { if (t) o.cancel(t); return 0; };
  function arm() {
    stallT = cancel(stallT);
    const ms = stallMs();
    if (ms && !hidden && close) stallT = o.later(() => { stallT = 0; fail(); }, ms);
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
    wake() { if (hidden) return; if (state === 'live' && (!stallMs() || o.now() - last < stallMs())) return; reconnect(); },
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

// streamingSupported: fetch with a readable body (every current browser); older ones keep the plain <img>.
export function streamingSupported() {
  return typeof fetch === 'function' && typeof ReadableStream === 'function' && typeof AbortController === 'function' &&
    typeof URL === 'function' && typeof URL.createObjectURL === 'function' && typeof Blob === 'function';
}

// openStream reads url as a multipart stream, calls onPart(bytes, type) per part and onEnd(err) once when it ends
// (not after the returned close function was called).
export function openStream(url, onPart, onEnd) {
  const ac = new AbortController();
  (async () => {
    try {
      const r = await fetch(url, { signal: ac.signal, cache: 'no-store', credentials: 'same-origin' });
      if (!r.ok || !r.body) throw new Error('stream answered ' + r.status);
      const p = partParser(onPart);
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

// registerVideo publishes the page hook: window.d2video.reconnect(), state(), fps().
export function registerVideo(ctl) {
  if (typeof window === 'undefined') return;
  window.d2video = { reconnect: () => ctl.reconnect(), state: () => ctl.state, fps: () => ctl.fps() };
}
