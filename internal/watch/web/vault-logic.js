// The Ethereal Vault page's logic without a DOM (docs/vault.md, UI section): the filter state and its query string, the same
// filter and sort the server applies (used to prune and re-order the loaded page after a press, never to decide what the
// vault holds), the multi-select set with shift ranges, and the capacity meter words. Pure functions: node tests run them
// (internal/watch/testdata/vault_logic.mjs). The page is vault.js.

export const QUALITY = { 1: 'low', 2: 'normal', 3: 'superior', 4: 'magic', 5: 'set', 6: 'rare', 7: 'unique', 8: 'crafted' };
export const QUALITY_IDS = Object.keys(QUALITY).map(Number);
export const SORTS = ['new', 'old', 'quality', 'ilvl', 'value', 'name'];
export const PAGE = 60;

export function defaultFilters() {
  return { q: '', quality: [], kind: '', type: '', ilvlMin: '', ilvlMax: '', sockets: '', eth: false, unid: false, pinned: false, state: '', sort: 'new' };
}

const num = (v) => { const n = parseInt(v, 10); return isFinite(n) && n >= 0 ? n : null; };

// toQuery is the query string of GET /api/vault for these filters (a value that is empty or false is left out).
export function toQuery(f, page) {
  const p = new URLSearchParams();
  if (f.q && f.q.trim()) p.set('q', f.q.trim());
  if (f.quality && f.quality.length) p.set('quality', f.quality.slice().sort((a, b) => a - b).join(','));
  if (f.kind) p.set('kind', f.kind);
  if (f.type) p.set('type', f.type);
  if (num(f.ilvlMin) !== null && num(f.ilvlMin) > 0) p.set('ilvlMin', String(num(f.ilvlMin)));
  if (num(f.ilvlMax) !== null && num(f.ilvlMax) > 0) p.set('ilvlMax', String(num(f.ilvlMax)));
  if (f.sockets !== '' && num(f.sockets) !== null) p.set('sockets', String(num(f.sockets)));
  if (f.eth) p.set('eth', '1');
  if (f.unid) p.set('identified', '0');
  if (f.pinned) p.set('pinned', '1');
  if (f.state) p.set('state', f.state);
  if (f.sort && f.sort !== 'new') p.set('sort', f.sort);
  if (page) {
    if (page.offset) p.set('offset', String(page.offset));
    if (page.limit) p.set('limit', String(page.limit));
    if (page.facets) p.set('facets', '1');
  }
  const s = p.toString();
  return s ? '?' + s : '';
}

// fromQuery reads the page's own address back into filters (a deep link, a reload); unknown keys are ignored.
export function fromQuery(search) {
  const p = new URLSearchParams(search || ''), f = defaultFilters();
  f.q = (p.get('q') || '').slice(0, 64);
  f.quality = (p.get('quality') || '').split(',').map(Number).filter((n) => QUALITY[n]);
  f.kind = p.get('kind') || '';
  f.type = p.get('type') || '';
  f.ilvlMin = num(p.get('ilvlMin')) ? String(num(p.get('ilvlMin'))) : '';
  f.ilvlMax = num(p.get('ilvlMax')) ? String(num(p.get('ilvlMax'))) : '';
  f.sockets = p.get('sockets') !== null && num(p.get('sockets')) !== null ? String(num(p.get('sockets'))) : '';
  f.eth = p.get('eth') === '1';
  f.unid = p.get('identified') === '0';
  f.pinned = p.get('pinned') === '1';
  f.state = p.get('state') || '';
  f.sort = SORTS.includes(p.get('sort')) ? p.get('sort') : 'new';
  return f;
}

// activeCount is how many filters narrow the list (the sort does not): the number on the phone's Filters button.
export function activeCount(f) {
  let n = 0;
  if (f.q && f.q.trim()) n++;
  if (f.quality && f.quality.length) n++;
  if (f.kind) n++;
  if (f.type) n++;
  if (num(f.ilvlMin) > 0 || num(f.ilvlMax) > 0) n++;
  if (f.sockets !== '' && num(f.sockets) !== null) n++;
  if (f.eth) n++;
  if (f.unid) n++;
  if (f.pinned) n++;
  if (f.state) n++;
  return n;
}

// matches mirrors the server's filter for one row. kinds: [{id, quality?, codePrefix?}] from /watch/vault.json. The type filter
// is the server's (a base type chain the row does not carry): rows are not filtered by it here.
export function matches(it, f, kinds) {
  if (f.q && f.q.trim() && !String(it.name || '').toLowerCase().includes(f.q.trim().toLowerCase())) return false;
  if (f.quality && f.quality.length && !f.quality.includes(it.q)) return false;
  if (f.kind) {
    const k = (kinds || []).find((x) => x.id === f.kind);
    if (!k) return false;
    if (k.quality && k.quality.length && !k.quality.includes(it.q)) return false;
    if (k.codePrefix && k.codePrefix.length && !k.codePrefix.some((p) => String(it.code || '').startsWith(p))) return false;
  }
  const lo = num(f.ilvlMin), hi = num(f.ilvlMax);
  if (lo && it.ilvl < lo) return false;
  if (hi && it.ilvl > hi) return false;
  if (f.sockets !== '' && num(f.sockets) !== null && (it.sockets || 0) !== num(f.sockets)) return false;
  if (f.eth && !it.ethereal) return false;
  if (f.unid && it.identified) return false;
  if (f.pinned && !it.pin) return false;
  const live = !f.state ? ['recorded', 'keepq', 'sellq'] : f.state.split(',');
  return live.includes(it.state);
}

// sortItems orders rows like the server: new (newest found first), old, quality and ilvl and value (highest first), name (A to Z);
// ties keep the newer first. Rows carry no insertion number, so the found time stands in for it.
export function sortItems(items, sort) {
  const t = (x) => Date.parse(x.found || '') || 0;
  const newer = (a, b) => t(b) - t(a);
  const by = {
    new: newer, old: (a, b) => t(a) - t(b),
    quality: (a, b) => b.q - a.q || newer(a, b), ilvl: (a, b) => b.ilvl - a.ilvl || newer(a, b), value: (a, b) => b.value - a.value || newer(a, b),
    name: (a, b) => String(a.name).toLowerCase().localeCompare(String(b.name).toLowerCase()) || newer(a, b),
  };
  return items.slice().sort(by[sort] || by.new);
}

// createSelection is the multi-select: a set of ids with an anchor for shift ranges. mode is "select mode" (a plain press
// toggles instead of opening the item), set by the Select button, a long press or the first shift press.
export function createSelection() {
  const set = new Set();
  let anchor = null, mode = false;
  const api = {
    has: (id) => set.has(id),
    get size() { return set.size; },
    get mode() { return mode; },
    set mode(v) { mode = !!v; },
    get anchor() { return anchor; },
    ids: () => [...set],
    toggle(id) { if (set.has(id)) set.delete(id); else set.add(id); anchor = id; if (set.size) mode = true; return set.has(id); },
    // range selects every id between the anchor and id in the order shown (inclusive); without an anchor it is a toggle
    range(order, id) {
      const a = order.indexOf(anchor), b = order.indexOf(id);
      if (a < 0 || b < 0) return api.toggle(id);
      for (let i = Math.min(a, b); i <= Math.max(a, b); i++) set.add(order[i]);
      anchor = id; mode = true;
      return true;
    },
    all(order) { order.forEach((id) => set.add(id)); mode = true; },
    clear() { set.clear(); anchor = null; mode = false; },
    // prune drops ids that are no longer in the list (discarded, filtered away); the mode ends with the last one (not on an empty selection)
    prune(order) {
      const keep = new Set(order);
      let dropped = false;
      for (const id of [...set]) if (!keep.has(id)) { set.delete(id); dropped = true; }
      if (anchor !== null && !keep.has(anchor)) anchor = null;
      if (dropped && !set.size) mode = false; // only when the last selected item left: a refresh that arrives while Select is on and nothing is picked yet must not end it (the next tap would open the card)
    },
  };
  return api;
}

// The full banner follows loot.vaultAutoTrim: off (default) nothing is removed and new drops are not recorded.
function fullBanner(c, T, v) {
  if (v && v.autoTrim) return T.fullBannerTrim;
  const fmt = (s, o) => String(s || '').replace(/\{(\w+)\}/g, (m, k) => (o && o[k] !== undefined ? o[k] : ''));
  return fmt(T.fullBanner, { n: c.count, cap: c.capacity });
}

// capacityView words the meter: the text, the level the colour follows, the bar width and the banner (over capacity).
export function capacityView(c, T, v) {
  const cap = (c && c.capacity) || 0, n = (c && c.count) || 0;
  const pct = cap > 0 ? Math.min(100, Math.round((n * 100) / cap)) : 0;
  const level = (c && c.level) || 'ok';
  const fmt = (s, o) => String(s || '').replace(/\{(\w+)\}/g, (m, k) => (o && o[k] !== undefined ? o[k] : ''));
  return {
    text: n + ' / ' + cap, pct, level, now: n, max: cap,
    banner: level === 'over' ? fmt(T.overBanner, { over: c.over, n }) : level === 'full' ? fullBanner(c, T, v) : '',
  };
}

const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
export function when(iso, nowMs) {
  const t = Date.parse(iso || '');
  if (!isFinite(t)) return '';
  const s = Math.max(0, Math.round(((nowMs === undefined ? Date.now() : nowMs) - t) / 1000));
  if (s < 60) return 'just now';
  if (s < 3600) return Math.floor(s / 60) + ' min ago';
  if (s < 86400) return Math.floor(s / 3600) + ' h ago';
  if (s < 172800) return 'yesterday';
  if (s < 30 * 86400) return Math.floor(s / 86400) + ' days ago';
  const d = new Date(t);
  return d.getUTCDate() + ' ' + MONTHS[d.getUTCMonth()] + ' ' + d.getUTCFullYear();
}

// toTipView gives the item tooltip (inventory-tip.js) the rows as an inventory view: each row gets a page-local number as its
// id (the tooltip looks items up by a number), and the map says which vault id that is.
export function toTipView(items) {
  const byN = new Map();
  const backpack = items.map((it, i) => { byN.set(i + 1, it.id); return Object.assign({}, it, { id: i + 1, vid: it.id, sell: it.sellPrice || 0 }); }); // the row's own sell is the Sell errand's state; the tooltip's is the vendor price
  return { view: { backpack, stash: [], belt: [], cube: [], equipment: {} }, byN };
}

// artKey is the icon key of a row for /watch/art/<key>.png, the inventory page's rule (inventory.js artKey): item-<art>, plus
// ~<tint> when the server sent one. '' when the row has no icon name (the cell then shows its name only).
const ART_NAME = /^[a-z0-9_]{1,40}$/, TINT = /^t\d+c\d+$/;
export function artKey(it) {
  if (!it || !it.art || !ART_NAME.test(it.art)) return '';
  return 'item-' + it.art + (it.tint && TINT.test(it.tint) ? '~' + it.tint : '');
}

// discardAsk is what the confirm sheet needs for n items: the typed number is asked above `typedAbove` items.
export function discardAsk(n, typedAbove) { return { n, typed: n > typedAbove }; }

// revealPlan is what a Reveal press on these ids would queue and what it costs, the numbers shown BEFORE the click. n counts the
// unidentified, not yet queued records that are still in the vault; free means Cain stands in the town (no charge is used);
// short means the pack holds fewer charges than n (have -1: unknown, the server decides).
export function revealPlan(items, ids, act) {
  const by = new Map((items || []).map((i) => [i.id, i]));
  const ok = (ids || []).filter((id) => { const it = by.get(id); return it && !it.identified && !it.reveal && (it.state === 'recorded' || it.state === 'keepq'); });
  const n = ok.length;
  const r = (act && act.reveal) || {}, per = r.perItem || 1, have = typeof r.charges === 'number' ? r.charges : -1, free = !!r.free;
  return { n, ids: ok, charges: n * per, have, free, short: !free && n > 0 && have >= 0 && have < n * per };
}

// sellPlan is what a Sell press would queue: live recorded items that are not pinned or queued; skipped counts the rest.
// gold is the vault's lower-bound price of the queued ones.
export function sellPlan(items, ids, instant) {
  const by = new Map((items || []).map((i) => [i.id, i]));
  let n = 0, gold = 0, pinned = 0, selling = 0, queued = 0, state = 0, gone = 0;
  const ok = [];
  for (const id of ids || []) {
    const it = by.get(id);
    if (!it) { gone++; continue; }
    if (it.pin) { pinned++; continue; }
    if (it.sell === 'selling') { selling++; continue; }
    if (it.sell === 'queued' && !instant) { queued++; continue; } // an instant sale also takes what the old queue still holds
    if (it.state !== 'recorded' && !(instant && it.sell === 'queued')) { state++; continue; }
    n++; gold += it.value || 0; ok.push(id);
  }
  const all = (ids || []).length;
  // reasons: why the rest is left out, in the order the page words them (counts add up to all - n)
  const reasons = [['queued', queued], ['selling', selling], ['pinned', pinned], ['state', state], ['gone', gone]].filter(([, c]) => c > 0).map(([code, c]) => ({ code, n: c }));
  return { n, ids: ok, gold, pinned, skipped: all - n, all, queued, selling, state, gone, reasons };
}

// sellWhy words the reasons of a plan or a server quote ("169 already waiting to be sold, 20 pinned"). words maps a code to a
// phrase with {n}; a code without one is shown as its server sentence (why) or the code itself.
export function sellWhy(reasons, words) {
  return (reasons || []).map((r) => {
    const w = words && words[r.code];
    return w ? w.replace('{n}', String(r.n)) : r.n + ' ' + (r.why || r.code);
  }).join(', ');
}

// sellLabel is the Sell button's text: "Sell (11)" when every selected item can go, "Sell 11 of 260" when some cannot (the
// reasons are shown next to the button and in the sheet; the number never silently differs from the selection).
export function sellLabel(plan, selected, T) {
  const total = plan.all || selected;
  if (!total) return T.sell;
  return plan.n === total ? T.sellSel.replace('{n}', String(plan.n)) : T.sellOf.replace('{n}', String(plan.n)).replace('{all}', String(total));
}

// mergeQuotes joins the answers of /api/vault/sellquote for a selection sent in chunks of up to `chunk` ids.
export function mergeQuotes(qs) {
  const out = { asked: 0, count: 0, lines: [], total: 0, stash: 0, stashAfter: 0, room: 0, reasons: [], confirmAt: 0, needConfirm: false, why: '' };
  qs.forEach((q, i) => {
    out.asked += q.asked || 0; out.count += q.count || 0; out.total += q.total || 0;
    (q.lines || []).forEach((l) => out.lines.push(l));
    if (i === 0) { out.stash = q.stash || 0; out.room = q.room || 0; out.confirmAt = q.confirmAt || 0; }
    if (q.why && !out.why) out.why = q.why;
    (q.reasons || []).forEach((r) => {
      const x = out.reasons.find((y) => y.code === r.code);
      if (x) x.n += r.n; else out.reasons.push({ code: r.code, n: r.n, why: r.why });
    });
  });
  out.stashAfter = out.stash + out.total;
  out.needConfirm = out.confirmAt > 0 && out.total >= out.confirmAt;
  return out;
}

// windowRange says which cells of a long list to draw (the page keeps up to thousands of rows; only the rows near the view are
// in the DOM). n items in rows of cols, every row rowH tall; top is how far the first row's top is above the view's top, view the
// view's height, over the rows kept above and below it. first and last are item indexes (last exclusive, row aligned); padTop
// and padBot stand in for the rows not drawn. Without a layout (rowH or view not positive) everything up to a cap is drawn.
export function windowRange(o) {
  const n = Math.max(0, o.n | 0), cols = Math.max(1, o.cols | 0), rowH = +o.rowH, view = +o.view, over = Math.max(0, o.over | 0);
  if (!(rowH > 0) || !(view > 0)) return { first: 0, last: Math.min(n, o.cap || 120), padTop: 0, padBot: 0 };
  const rows = Math.ceil(n / cols);
  const top = Math.max(0, +o.top || 0);
  const fr = Math.min(Math.max(0, Math.floor(top / rowH) - over), Math.max(0, rows - 1));
  const lr = Math.min(rows, Math.max(fr + 1, Math.ceil((top + view) / rowH) + over));
  return { first: fr * cols, last: Math.min(n, lr * cols), padTop: fr * rowH, padBot: (rows - lr) * rowH };
}

// The Reveal bar (docs/vault.md, "Reveal paid in gold"): one bar over the list instead of a notice paragraph. The model is pure:
// the server's quote (act.reveal.bar, act.reveal.gold) and the player's pick in, what to draw out.
const ORDER_STEPS = ['queued', 'paid', 'revealed'];
const f2 = (s, o) => String(s || '').replace(/\{(\w+)\}/g, (m, k) => (o && o[k] !== undefined ? o[k] : ''));
const grp = (n) => String(Math.max(0, n | 0)).replace(/\B(?=(\d{3})+(?!\d))/g, ',');

// orderModel words a gold order: the chain queued -> paid -> revealed (it runs for seconds), where it stands, and whether it can
// be cancelled (only before the payment is confirmed). A partial, failed or cancelled order ends the chain with its own label.
export function orderModel(o, T) {
  if (!o) return null;
  const ended = o.state === 'revealed' || o.state === 'partial' || o.state === 'failed' || o.state === 'cancelled';
  const at = ended ? ORDER_STEPS.length - 1 : Math.max(0, ORDER_STEPS.indexOf(o.state));
  const steps = ORDER_STEPS.map((s, i) => ({ id: s, label: T.steps[s], current: i === at, past: i < at }));
  if (o.state === 'partial' || o.state === 'failed' || o.state === 'cancelled') steps[steps.length - 1].label = T.steps[o.state];
  const paidText = o.paid ? f2(T.paidGold, { gold: grp(o.paid) }) : '';
  const stashText = o.paid ? f2(T.stashGold, { n: grp(o.stashAfter) }) : '';
  return { id: o.id, state: o.state, steps, active: o.state === 'queued', progress: f2(T.progress, { done: o.done, total: o.want || o.total }), paidText, stashText, line: o.line, note: o.note || '' };
}

// revealMethods is the set of reveal payments the page draws: the server's act.reveal.methods (knob loot.vaultRevealMethods, default only
// gold; the others are hidden for now, user decision 2026-11-21). Without the field (an older bot) all three show.
export function revealMethods(rev) {
  return new Set(rev && Array.isArray(rev.methods) ? rev.methods : ['free', 'ether', 'gold']);
}

// identifyAllPlan is the always-visible "Identify all (N)" button and its confirm sheet. N is the server's count of unrevealed
// records of the WHOLE vault (act.reveal.gold.unrevealed: no filter, search or selection applies). K is what the stash pays for.
// show is false at N = 0, without gold, or while only other payments are offered; off says the press cannot go (a running order,
// the server's reason, or a stash below one reveal).
export function identifyAllPlan(rev, T) {
  const g = rev && rev.gold, n = g ? g.unrevealed | 0 : 0, price = g ? g.scrollGold | 0 : 0, stash = g ? g.stash | 0 : 0;
  const m = { show: false, n, k: 0, cost: 0, stash, after: stash, short: false, off: false, why: '', label: '', title: '', body: '', go: '' };
  const f = (str, o) => String(str || '').replace(/\{(\w+)\}/g, (x, key) => (o[key] !== undefined ? o[key] : ''));
  const via = revealMethods(rev);
  if (!g || !g.on || n <= 0 || !via.has('gold') || via.has('free')) return m;
  m.show = true;
  m.k = price > 0 ? Math.min(n, Math.floor(stash / price)) : 0;
  m.cost = m.k * price; m.after = stash - m.cost; m.short = m.k < n;
  m.label = f(T.identifyAll, { n });
  const o = g.order;
  if (o && (o.state === 'queued' || o.state === 'paid')) { m.off = true; m.why = T.identifyAllRunning; }
  else if (g.why) { m.off = true; m.why = g.why; }
  else if (m.k < 1) { m.off = true; m.why = f(T.goldShort, { price: grp(price), stash: grp(stash) }); }
  const a = { n, k: m.k, cost: grp(m.cost), stash: grp(stash), after: grp(m.after) };
  m.title = f(T.identifyAllTitle, a);
  m.body = m.short ? f(T.identifyAllShort, a) : f(T.identifyAllBody, a);
  m.go = f(T.identifyAllGo, { k: m.k });
  return m;
}

// sellAllPlan is the always-visible "Sell all (N)" button. N is the server's count of the records a whole-vault sale would take
// now (act.sell.all.n: not pinned, not kept, in no active order; the filter, search and selection never matter). The button is
// hidden at N = 0; off says the press cannot go (the bot's reason).
export function sellAllPlan(sell, T) {
  const a = sell && sell.all, n = a ? a.n | 0 : 0;
  const m = { show: n > 0, n, off: false, why: '', label: String(T.sellAll || '').replace('{n}', String(n)), instant: !!(sell && sell.instant) };
  if (sell && !sell.enabled) { m.off = true; m.why = sell.why || ''; }
  return m;
}

// sellAllConfirm words the armed button from the quote of scope all_sellable (the same plan the order uses, worked out now; the
// bot lists the records again when the second tap arrives, so these numbers are an estimate). Without a quote (the old town-visit
// queue) it names the count only. partial is true when the stash cannot take all of it.
export function sellAllConfirm(q, n, T) {
  const f = (str, o) => String(str || '').replace(/\{(\w+)\}/g, (x, key) => (o[key] !== undefined ? o[key] : ''));
  if (!q) return { text: f(T.sellAllArmQueue, { n }), n, k: n, partial: false };
  const room = ((q.reasons || []).find((r) => r.code === 'room') || {}).n | 0, k = q.count | 0, all = k + room;
  const a = { n: k, all, gold: grp(q.total), stash: grp(q.stash), after: grp(q.stashAfter) };
  return { text: room > 0 ? f(T.sellAllArmFull, a) : f(T.sellAllArm, a), n: k, k, all, partial: room > 0, gold: q.total | 0 };
}

// sellAllChip is the progress and result chip of a Sell all order: Queued, Paid, Sold n of N, then the server's result line
// ("Sold 187 for 412,300 gold; 9 skipped: 4 pinned, 5 in an order"). null for any other sale.
export function sellAllChip(sale, T) {
  if (!sale || !sale.all) return null;
  const f = (str, o) => String(str || '').replace(/\{(\w+)\}/g, (x, key) => (o[key] !== undefined ? o[key] : ''));
  const a = { n: sale.done | 0, all: sale.items | 0, gold: grp(sale.gold) };
  switch (sale.state) {
    case 'queued': return { level: 'info', text: f(T.sellAllQueued, a), step: 'queued' };
    case 'credited': return { level: 'info', text: f(T.sellAllPaid, a), step: 'paid' };
    case 'done': return { level: sale.note ? 'info' : 'good', text: sale.line || f(T.sellAllDone, a), step: 'done' };
    default: return { level: 'bad', text: sale.line || sale.note || '', step: 'failed' };
  }
}

// revealBarModel is what the bar draws. rev is act.reveal of GET /api/vault, pay the option the player picked ('' = the server's
// default). show is false when there is nothing to reveal and no order to follow.
export function revealBarModel(rev, pay, T) {
  const bar = rev && rev.bar, order = orderModel(rev && rev.gold && rev.gold.order, T);
  if (!bar || (!bar.total && !order)) return { show: false };
  const opts = bar.options || [];
  const pick = opts.find((o) => o.id === pay && o.ok) || opts.find((o) => o.id === bar.default) || null;
  const m = { show: true, total: bar.total, title: f2(bar.total === 1 ? T.titleOne : T.title, { n: bar.total }), line: bar.line, order, sel: pick ? pick.id : '', confirm: false, go: '', goOff: true, why: '', skipped: [], skippedN: 0, note: revealMethods(rev).has('free') ? rev.note || '' : '' };
  m.options = opts.map((o) => ({ id: o.id, label: T.pay[o.id] || o.label, line: o.line, ok: o.ok, why: o.why || '', checked: !!pick && o.id === pick.id }));
  if (!bar.total) return m;
  if (!pick) { m.why = (opts.find((o) => o.id === 'free') || {}).why || T.nothingPayable; return m; }
  const n = pick.count;
  m.go = pick.id === 'gold' ? f2(n === 1 ? T.goGoldOne : T.goGold, { n, cost: grp(pick.cost), stash: grp(rev.gold ? rev.gold.stash : 0) }) : pick.id === 'ether' ? f2(T.goEther, { n, cost: grp(pick.cost) }) : f2(n === 1 ? T.goFreeOne : T.goFree, { n });
  m.goOff = !pick.ok || !rev.enabled && pick.id !== 'ether';
  if (m.goOff && !pick.ok) m.why = pick.why;
  else if (m.goOff) m.why = rev.why || '';
  m.goLabel = m.go; m.goSub = '';
  if (pick.id === 'gold') { m.goLabel = f2(n === 1 ? T.goGoldLabelOne : T.goGoldLabel, { n }); m.goSub = f2(T.goGoldSub, { cost: grp(pick.cost), stash: grp(rev.gold ? rev.gold.stash : 0) }); } // the cost is a muted line under the button's name
  m.confirm = !!pick.confirm;
  m.confirmAsk = pick.id === 'gold' ? f2(T.confirmGold, { cost: grp(pick.cost), n }) : '';
  m.skipped = (pick.skipped || []).map((s) => ({ n: s.n, why: s.why }));
  m.skippedN = Math.max(0, bar.total - n);
  if (m.skippedN && !m.skipped.length) m.skipped = [{ n: m.skippedN, why: T.skipOther }];
  m.skipTitle = f2(T.skipped, { n: m.skippedN });
  return m;
}

// equipPlan turns the GET /api/vault/equip/<id> answer into what the Equip sheet draws: the state, the buttons and the line.
// The three states of the sheet: off (the knob or the gate says no, why), blocked (the item cannot be worn: a charm, an
// unidentified item, a requirement), ready (buttons) and active/done/failed (an order exists). Rings get a left and a right
// button besides the default "replace weaker"; every other item one button. Nothing here decides: the server refuses again.
export function equipPlan(sh) {
  if (!sh) return { state: 'none', buttons: [] };
  const order = sh.order || null;
  const slots = sh.slots || [];
  const ring = slots.some((x) => x.side);
  const buttons = [];
  if (sh.wearable) {
    const can = slots.filter((x) => x.can);
    buttons.push({ slot: 'auto', ring, can: can.length > 0, why: can.length ? '' : (slots[0] && slots[0].why) || '' });
    if (ring) {
      for (const side of ['left', 'right']) {
        const x = slots.find((y) => y.side === side);
        if (x) buttons.push({ slot: side, ring, can: !!x.can, why: x.why || '', view: x });
      }
    }
  }
  let state = 'ready';
  if (order && ['queued', 'withdrawing', 'equipping'].includes(order.state)) state = 'active';
  else if (order && order.state === 'done') state = 'done';
  else if (order && order.state === 'failed') state = 'failed';
  else if (!sh.enabled) state = 'off';
  else if (!sh.wearable) state = sh.reveal ? 'reveal' : 'blocked';
  return { state, buttons, ring, order, why: sh.why || '', offWhy: sh.offWhy || '' };
}

// equipUpgradeWord is the sheet's one word on whether the candidate beats what is worn ('' when the page cannot tell).
export function equipUpgradeWord(sv) {
  if (!sv || !sv.worn) return 'empty';
  if (sv.upgrade === true) return 'better';
  if (sv.upgrade === false) return 'worse';
  return '';
}

// equipOneTap says whether the Equip tap on an item may fire the order at once (no confirm sheet): the item is wearable,
// the choice is not a ring side, and every slot it could take is empty, so nothing is swapped out. A swap or a ring keeps
// the sheet as the one confirm. equipTaps counts the taps after the item is open: 1 (Equip) or 2 (Equip, confirm).
export function equipOneTap(sh) {
  const p = equipPlan(sh);
  if (p.state !== 'ready' || p.ring) return false;
  const auto = p.buttons.find((b) => b.slot === 'auto');
  if (!auto || !auto.can) return false;
  if (sh.direct) return true; // Equip v2: the swap is instant and the old piece goes to the Vault, so no confirm
  return (sh.slots || []).filter((x) => x.can).every((x) => !x.worn && !x.note); // a note is the two-hander pushing a shield off
}
export function equipTaps(sh) { return equipOneTap(sh) ? 1 : 2; }

// ---- the item sheet on a phone: keep the selected cell in view, and compare with what is worn ----

// clearDelta is how far the page must scroll (positive: down) so the cell sits fully in the band between the top bar
// (top px) and the bottom sheet (sheetH px high, in a viewport vh px high), margin m px from both. The cell is at rect
// {top, bottom}. When the band is shorter than the cell the cell's top wins. 0 when it is in the band already.
export function clearDelta(rect, vh, sheetH, top, m = 8) {
  const lo = (top || 0) + m, hi = vh - (sheetH || 0) - m;
  if (rect.top < lo) return rect.top - lo;
  if (rect.bottom > hi) return Math.min(rect.bottom - hi, rect.top - lo);
  return 0;
}

export const COMPARE_MODES = ['off', 'hero', 'merc'];
// nextCompare is the one button: off, then the hero, then the merc, then off. An unknown stored value is off.
export function nextCompare(mode) { const i = COMPARE_MODES.indexOf(mode); return COMPARE_MODES[(i + 1) % COMPARE_MODES.length] || 'off'; }
export function validCompare(mode) { return COMPARE_MODES.includes(mode) ? mode : 'off'; }

// comparePlan says what the compare block of an open item draws. mode: off|hero|merc. sh: GET /api/vault/equip/<id> (false or
// null when the bot has no such route): sh.slots are the hero's slots for the item (both ring slots for a ring), sh.merc the
// merc side. inv: GET /api/inventory (equipment, merc.equipment). cand: the item as the tooltip sees it (toTipView row).
// cmp(cand, worn) is the inventory tooltip's own comparison (inventory-tip.js compare: rows {label, old, now, delta, tone}); the
// numbers are never worked out here. Result: {state: off|loading|refused|ok, who, why, blocks: [{slot, label, worn, empty, note, rows}]}.
export function comparePlan(mode, sh, inv, cand, cmp, T) {
  mode = validCompare(mode);
  if (mode === 'off') return { state: 'off', blocks: [] };
  if (sh === undefined || inv === undefined) return { state: 'loading', who: mode, blocks: [] };
  if (!sh || !inv) return { state: 'refused', who: mode, why: T.cmpNoData, blocks: [] };
  if (!sh.wearable) return { state: 'refused', who: mode, why: sh.why || T.cmpNotWorn, blocks: [] };
  const block = (slot, label, worn, note) => {
    const b = { slot, label, note: note || '', worn: worn ? (worn.name || worn.code || '') : '', empty: !worn, rows: [] };
    if (worn) b.rows = cmp(cand, worn) || [];
    return b;
  };
  if (mode === 'merc') {
    const m = sh.merc;
    if (!m || !m.owned) return { state: 'refused', who: mode, why: (m && m.why) || T.cmpNoMerc, blocks: [] };
    if (!m.can) { const known = !!m.kind; return { state: 'refused', who: mode, why: known ? f2(T.cmpMercNo, { kind: m.kind }) : (m.why || T.cmpMercNo), mercNo: known, blocks: [] }; } // no kind: he is out of view, the server's sentence says so
    const worn = inv.merc && inv.merc.equipment && inv.merc.equipment[m.slot] || null;
    return { state: 'ok', who: mode, blocks: [block(m.slot, T.cmpSlots[m.slot] || m.slot, worn, '')] };
  }
  const eq = inv.equipment || {};
  const blocks = (sh.slots || []).map((x) => {
    let note = x.note || '';
    if (x.note && x.slot === 'rhand' && eq.lhand) note = f2(T.cmpTakesOff, { name: eq.lhand.name || eq.lhand.code || '' }); // a two-hander pushes the shield off
    else if (!x.can && x.why) note = x.why;
    return block(x.slot, x.label || x.slot, eq[x.slot] || null, note);
  });
  if (!blocks.length) return { state: 'refused', who: mode, why: T.cmpNotWorn, blocks: [] };
  return { state: 'ok', who: mode, blocks };
}

// equipWornWords turns the Equip v2 reply (`equipped`: per piece slot, removed, where; `failed`: id, error) into the toast: what
// was worn, what went to the Vault, and each refused piece with its own reason. names maps id -> item name.
export function equipWornWords(rep, names, T) {
  const worn = (rep && rep.equipped) || [], failed = (rep && rep.failed) || [];
  const nm = (id) => (names && names[id]) || id;
  const parts = [];
  if (worn.length) {
    const off = worn.reduce((a, x) => a.concat(x.removed || []), []);
    const where = (worn.find((x) => x.where && x.where !== 'vault') || {}).where;
    parts.push((worn.length === 1 ? T.okWornOne.replace('{slot}', (T.mercSlots && T.mercSlots[worn[0].slot]) || worn[0].slot) : T.okWornSet.replace('{n}', String(worn.length)))
      + (off.length ? ' ' + T.okWornOff.replace('{old}', off.join(', ')).replace('{where}', where || 'Vault') : ''));
  }
  failed.slice(0, 3).forEach((x) => parts.push(nm(x.id) + ': ' + (x.error || x.code)));
  if (failed.length > 3) parts.push(T.okWornMore.replace('{k}', String(failed.length - 3)));
  return { text: parts.join(' '), kind: worn.length ? (failed.length ? 'info' : 'good') : 'bad' };
}
