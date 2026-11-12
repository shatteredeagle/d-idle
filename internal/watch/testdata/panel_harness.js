// Loads a panel module (web/panels/<id>/panel.js) in a vm with a stub DOM, without an ES module loader: `export` is
// stripped and the exports are returned. Used by video_panel.js and map_panel.js.
const fs = require('fs'), vm = require('vm');
function mk(tag) {
  const listeners = {}, attrs = {};
  const e = {
    tag, hidden: false, dataset: {}, children: [], textContent: '', width: 0, height: 0, clientWidth: 320,
    setAttribute(k, v) { attrs[k] = String(v); }, getAttribute(k) { return attrs[k]; }, removeAttribute(k) { delete attrs[k]; }, attrs,
    addEventListener(t, f) { (listeners[t] = listeners[t] || []).push(f); },
    fire(t, ev) { (listeners[t] || []).forEach((f) => f(Object.assign({ preventDefault() {}, clientX: 0, clientY: 0, pointerId: 1 }, ev))); },
    append(...k) { e.children.push(...k); }, appendChild(k) { e.children.push(k); return k; },
    getContext: () => new Proxy({}, { get: (t, k) => (k in t ? t[k] : () => ({ width: 1 })), set: (t, k, v) => { t[k] = v; return true; } }),
    getBoundingClientRect: () => ({ width: 320 }), setPointerCapture() {}, requestFullscreen() { e.fullscreen = true; },
  };
  return e;
}
function load(file, extra) {
  const src = fs.readFileSync(file, 'utf8').replace(/^export /gm, '').replace(/^import .*$/gm, ''); // imports come in through extra
  const names = [...src.matchAll(/^function (\w+)/gm)].map((m) => m[1]);
  const document = { fullscreenElement: null, exitFullscreen() {} };
  const ctx = Object.assign({ document, Math, Object, JSON, Date, Uint8Array, encodeURIComponent, console,
    getComputedStyle: () => ({ getPropertyValue: () => '' }) }, extra || {});
  ctx.window = ctx;
  vm.runInNewContext(src + '\n;this.__x = {' + names.join(',') + '};', ctx);
  return ctx.__x;
}
// A ctx like the shell's (web/shell.js makeCtx) with recorded timers and storage.
function shellCtx(cfg, store, timers) {
  const root = mk('section'), body = mk('div'), meta = mk('span'), actions = mk('span');
  const c = {
    id: 'x', config: cfg, token: 'tok', body, meta, actions, hidden: null, st: '',
    el: (tag, cls, text) => { const e = mk(tag); e.className = cls || ''; if (text !== undefined) e.textContent = text; return e; },
    store: (k) => ({ get: () => (k in store ? store[k] : null), set: (v) => { store[k] = v; } }),
    toast() {}, hide(on) { c.hidden = !!on; }, state(s) { c.st = s; },
    later(fn, ms) { timers.push({ fn, ms }); return timers.length; },
  };
  return { root, c };
}
module.exports = { load, shellCtx, mk };
