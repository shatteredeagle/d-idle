// Node test of web/vault-logic.js (run by TestVaultLogicJS): query round trips, the client mirror of the server filter and sort,
// the multi-select with shift ranges, the capacity words. Usage: node vault_logic.mjs <path to vault-logic.js>
import assert from 'node:assert/strict';
import { pathToFileURL } from 'node:url';

const L = await import(pathToFileURL(process.argv[2]).href);
const T = { overBanner: 'Over by {over} ({n} held)', fullBanner: 'Full' };
const kinds = [{ id: 'unique', quality: [7] }, { id: 'rune', codePrefix: ['r0', 'r1', 'r2', 'r3'] }];
let n = 0;
const ok = (name, fn) => { fn(); n++; };

const row = (o) => Object.assign({ id: 'v-' + Math.random(), name: 'Cap', code: 'cap', q: 4, quality: 'magic', ilvl: 10, sockets: 0, ethereal: false, identified: true, pin: false, state: 'recorded', value: 5, found: '2026-11-16T10:00:00Z' }, o);

ok('default filters make an empty query', () => assert.equal(L.toQuery(L.defaultFilters()), ''));
ok('query round trip', () => {
  const f = Object.assign(L.defaultFilters(), { q: ' war ', quality: [7, 4], kind: 'unique', type: 'swor', ilvlMin: '20', ilvlMax: '30', sockets: '3', eth: true, unid: true, pinned: true, state: 'kept', sort: 'ilvl' });
  const q = L.toQuery(f, { offset: 60, limit: 60, facets: true });
  const p = new URLSearchParams(q);
  assert.equal(p.get('q'), 'war'); assert.equal(p.get('quality'), '4,7'); assert.equal(p.get('identified'), '0'); assert.equal(p.get('eth'), '1');
  assert.equal(p.get('offset'), '60'); assert.equal(p.get('facets'), '1'); assert.equal(p.get('sort'), 'ilvl'); assert.equal(p.get('sockets'), '3');
  const g = L.fromQuery(q);
  assert.deepEqual(g.quality.sort(), [4, 7]); assert.equal(g.kind, 'unique'); assert.equal(g.ilvlMin, '20'); assert.equal(g.unid, true); assert.equal(g.q, 'war'); assert.equal(g.sort, 'ilvl');
});
ok('sockets 0 is a filter, empty is not', () => {
  assert.equal(L.toQuery(Object.assign(L.defaultFilters(), { sockets: '0' })), '?sockets=0');
  assert.equal(L.activeCount(Object.assign(L.defaultFilters(), { sockets: '0' })), 1);
  assert.equal(L.activeCount(L.defaultFilters()), 0);
});
ok('a bad address is ignored, not trusted', () => {
  const f = L.fromQuery('?quality=99,rare&sort=drop&ilvlMin=-3&kind=x');
  assert.deepEqual(f.quality, []); assert.equal(f.sort, 'new'); assert.equal(f.ilvlMin, '');
});
ok('active count: sort does not count, an ilvl range is one', () => {
  const f = Object.assign(L.defaultFilters(), { sort: 'name', ilvlMin: '5', ilvlMax: '9', quality: [4, 6], eth: true });
  assert.equal(L.activeCount(f), 3);
});

ok('matches mirrors the server', () => {
  const f = L.defaultFilters();
  assert.equal(L.matches(row({}), f, kinds), true);
  assert.equal(L.matches(row({ state: 'discarded' }), f, kinds), false, 'live states only by default');
  assert.equal(L.matches(row({ state: 'discarded' }), Object.assign({}, f, { state: 'discarded,trimmed' }), kinds), true);
  assert.equal(L.matches(row({ q: 7 }), Object.assign({}, f, { kind: 'unique' }), kinds), true);
  assert.equal(L.matches(row({ q: 4 }), Object.assign({}, f, { kind: 'unique' }), kinds), false);
  assert.equal(L.matches(row({ code: 'r05' }), Object.assign({}, f, { kind: 'rune' }), kinds), true);
  assert.equal(L.matches(row({ code: 'cap' }), Object.assign({}, f, { kind: 'rune' }), kinds), false);
  assert.equal(L.matches(row({ ilvl: 29 }), Object.assign({}, f, { ilvlMin: '30' }), kinds), false);
  assert.equal(L.matches(row({ ilvl: 31 }), Object.assign({}, f, { ilvlMax: '30' }), kinds), false);
  assert.equal(L.matches(row({ sockets: 2 }), Object.assign({}, f, { sockets: '2' }), kinds), true);
  assert.equal(L.matches(row({ sockets: 0 }), Object.assign({}, f, { sockets: '0' }), kinds), true);
  assert.equal(L.matches(row({ ethereal: false }), Object.assign({}, f, { eth: true }), kinds), false);
  assert.equal(L.matches(row({ identified: true }), Object.assign({}, f, { unid: true }), kinds), false);
  assert.equal(L.matches(row({ pin: true }), Object.assign({}, f, { pinned: true }), kinds), true);
  assert.equal(L.matches(row({ name: 'Rare War Sword' }), Object.assign({}, f, { q: 'war sw' }), kinds), true);
  assert.equal(L.matches(row({ q: 6 }), Object.assign({}, f, { quality: [4, 7] }), kinds), false);
  assert.equal(L.matches(row({ q: 4 }), Object.assign({}, f, { kind: 'missing' }), kinds), false, 'an unknown kind matches nothing');
});
ok('sortItems', () => {
  const a = row({ id: 'a', name: 'b item', q: 4, ilvl: 10, value: 3, found: '2026-11-16T10:00:00Z' });
  const b = row({ id: 'b', name: 'A item', q: 7, ilvl: 5, value: 9, found: '2026-11-16T09:00:00Z' });
  const c = row({ id: 'c', name: 'c item', q: 7, ilvl: 50, value: 1, found: '2026-11-16T11:00:00Z' });
  const ids = (s) => L.sortItems([a, b, c], s).map((x) => x.id).join('');
  assert.equal(ids('new'), 'cab'); assert.equal(ids('old'), 'bac'); assert.equal(ids('quality'), 'cba'); assert.equal(ids('ilvl'), 'cab');
  assert.equal(ids('value'), 'bac'); assert.equal(ids('name'), 'bac'); assert.equal(ids('nope'), 'cab');
});

ok('selection: toggle, mode, anchor', () => {
  const s = L.createSelection();
  assert.equal(s.size, 0); assert.equal(s.mode, false);
  assert.equal(s.toggle('a'), true); assert.equal(s.mode, true); assert.equal(s.anchor, 'a');
  assert.equal(s.toggle('a'), false); assert.equal(s.size, 0);
  s.toggle('b'); assert.deepEqual(s.ids(), ['b']);
});
ok('selection: shift range from the anchor, either direction', () => {
  const order = ['a', 'b', 'c', 'd', 'e', 'f'], s = L.createSelection();
  s.toggle('b'); s.range(order, 'd');
  assert.deepEqual(s.ids().sort(), ['b', 'c', 'd']); assert.equal(s.anchor, 'd');
  s.range(order, 'a');
  assert.deepEqual(s.ids().sort(), ['a', 'b', 'c', 'd']);
  const t = L.createSelection(); t.range(order, 'c'); // no anchor: a plain toggle
  assert.deepEqual(t.ids(), ['c']);
});
ok('selection: select all, prune, clear', () => {
  const s = L.createSelection();
  s.all(['a', 'b', 'c']); assert.equal(s.size, 3);
  s.prune(['b', 'c', 'x']); assert.deepEqual(s.ids().sort(), ['b', 'c']);
  s.prune([]); assert.equal(s.size, 0); assert.equal(s.mode, false, 'the mode ends with the last item');
  s.toggle('a'); s.clear(); assert.equal(s.size, 0); assert.equal(s.anchor, null); assert.equal(s.mode, false);
  s.mode = true; assert.equal(s.mode, true);
  s.prune(['a', 'b']); assert.equal(s.mode, true, 'a refresh while Select is on and nothing is picked keeps the mode (2026-11-22 fit flake)');
});

ok('capacity words', () => {
  assert.deepEqual(L.capacityView({ count: 412, capacity: 600, level: 'ok' }, T), { text: '412 / 600', pct: 69, level: 'ok', now: 412, max: 600, banner: '' });
  assert.equal(L.capacityView({ count: 600, capacity: 600, level: 'full' }, T).banner, 'Full');
  const o = L.capacityView({ count: 450, capacity: 400, level: 'over', over: 50 }, T);
  assert.equal(o.banner, 'Over by 50 (450 held)'); assert.equal(o.pct, 100);
  assert.equal(L.capacityView(null, T).text, '0 / 0');
});
ok('when', () => {
  const now = Date.parse('2026-11-16T12:00:00Z');
  assert.equal(L.when('2026-11-16T11:59:30Z', now), 'just now'); assert.equal(L.when('2026-11-16T11:00:00Z', now), '1 h ago');
  assert.equal(L.when('2026-11-15T10:00:00Z', now), 'yesterday'); assert.equal(L.when('2026-09-01T10:00:00Z', now), '1 Sep 2026'); assert.equal(L.when('x', now), '');
});
ok('tip view numbers the rows and keeps the vault id', () => {
  const tv = L.toTipView([row({ id: 'v-a' }), row({ id: 'v-b' })]);
  assert.equal(tv.view.backpack[1].id, 2); assert.equal(tv.view.backpack[1].vid, 'v-b'); assert.equal(tv.byN.get(1), 'v-a');
});
ok('discard ask', () => { assert.equal(L.discardAsk(10, 10).typed, false); assert.equal(L.discardAsk(11, 10).typed, true); });
ok('reveal plan: the price shown before the click', () => {
  const items = [row({ id: 'a', identified: false }), row({ id: 'b', identified: false }), row({ id: 'c', identified: true }), row({ id: 'd', identified: false, reveal: 'queued' }), row({ id: 'e', identified: false, state: 'sold' })];
  const act = (r) => ({ reveal: Object.assign({ charges: 5, free: false, perItem: 1 }, r) });
  let p = L.revealPlan(items, ['a', 'b', 'c', 'd', 'e', 'zz'], act());
  assert.deepEqual([p.n, p.charges, p.have, p.free, p.short], [2, 2, 5, false, false]);
  p = L.revealPlan(items, ['a', 'b'], act({ charges: 1 }));
  assert.equal(p.short, true);
  p = L.revealPlan(items, ['a', 'b'], act({ charges: 0, free: true }));
  assert.equal(p.short, false); assert.equal(p.free, true); // at Cain: no charge needed
  p = L.revealPlan(items, ['a'], act({ charges: -1 }));
  assert.equal(p.short, false); // unknown: the server decides
  assert.equal(L.revealPlan(items, ['a'], null).n, 1);
});
ok('sell plan: pinned and queued items are skipped with a reason, the gold is the lower bound', () => {
  const items = [row({ id: 'a', value: 30 }), row({ id: 'b', value: 12 }), row({ id: 'c', pin: true, value: 99 }), row({ id: 'd', sell: 'queued', state: 'sellq', value: 7 }), row({ id: 'e', state: 'kept', value: 1 })];
  const p = L.sellPlan(items, ['a', 'b', 'c', 'd', 'e', 'zz']);
  assert.deepEqual([p.n, p.gold, p.pinned, p.skipped, p.all], [2, 42, 1, 4, 6]);
  assert.deepEqual(p.ids, ['a', 'b']);
  assert.deepEqual(p.reasons, [{ code: 'queued', n: 1 }, { code: 'pinned', n: 1 }, { code: 'state', n: 1 }, { code: 'gone', n: 1 }]);
  assert.equal(p.reasons.reduce((a, r) => a + r.n, 0) + p.n, p.all, 'the reasons add up to the selection');
  const q = L.sellPlan(items, ['a', 'd'], true); // the instant sale also takes what the old queue holds
  assert.deepEqual([q.n, q.queued], [2, 0]);
  const e = L.sellPlan(items, []);
  assert.deepEqual([e.n, e.ids, e.gold, e.pinned, e.skipped], [0, [], 0, 0, 0]);
  const T = { sell: 'Sell', sellSel: 'Sell ({n})', sellOf: 'Sell {n} of {all}' };
  assert.equal(L.sellLabel(L.sellPlan(items, ['a', 'b']), 2, T), 'Sell (2)');
  assert.equal(L.sellLabel(p, 6, T), 'Sell 2 of 6');
  assert.equal(L.sellLabel(L.sellPlan(items, []), 0, T), 'Sell');
  assert.equal(L.sellWhy(p.reasons, { queued: '{n} waiting', pinned: '{n} pinned' }), '1 waiting, 1 pinned, 1 state, 1 gone');
});
ok('merged sell quotes: totals add, reasons group, the stash after follows', () => {
  const m = L.mergeQuotes([{ asked: 150, count: 100, lines: [{ id: 'a', gold: 5 }], total: 500, stash: 1000, room: 9, confirmAt: 400, reasons: [{ code: 'pinned', n: 50 }] }, { asked: 20, count: 10, lines: [{ id: 'b', gold: 5 }], total: 50, stash: 1000, confirmAt: 400, reasons: [{ code: 'pinned', n: 4 }, { code: 'room', n: 6, why: 'x' }] }]);
  assert.deepEqual([m.asked, m.count, m.total, m.stash, m.stashAfter, m.needConfirm, m.lines.length], [170, 110, 550, 1000, 1550, true, 2]);
  assert.deepEqual(m.reasons.map((r) => [r.code, r.n]), [['pinned', 54], ['room', 6]]);
});

ok('windowRange draws the rows near the view and pads the rest', () => {
  const w = (o) => L.windowRange(Object.assign({ n: 2400, cols: 6, rowH: 70, view: 700, top: 0, over: 4 }, o));
  let r = w({});
  assert.equal(r.first, 0); assert.equal(r.padTop, 0); assert.equal(r.last, 6 * (10 + 4)); // 10 rows in view plus 4 below
  assert.equal(r.padBot, (400 - 14) * 70);
  r = w({ top: 70 * 100 + 5 }); // deep in the list
  assert.equal(r.first, 6 * 96); assert.equal(r.last, 6 * (100 + 11 + 4)); assert.equal(r.padTop, 96 * 70);
  assert.equal(r.padTop + (r.last - r.first) / 6 * 70 + r.padBot, 400 * 70, 'the padding keeps the full height');
  r = w({ top: 70 * 5000 }); // past the end (a shrunk list): the last rows, never an empty window
  assert.ok(r.last > r.first && r.last === 2400 && r.padBot === 0);
  r = w({ n: 7, cols: 6 }); assert.deepEqual([r.first, r.last, r.padTop, r.padBot], [0, 7, 0, 0]);
  r = w({ n: 0 }); assert.deepEqual([r.first, r.last], [0, 0]);
  r = w({ cols: 1, rowH: 64, view: 640 }); assert.equal(r.last, 14); // the list mode is one column
  r = w({ rowH: 0 }); assert.equal(r.last, 120, 'no layout: a capped first screen'); assert.equal(w({ view: 0, n: 30 }).last, 30);
  for (let top = 0; top < 400 * 70; top += 333) { const q = w({ top }); assert.ok(q.last - q.first <= 6 * (10 + 8 + 2), 'window stays small at ' + top); }
});

// The Reveal bar model: the default payment, the one cost line per option, the skipped line, the confirm step, the order chain.
import { readFileSync } from 'node:fs';
const RT = JSON.parse(readFileSync(new URL('../web/vault.json', import.meta.url), 'utf8')).revealBar;
const opt = (id, o) => Object.assign({ id, label: id, line: id + ' line', ok: true, count: 3, ids: ['a', 'b', 'c'], cost: 0, skipped: [] }, o);
const rev = (o) => Object.assign({ enabled: true, bar: { total: 5, default: 'gold', line: 'x', options: [opt('free', { ok: false, count: 0, ids: [] }), opt('gold', { cost: 5800, count: 5, confirm: true, skipped: [] })] }, gold: { on: true, stash: 7200 } }, o);
ok('bar: default option, one plain line, the press words', () => {
  const m = L.revealBarModel(rev(), '', RT);
  assert.equal(m.show, true); assert.equal(m.sel, 'gold'); assert.equal(m.title, 'Reveal 5 items in this filter');
  assert.equal(m.go, 'Reveal 5 items for 5,800 gold (stash 7,200)'); assert.equal(m.goOff, false); assert.equal(m.confirm, true); assert.match(m.confirmAsk, /5,800 gold/);
  assert.deepEqual(m.options.map((o) => [o.id, o.checked]), [['free', false], ['gold', true]]);
});
ok('reveal methods: hidden for now, only gold unless the server names more (user decision 2026-11-21)', () => {
  assert.deepEqual([...L.revealMethods({ methods: ['gold'] })], ['gold']);
  assert.deepEqual([...L.revealMethods({})].sort(), ['ether', 'free', 'gold']); // an older bot: all three
  const r = rev({ methods: ['gold'], note: 'Cain is not rescued' });
  assert.equal(L.revealBarModel(r, '', RT).note, '', 'the Cain and scroll notice is gone with the free option');
  assert.equal(L.revealBarModel(rev({ methods: ['free', 'gold'], note: 'n' }), '', RT).note, 'n');
});
ok('bar: a pick that cannot be paid falls back to the default', () => {
  assert.equal(L.revealBarModel(rev(), 'free', RT).sel, 'gold');
});
ok('bar: skipped items are counted and named', () => {
  const r = rev(); r.bar.options[1].count = 3; r.bar.options[1].skipped = [{ n: 2, why: 'not enough gold' }];
  const m = L.revealBarModel(r, 'gold', RT);
  assert.equal(m.skippedN, 2); assert.equal(m.skipTitle, '2 skipped'); assert.deepEqual(m.skipped, [{ n: 2, why: 'not enough gold' }]);
});
ok('bar: nothing payable disables the press and says why', () => {
  const r = rev(); r.bar.default = ''; r.bar.options = [opt('free', { ok: false, count: 0, ids: [], why: 'no scroll' })];
  const m = L.revealBarModel(r, '', RT);
  assert.equal(m.sel, ''); assert.equal(m.goOff, true); assert.equal(m.why, 'no scroll');
});
ok('bar: hidden without items or an order; shown for an order alone', () => {
  assert.equal(L.revealBarModel({ bar: { total: 0, options: [] } }, '', RT).show, false);
  assert.equal(L.revealBarModel(undefined, '', RT).show, false);
  const r = { bar: { total: 0, options: [] }, gold: { order: { id: 'g', state: 'queued', total: 40, want: 40, done: 12, paid: 0 } } };
  const m = L.revealBarModel(r, '', RT);
  assert.equal(m.show, true); assert.equal(m.order.active, true); assert.equal(m.order.progress, '12 of 40 revealed');
  assert.deepEqual(m.order.steps.map((s) => s.current), [true, false, false]);
});
ok('bar: a finished or cancelled order ends the chain and cannot be cancelled', () => {
  for (const [st, label] of [['revealed', 'Revealed'], ['partial', 'Done in part'], ['failed', 'Failed'], ['cancelled', 'Cancelled']]) {
    const o = L.orderModel({ id: 'g', state: st, total: 4, want: 4, done: 4, paid: 320, stashAfter: 80 }, RT);
    assert.equal(o.active, false); assert.equal(o.steps[2].current, true); assert.equal(o.steps[2].label, label);
    assert.equal(o.paidText, '320 gold paid'); assert.equal(o.stashText, 'Stash gold: 80');
  }
  assert.equal(L.orderModel({ id: 'g', state: 'paid', total: 4, want: 4, done: 0, paid: 320 }, RT).active, false);
  assert.equal(L.orderModel({ id: 'g', state: 'queued', total: 4, want: 4, done: 0 }, RT).active, true);
});
ok('bar: Ether and free words', () => {
  const r = rev(); r.bar.options = [opt('free', { count: 1, ids: ['a'] }), opt('ether', { cost: 29, count: 29 })]; r.bar.default = 'free';
  assert.equal(L.revealBarModel(r, '', RT).go, 'Reveal 1 item');
  assert.equal(L.revealBarModel(r, 'ether', RT).go, 'Reveal 29 with 29 Ether');
});

ok('equipPlan: ring gets left, right and auto; other items one button; states', () => {
  const ring = { enabled: true, wearable: true, slots: [{ slot: 'lring', side: 'left', can: true, worn: { name: 'A' }, upgrade: true }, { slot: 'rring', side: 'right', can: false, why: 'needs level 30' }] };
  let p = L.equipPlan(ring);
  assert.deepEqual(p.buttons.map((b) => b.slot + (b.can ? '+' : '-')), ['auto+', 'left+', 'right-']);
  assert.equal(p.state, 'ready'); assert.equal(p.ring, true);
  p = L.equipPlan({ enabled: true, wearable: true, slots: [{ slot: 'head', can: true }] });
  assert.deepEqual(p.buttons.map((b) => b.slot), ['auto']);
  p = L.equipPlan({ enabled: true, wearable: false, why: 'charms are not equipped', slots: [] });
  assert.equal(p.state, 'blocked'); assert.equal(p.buttons.length, 0);
  assert.equal(L.equipPlan({ enabled: true, wearable: false, reveal: true, slots: [] }).state, 'reveal');
  assert.equal(L.equipPlan({ enabled: false, offWhy: 'off', wearable: true, slots: [{ slot: 'head', can: true }] }).state, 'off');
  assert.equal(L.equipPlan(Object.assign({ order: { state: 'withdrawing' } }, ring)).state, 'active');
  assert.equal(L.equipPlan(Object.assign({ order: { state: 'done' } }, ring)).state, 'done');
  assert.equal(L.equipPlan(Object.assign({ order: { state: 'failed' } }, ring)).state, 'failed');
  assert.equal(L.equipUpgradeWord({ worn: null }), 'empty'); assert.equal(L.equipUpgradeWord({ worn: {}, upgrade: false }), 'worse'); assert.equal(L.equipUpgradeWord({ worn: {} }), '');
});

console.log('vault logic: ' + n + ' groups ok');

ok('equipOneTap: empty slot fires at once (1 tap), a swap, a ring, a refusal and an active order keep the confirm (2 taps)', () => {
  const empty = { enabled: true, wearable: true, slots: [{ slot: 'head', can: true, worn: null }] };
  assert.equal(L.equipOneTap(empty), true); assert.equal(L.equipTaps(empty), 1);
  const swap = { enabled: true, wearable: true, slots: [{ slot: 'head', can: true, worn: { name: 'Cap' } }] };
  assert.equal(L.equipOneTap(swap), false); assert.equal(L.equipTaps(swap), 2);
  const ring = { enabled: true, wearable: true, slots: [{ slot: 'lring', side: 'left', can: true, worn: null }, { slot: 'rring', side: 'right', can: true, worn: null }] };
  assert.equal(L.equipOneTap(ring), false);
  const twohand = { enabled: true, wearable: true, slots: [{ slot: 'rhand', can: true, worn: null, note: 'shield comes off' }, { slot: 'lhand', can: false, why: 'x' }] };
  assert.equal(L.equipOneTap(twohand), false); // the shield comes off: confirm
  assert.equal(L.equipOneTap({ enabled: true, wearable: true, slots: [{ slot: 'head', can: false, why: 'needs level 30' }] }), false);
  assert.equal(L.equipOneTap({ enabled: true, wearable: false, why: 'charm', slots: [] }), false);
  assert.equal(L.equipOneTap(Object.assign({ order: { state: 'queued' } }, empty)), false);
  assert.equal(L.equipOneTap(null), false);
});

ok('Equip v2: with the direct verbs a swap, a two-hander and a plain item are ONE tap; a ring, a refusal and an unrevealed item are not', () => {
  const swap = { enabled: true, direct: true, wearable: true, slots: [{ slot: 'head', can: true, worn: { name: 'Cap' } }] };
  assert.equal(L.equipOneTap(swap), true); assert.equal(L.equipTaps(swap), 1);
  const twohand = { enabled: true, direct: true, wearable: true, slots: [{ slot: 'rhand', can: true, worn: null, note: 'shield comes off' }] };
  assert.equal(L.equipOneTap(twohand), true);
  const ring = { enabled: true, direct: true, wearable: true, slots: [{ slot: 'lring', side: 'left', can: true }, { slot: 'rring', side: 'right', can: true }] };
  assert.equal(L.equipOneTap(ring), false);
  assert.equal(L.equipOneTap({ enabled: true, direct: true, wearable: true, slots: [{ slot: 'head', can: false, why: 'needs level 30' }] }), false);
  assert.equal(L.equipPlan({ enabled: true, direct: true, wearable: false, reveal: true, slots: [] }).state, 'reveal');
});
ok('Equip v2 toast: worn pieces, what went to the vault, each refused piece with its own reason', () => {
  const TT = { okWornOne: 'Worn in {slot}.', okWornSet: 'Wearing {n} pieces.', okWornOff: '{old} went to the {where}.', okWornMore: '{k} more refused.' };
  const names = { a: 'Cap', b: 'Plate', c: 'Ring' };
  const one = L.equipWornWords({ equipped: [{ id: 'a', slot: 'head', removed: ['Old Cap'], where: 'vault' }] }, names, TT);
  assert.equal(one.kind, 'good'); assert.equal(one.text, 'Worn in head. Old Cap went to the Vault.');
  const set = L.equipWornWords({ equipped: [{ id: 'a', slot: 'head' }, { id: 'b', slot: 'torso' }], failed: [{ id: 'c', error: 'needs level 61' }] }, names, TT);
  assert.equal(set.kind, 'info'); assert.equal(set.text, 'Wearing 2 pieces. Ring: needs level 61');
  const none = L.equipWornWords({ failed: [{ id: 'c', error: 'Reveal it first' }] }, names, TT);
  assert.equal(none.kind, 'bad'); assert.equal(none.text, 'Ring: Reveal it first');
  const four = L.equipWornWords({ failed: [1, 2, 3, 4].map((i) => ({ id: 'a', error: 'x' + i })) }, names, TT);
  assert.ok(four.text.endsWith('1 more refused.'));
});
ok('Equip on merc toast names the mercenary slot', () => {
  const TT = { okWornOne: 'Worn in {slot}.', okWornOff: '{old} went to the {where}.', mercSlots: { mhead: "the mercenary's helm" } };
  const r = L.equipWornWords({ equipped: [{ id: 'a', slot: 'mhead', removed: ['Old Cap'], where: 'vault' }] }, { a: 'Cap' }, TT);
  assert.equal(r.text, "Worn in the mercenary's helm. Old Cap went to the Vault.");
});
