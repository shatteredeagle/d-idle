// Runs web/inventory.js's durability helpers: a broken item arrives without "dur" (omitempty on 0) and must read
// 0/max and BROKEN, never "undefined". Usage: node inventory_logic.js path/to/inventory.js
const fs = require('fs'), vm = require('vm'), assert = require('assert');
let h = null;
const ctx = { window: { __invTest: (x) => { h = x; } }, document: {}, console };
ctx.window.window = ctx.window;
vm.runInNewContext(fs.readFileSync(process.argv[2], 'utf8').replace("typeof window !== 'undefined' && window.__invTest", 'window.__invTest'), ctx);
assert.ok(h, 'test hook not called');
assert.strictEqual(JSON.stringify(h.dur({ maxDur: 24 })), '{"cur":0,"max":24}', 'missing dur is 0');
assert.strictEqual(h.level(h.dur({ maxDur: 24 })), 'bad');
assert.strictEqual(JSON.stringify(h.lines({ maxDur: 12, identified: true })), '["durability 0/12 (BROKEN)"]');
assert.strictEqual(h.level(h.dur({ dur: 5, maxDur: 24 })), 'warn');
assert.strictEqual(h.level(h.dur({ dur: 20, maxDur: 24 })), 'ok');
assert.strictEqual(h.dur({ code: 'tsc' }), null, 'no durability, no line');
for (const l of h.lines({ maxDur: 3 })) assert.ok(!/undefined/.test(l), l);
console.log('ok');
