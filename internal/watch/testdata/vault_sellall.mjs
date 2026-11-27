// Sell all against a fake bot (TestVaultSellAllJS): the button never silently differs from the selection,
// the reasons are shown, the confirm sheet states the gold and the stash after, the press sells at once. Two fixture states (the screenshot of 2026-11-22: 260 selected).
import { readFileSync } from 'fs';
import { createRequire } from 'module';
globalThis.dToast = createRequire(import.meta.url)('./toast.cjs');
const dWhy = createRequire(import.meta.url)('./why.cjs');
import { mountVault } from './vault.mjs';
const T = JSON.parse(readFileSync(new URL('./vault.json', import.meta.url), 'utf8'));
const fails = [];
const ok = (c, m) => { if (!c) fails.push(m); };
const mk = (tag) => { const n = { tag, children: [], hidden: false, className: '', textContent: '', dataset: {}, attrs: {}, listeners: {}, disabled: false, value: '', id: '', style: { setProperty() {} },
  classList: { add(c) { n.className += ' ' + c; }, remove() {}, contains() { return false; }, toggle() {} },
  append(...c) { n.children.push(...c); }, appendChild(c) { c.parent = n; n.children.push(c); return c; }, replaceChildren(...c) { n.children = c; c.forEach((x) => { x.parent = n; }); }, setAttribute(k, v) { n.attrs[k] = v; }, getAttribute(k) { return n.attrs[k]; }, removeAttribute(k) { delete n.attrs[k]; },
  addEventListener(e, f) { (n.listeners[e] = n.listeners[e] || []).push(f); }, focus() { doc.activeElement = n; }, contains() { return false; }, querySelectorAll() { return []; }, click() { return fire(n, 'click'); } };
  return n; };
const docL = {};
const doc = { activeElement: null, body: mk('body'), createElement: mk, addEventListener(e, f) { (docL[e] = docL[e] || []).push(f); } };
const winL = {};
const win = { addEventListener(e, f) { (winL[e] = winL[e] || []).push(f); }, scrollY: 0 };
const tick = () => new Promise((r) => setTimeout(r, 5));
const fire = async (n, e, arg) => { for (const f of n.listeners[e] || []) await f(arg || { key: '', preventDefault() {}, target: n }); await tick(); };
const all = (e, o = []) => { o.push(e); (e.children || []).forEach((c) => { c.parent = e; all(c, o); }); return o; };
const root = mk('div'); doc.body.append(root);
const find = (k) => all(root).find((n) => n.dataset && n.dataset.key === k);
const cls = (c) => all(root).filter((n) => (n.className || '').split(/\s+/).includes(c));
const shownText = (n) => (n.hidden ? '' : (n.children && n.children.length ? n.children.map(shownText).join(' ') : n.textContent || ''));


let clock = 0, timers = [], items = [], act = null;
const later = (f, ms) => { const t = { f, at: clock + ms }; timers.push(t); return t; };
const clear = (t) => { timers = timers.filter((x) => x !== t); };
const advance = async (ms) => { clock += ms; for (;;) { const due = timers.filter((t) => t.at <= clock).sort((a, b) => a.at - b.at)[0]; if (!due) break; timers = timers.filter((t) => t !== due); await due.f(); } await tick(); };
const item = (id, extra) => Object.assign({ id, code: 'cap', name: 'Cap ' + id, state: 'recorded', quality: 'magic', q: 4, ilvl: 5, identified: true, kind: 'armor', value: 30, pin: false }, extra);
const reveal0 = { enabled: false, charges: 0, free: false, balance: 0, queued: 0, working: 0, done: 0, methods: ['gold'], bar: { total: 0, default: '', line: '', options: [] }, gold: { on: false } };
let typeN = 20; // the server's count of one slot; sent only when the page asks for facets=1, as the real route does
const view = (p) => ({ schema: 'watch.vault/1', mode: 'on', hero: 'Sorc', total: items.length, offset: 0, limit: 300, items, kinds: [], types: /facets=1/.test(p || '') ? [{ id: 'tors', label: 'Body armor', n: typeN }] : undefined, act, marked: null,
  capacity: { count: items.length, capacity: 400, pins: 0, pinLimit: 0, percent: 1, level: 'ok' }, upgrade: { available: false, currency: 'Ether', reason: 'x', max: 2400 } });
const get = async (p) => (p === '/api/ether' ? { schema: 'watch.ether/1', enabled: false } : view(p));
const posts = [];
let quoteReply = null, sellReply = null;
const post = async (route, body) => {
  posts.push({ route, body });
  if (route === '/api/vault/sellquote') return quoteReply(body);
  if (route === '/api/vault/sell') return sellReply(body);
  return { ok: true, done: [] };
};
const page = mountVault({ toastHost: root, doc, win, root, get, post, later, clear, texts: T, now: () => clock });
const sells = () => posts.filter((p) => p.route === '/api/vault/sell');
const quotes = () => posts.filter((p) => p.route === '/api/vault/sellquote');
const sa = () => find('sell-all');
const docClick = async (target) => { for (const f of docL.click || []) await f({ target, preventDefault() {} }); await tick(); };

// fixture 1: 30 records = 20 plain + 6 pinned + 4 in an equip order; the bot says 20 can go
const fx1 = () => { const o = []; for (let i = 0; i < 20; i++) o.push(item('r' + i)); for (let i = 0; i < 6; i++) o.push(item('p' + i, { pin: true })); for (let i = 0; i < 4; i++) o.push(item('e' + i, { equip: 'queued' })); return o; };
items = fx1();
act = { reveal: reveal0, sell: { enabled: true, instant: true, confirmAt: 2000, queued: 0, working: 0, sold: 0, gold: 0, all: { n: 20, left: [{ code: 'pinned', n: 6, why: 'pinned' }, { code: 'order', n: 4, why: 'in an order' }] } } };
await page.start(); await tick();
ok(sa() && !sa().hidden && sa().textContent === 'Sell all (20)', 'fixture 1: the button counts the 20 that can go, not the 30 records: ' + (sa() && sa().textContent));
ok(sa().dataset.state === 'idle', 'idle state');
// the toolbar holds it beside Identify all, whatever the selection
ok(find('identify-all') !== undefined, 'identify all is still there');

// fixture 2: another mix: 50 plain, 12 pinned, 3 kept, 5 waiting to equip: N is 50
items = []; for (let i = 0; i < 70; i++) items.push(item('x' + i, i < 12 ? { pin: true } : {}));
act.sell.all = { n: 50, left: [] };
await page.reload(false);
ok(sa().textContent === 'Sell all (50)', 'fixture 2: ' + sa().textContent);
// hidden at zero
act.sell.all = { n: 0, left: [] }; await page.reload(false);
ok(sa().hidden === true, 'hidden at N = 0');
act.sell.all = { n: 20, left: [] }; items = fx1(); await page.reload(false);
ok(sa().hidden === false && sa().textContent === 'Sell all (20)', 'and back');

// ---- the first tap arms in place and sends no order ----
let stash = 1000;
quoteReply = () => ({ asked: 30, count: 20, lines: [], total: 412300, stash: 1000, stashAfter: 413300, room: 2499000, confirmAt: 2000, needConfirm: true, reasons: [{ code: 'pinned', n: 6, why: 'pinned' }, { code: 'order', n: 4, why: 'in an order' }] });
posts.length = 0;
await sa().click();
ok(sa().dataset.state === 'confirm', 'armed: ' + sa().dataset.state);
ok(sa().textContent === 'Tap again to sell 20 items for 412,300 gold (stash 1,000 -> 413,300)', 'the label says what the second tap does: ' + sa().textContent);
ok(sells().length === 0, 'the first tap sends no order');
ok(quotes().length === 1 && quotes()[0].body.scope === 'all_sellable' && !quotes()[0].body.ids, 'only the quote, with the scope and no ids: ' + JSON.stringify(quotes()));

// ---- reverts after 6 s ----
await advance(5900); ok(sa().dataset.state === 'confirm', 'still armed at 5.9 s');
await advance(200);
ok(sa().dataset.state === 'idle' && sa().textContent === 'Sell all (20)', 'back after 6 s: ' + sa().textContent + ' ' + sa().dataset.state);
ok(sells().length === 0, 'nothing sent by the revert');

// ---- reverts on Escape, on another tap, on a scroll ----
await sa().click(); ok(sa().dataset.state === 'confirm', 'armed again');
for (const f of docL.keydown || []) await f({ key: 'Escape', preventDefault() {} });
ok(sa().dataset.state === 'idle', 'Escape reverts');
await sa().click(); await docClick(find('select-toggle'));
ok(sa().dataset.state === 'idle', 'another tap reverts');
await sa().click(); for (const f of winL.scroll || []) await f({});
ok(sa().dataset.state === 'idle', 'a scroll reverts');
await sa().click(); await docClick(sa());
ok(sa().dataset.state === 'confirm', 'a tap on the button itself does not revert');
await advance(100);
ok(sells().length === 0, 'still no order');

// ---- the second tap sends once, with the scope and an order id, no ids ----
const slotChip = (id) => { const c = all(root).find((n) => n.dataset && n.dataset.slot === id); return c ? c.textContent : undefined; };
ok(slotChip('tors') === 'Body armor 20', 'the slot chip starts with the count: ' + slotChip('tors'));
sellReply = (b) => (typeN = 0, { ok: true, done: [], sale: { id: 'sa-' + b.orderId, state: 'done', items: 20, done: 20, gold: 412300, stashAfter: 413300, all: true, line: 'Sold 20 for 412,300 gold; 10 skipped: 6 pinned, 4 in an order' } });
posts.length = 0;
await sa().click();
ok(sells().length === 1, 'the second tap sends one order: ' + sells().length);
const b0 = sells()[0].body;
ok(b0.scope === 'all_sellable' && b0.confirm === true && !b0.ids && /^sa[a-z0-9]+$/.test(b0.orderId), 'scope, confirm, an order id and no ids: ' + JSON.stringify(b0));
ok(/Sold 20 for 412,300 gold/.test(shownText(root)), 'the toast');
ok(!sa().disabled && sa().dataset.state === 'idle', 'idle after the sale');
ok(slotChip('tors') === 'Body armor 0' || slotChip('tors') === undefined, 'the type counts moved with the sale, no page refresh needed: ' + slotChip('tors'));
// double tap while the order is busy sends nothing more
act.sell.sale = { id: 'sa-x', state: 'done', items: 20, done: 20, gold: 412300, stashAfter: 413300, all: true, line: 'Sold 20 for 412,300 gold; 10 skipped: 6 pinned, 4 in an order' };
await page.reload(false);
const chip = find('sell-all-chip');
ok(chip && !chip.hidden && chip.textContent === 'Sold 20 for 412,300 gold; 10 skipped: 6 pinned, 4 in an order', 'the result chip: ' + (chip && chip.textContent));
await advance(10100);
ok(find('sell-all-chip').hidden === true, 'the chip goes after ~10 s');
// progress states
act.sell.sale = { id: 'sa-y', state: 'queued', items: 20, done: 0, gold: 100, all: true, line: '' };
await page.reload(false);
ok(find('sell-all-chip').textContent === 'Queued: selling the Vault', 'queued: ' + find('sell-all-chip').textContent);
act.sell.sale = { id: 'sa-y', state: 'credited', items: 20, done: 12, gold: 100, all: true, line: '' };
await page.reload(false);
ok(find('sell-all-chip').textContent === 'Paid: sold 12 of 20', 'paid: ' + find('sell-all-chip').textContent);
act.sell.sale = null; await page.reload(false);

// the id is the same for a resend of one press, new for a new press
await sa().click(); await sa().click(); const id1 = sells()[sells().length - 1].body.orderId;
await sa().click(); await sa().click(); const id2 = sells()[sells().length - 1].body.orderId;
ok(id1 !== id2, 'a new press is a new order id');

// ---- overflow: the stash cannot take it all ----
quoteReply = () => ({ asked: 30, count: 12, lines: [], total: 180, stash: 2499820, stashAfter: 2500000, room: 180, confirmAt: 2000, needConfirm: false, reasons: [{ code: 'room', n: 8, why: 'no room' }, { code: 'pinned', n: 6, why: 'pinned' }] });
await sa().click();
ok(sa().textContent === 'Sell 12 of 20 for 180 gold: stash full (stash 2,499,820 -> 2,500,000). Tap again', 'the overflow label says what it does: ' + sa().textContent);
await docClick(find('select-toggle'));

// ---- a refusal: shown, the button is back, nothing sold ----
sellReply = () => ({ code: 'measured', error: 'a sale is never done in a measured run' });
quoteReply = () => ({ asked: 30, count: 20, lines: [], total: 100, stash: 0, stashAfter: 100, room: 1, confirmAt: 2000, reasons: [] });
await sa().click(); await sa().click();
ok(/never done in a measured run/.test(shownText(root)) && sa().dataset.state === 'idle', 'a refusal is shown: ' + (shownText(root).match(/never[^]{0,40}/) || [''])[0]);

// ---- the old queue (no instant sale): no quote, armed at once ----
act.sell.instant = false; await page.reload(false);
posts.length = 0;
await sa().click();
ok(quotes().length === 0 && sa().textContent === 'Tap again to queue 20 items for your next town visit', 'queue: ' + sa().textContent);
sellReply = (b) => ({ ok: true, done: ['a', 'b', 'c'] });
await sa().click();
ok(sells().length === 1 && /Queued 3 items/.test(shownText(root)), 'queue result');

for (const w of dWhy.unexplained(all(root))) fails.push('why: ' + w);
console.log(JSON.stringify({ fails }));
