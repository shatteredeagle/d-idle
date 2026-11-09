// Runs web/panels/video/panel.js against a stub DOM: shown only with a stream, reconnect backoff, collapse memory,
// full screen on the panel. Usage: node video_panel.js path/to/panel.js
const assert = require('assert'), { load, shellCtx } = require('./panel_harness.js');
const P = load(process.argv[2]);
// no stream: the panel hides itself
{ const t = []; const { root, c } = shellCtx({}, {}, t); P.mount(root, c); assert.strictEqual(c.hidden, true); }
const store = {}, timers = [];
const { root, c } = shellCtx({ frameUrl: '/stream' }, store, timers);
P.mount(root, c);
const shot = c.body.children[0].children[0], [full, col] = c.actions.children;
assert.notStrictEqual(c.hidden, true, 'shown when the server offers a stream');
assert.ok(/^\/stream\?t=tok&r=\d+$/.test(shot.src), 'src ' + shot.src);
const waits = [];
for (let i = 0; i < 20; i++) { timers.length = 0; shot.fire('error'); waits.push(timers[0].ms); }
assert.deepStrictEqual(waits.slice(0, 3), [1000, 2000, 3000]);
assert.strictEqual(waits[19], 15000);
shot.fire('load'); timers.length = 0; shot.fire('error');
assert.strictEqual(timers[0].ms, 1000, 'a good frame resets the backoff');
assert.strictEqual(c.st, 'ready');
col.fire('click');
assert.strictEqual(c.body.hidden, true); assert.strictEqual(store.collapsed, '1');
{ const t = []; const s2 = shellCtx({ frameUrl: '/stream' }, { collapsed: '1' }, t); P.mount(s2.root, s2.c); assert.strictEqual(s2.c.body.hidden, true, 'collapse restored'); }
col.fire('click');
assert.strictEqual(c.body.hidden, false); assert.strictEqual(store.collapsed, '0');
full.fire('click'); assert.ok(root.fullscreen, 'full screen asks the panel');
console.log('ok');
