// Runs web/panels/video/panel.js against a stub DOM: shown only with a stream, reconnect backoff, collapse memory,
// full screen on the panel. Usage: node video_panel.js path/to/panel.js
const assert = require('assert'), { load, shellCtx } = require('./panel_harness.js');
const path = require('path');
const M = load(path.join(path.dirname(process.argv[2]), '../../mjpeg.js'));
const P = load(process.argv[2], M);
// no stream: the panel hides itself
{ const t = []; const { root, c } = shellCtx({}, {}, t); P.mount(root, c); assert.strictEqual(c.hidden, true); }
const store = {}, timers = [];
const { root, c } = shellCtx({ frameUrl: '/stream' }, store, timers);
P.mount(root, c);
const shot = c.body.children[0].children[0], [full, col] = c.actions.children;
assert.notStrictEqual(c.hidden, true, 'shown when the server offers a stream');
assert.ok(/^\/stream\?t=tok&r=\d+$/.test(shot.src), 'src ' + shot.src);
const waits = [];
for (let i = 0; i < 20; i++) { timers.length = 0; shot.fire('error'); waits.push(timers[0].ms); timers[0].fn(); }
assert.deepStrictEqual(waits.slice(0, 5), [1000, 2000, 4000, 8000, 10000]);
assert.strictEqual(waits[19], 10000);
timers.length = 0; shot.fire('load'); assert.strictEqual(c.st, 'ready'); timers.length = 0; shot.fire('error');
assert.strictEqual(timers[0].ms, 1000, 'a good frame resets the backoff');
assert.strictEqual(c.st, 'reconnecting');
col.fire('click');
assert.strictEqual(c.body.hidden, true); assert.strictEqual(store.collapsed, '1');
{ const t = []; const s2 = shellCtx({ frameUrl: '/stream' }, { collapsed: '1' }, t); P.mount(s2.root, s2.c); assert.strictEqual(s2.c.body.hidden, true, 'collapse restored'); }
col.fire('click');
assert.strictEqual(c.body.hidden, false); assert.strictEqual(store.collapsed, '0');
full.fire('click'); assert.ok(root.fullscreen, 'full screen asks the panel');

// the connection logic on a fake clock: stall -> reconnect with backoff, hidden -> closed, visible -> reopened
{
  let now = 0, id = 0, opens = 0, closes = 0, onFrame = null, onEnd = null;
  const q = new Map(), states = [];
  const fire = (ms) => { now += ms; for (const [k, t] of [...q]) if (t.at <= now) { q.delete(k); t.fn(); } };
  const ctl = M.videoController({
    open(f, e) { opens++; onFrame = f; onEnd = e; return () => { closes++; }; },
    later(fn, ms) { q.set(++id, { fn, at: now + ms }); return id; },
    cancel(k) { q.delete(k); },
    now: () => now, state: (s) => states.push(s),
  });
  ctl.start();
  assert.strictEqual(opens, 1); assert.strictEqual(ctl.state, 'connecting');
  for (let i = 0; i < 10; i++) { fire(40); onFrame(); }
  assert.strictEqual(ctl.state, 'live');
  assert.ok(ctl.fps() > 20 && ctl.fps() < 30, 'fps ' + ctl.fps());
  // frames stop: 3 s later the stream is torn down and retried after 1 s, then 2 s, 4 s
  fire(2999); assert.strictEqual(closes, 0, 'no stall before 3 s');
  fire(1); assert.strictEqual(closes, 1); assert.strictEqual(ctl.state, 'reconnecting');
  fire(999); assert.strictEqual(opens, 1); fire(1); assert.strictEqual(opens, 2, 'retry after 1 s');
  fire(3000); assert.strictEqual(closes, 2); fire(2000); assert.strictEqual(opens, 3, 'retry after 2 s');
  fire(3000); fire(4000); assert.strictEqual(opens, 4, 'retry after 4 s');
  // a stream that ends by itself (server restart) counts the same
  onEnd(); fire(8000); assert.strictEqual(opens, 5);
  onEnd(); assert.strictEqual(ctl.state, 'down', 'five failures: stream down');
  fire(10000); assert.strictEqual(opens, 6, 'down still retries at 10 s');
  // Retry: at once, fresh backoff
  ctl.reconnect(); assert.strictEqual(opens, 7); onFrame(); assert.strictEqual(ctl.state, 'live');
  // the tab is hidden: no stall alarm, closed after 10 s; visible again: reopened at once
  ctl.visibility(true); fire(9999); assert.strictEqual(opens, 7); assert.strictEqual(ctl.state, 'live');
  const c0 = closes; fire(1); assert.strictEqual(closes, c0 + 1); assert.strictEqual(ctl.state, 'paused');
  fire(60000); assert.strictEqual(opens, 7, 'nothing reopens a hidden tab');
  ctl.visibility(false); assert.strictEqual(opens, 8, 'visible: reopened at once');
  onFrame();
  // a short hide does not close a live stream
  ctl.visibility(true); fire(5000); ctl.visibility(false); assert.strictEqual(opens, 8); onFrame();
  // wake (focus/online) leaves a live stream alone and revives a stale one
  ctl.wake(); assert.strictEqual(opens, 8);
  fire(2000); now += 2000; ctl.wake(); assert.ok(opens >= 9, 'stale stream revived on wake');
}
// the part parser: Content-Length framed parts in arbitrary chunks
{
  const parts = [];
  const p = M.partParser((b, t) => parts.push([Buffer.from(b).toString(), t]));
  const msg = '--f\r\nContent-Type: image/jpeg\r\nContent-Length: 5\r\nX-Frame-Ticks: 3\r\n\r\nAAAAA\r\n--f\r\nContent-Type: image/png\r\nContent-Length: 3\r\n\r\nBBB\r\n';
  const bytes = Buffer.from(msg);
  for (let i = 0; i < bytes.length; i += 7) p.push(new Uint8Array(bytes.subarray(i, i + 7)));
  assert.deepStrictEqual(parts, [['AAAAA', 'image/jpeg'], ['BBB', 'image/png']]);
  assert.throws(() => M.partParser(() => {}).push(new Uint8Array(Buffer.from('--f\r\nContent-Type: image/jpeg\r\n\r\nxx'))), /Content-Length/);
}
console.log('ok');
