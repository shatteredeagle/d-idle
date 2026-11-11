// Layout manifest logic (docs/ui-design-system.md section 6.4), pure: no DOM, so the node tests run it as is. The
// shell (shell.js) applies a plan to the page; the editor (layoutedit.js) moves entries with the same packer as the
// mockup editor (docs/mockups/ui/editor.html).

export const WIDTHS = [['phone', 0], ['tablet', 640], ['laptop', 1024], ['wide', 1440]]; // the breakpoints of section 4.2

export function breakpointFor(width, mode) {
  if (mode === 'overlay') return 'overlay';
  let name = 'phone';
  for (const [n, min] of WIDTHS) if (width >= min) name = n;
  return name;
}

// The breakpoint entry for a name; a missing one falls back to the next wider (never to overlay), then to null.
export function pickBreakpoint(layout, name) {
  if (!layout || layout.schema !== 'watch.layout/1' || !layout.breakpoints) return null;
  const order = WIDTHS.map((w) => w[0]);
  const from = order.indexOf(name);
  const names = from < 0 ? [name] : order.slice(from);
  for (const n of names) {
    const bp = layout.breakpoints[n];
    if (bp && Array.isArray(bp.panels) && bp.panels.length) return { name: n, cols: bp.cols || layout.cols || 12, panels: bp.panels };
  }
  return null;
}

// ---- the packer (the editor's): first free cell top-down, and push down on a collision ----
export function occ(panels, skip) {
  const m = {};
  for (const p of panels) {
    if (p === skip || p.hidden) continue; // a hidden panel reserves nothing
    const h = p.collapsed ? 1 : p.rowSpan;
    for (let r = p.row; r < p.row + h; r++) for (let c = p.col; c < p.col + p.colSpan; c++) m[r + ':' + c] = p.id;
  }
  return m;
}
export function free(m, c, r, w, h) {
  for (let y = r; y < r + h; y++) for (let x = c; x < c + w; x++) if (m[y + ':' + x]) return false;
  return true;
}
export function placeNear(panels, p) {
  const m = occ(panels, p), h = p.collapsed ? 1 : p.rowSpan;
  for (let r = p.row; r < p.row + 400; r++) if (free(m, p.col, r, p.colSpan, h)) { p.row = r; return; }
}
function placeFrom(panels, p, cols, fromRow) {
  const m = occ(panels, p);
  for (let r = fromRow; r < fromRow + 400; r++) {
    for (let c = 1; c + p.colSpan - 1 <= cols; c++) if (free(m, c, r, p.colSpan, p.rowSpan)) { p.col = c; p.row = r; return; }
  }
}

const SPAN = { s: [3, 5], m: [6, 6], l: [12, 6] }; // columns and rows of a panel the layout does not mention

// plan(layout, width, mode, conf, targets): where every target goes. targets are the placeable elements of the page
// in reading order ({id, size}); conf is the layout block of /watch/panels.json (aliases, compat, reserved). Returns
// null when no breakpoint applies (the shell then places by manifest), else
// { name, cols, rowHeight, flow, items: [{id, col, row, colSpan, rowSpan, collapsed, order, placed}], unknown: [ids] }.
// flow (phone and tablet): rows are not fixed, the shell uses order and the column span and lets the rows flow.
// A hidden target (a locked feature, a route that is off) takes no cell and the panels under it rise into the room, so the
// page has no dead bands. With edit set (the editor) the cells are the same as on the normal page, and the hidden
// panels the file names come along as { hidden: true } at their file position: they are saved again but reserve no
// cell (they once did, invisibly, which is why a panel could not be dropped into room that looked free).
export function plan(layout, width, mode, conf, targets, edit) {
  const bpName = breakpointFor(width, mode);
  const bp = pickBreakpoint(layout, bpName);
  if (!bp) return null;
  const aliases = (conf && conf.aliases) || {}, reserved = (conf && conf.reserved) || [];
  const known = {};
  targets.forEach((t) => { known[t.id] = t; });
  const items = [], used = {}, unknown = [], kept = [], off = [];
  let maxRow = 0, maxOrder = -1;
  for (const e of bp.panels) {
    const id = known[e.id] ? e.id : aliases[e.id];
    if (!id || !known[id]) { if (!reserved.includes(e.id)) unknown.push(e.id); continue; }
    if (used[id]) continue; // the first entry wins when an alias and its target are both named
    used[id] = true;
    const colSpan = Math.max(1, Math.min(bp.cols, e.colSpan | 0)), col = Math.max(1, Math.min(bp.cols - colSpan + 1, e.col | 0));
    const it = { id, col, row: Math.max(1, e.row | 0), colSpan, rowSpan: Math.max(1, e.rowSpan | 0), collapsed: !!e.collapsed, order: e.order | 0, placed: true };
    // an entry with hidden: true is a panel the user switched off in the editor: no cell, and the shell hides it
    if (e.hidden) { off.push(id); if (edit) kept.push(Object.assign(it, { hidden: true, off: true })); continue; }
    if (known[id].hidden) { if (edit) kept.push(Object.assign(it, { hidden: true })); continue; }
    placeNear(items, it); // a file with overlapping entries (the editor's presets have some) still loads: push down
    items.push(it);
    maxRow = Math.max(maxRow, it.row + (it.collapsed ? 1 : it.rowSpan) - 1);
    maxOrder = Math.max(maxOrder, it.order);
  }
  // panels the layout does not mention are packed after the placed ones, in page order
  const flow = bpName === 'phone' || bpName === 'tablet';
  for (const t of targets) {
    if (used[t.id] || t.hidden) continue;
    const [w, h] = SPAN[t.size] || SPAN.s;
    const it = { id: t.id, col: 1, row: maxRow + 1, colSpan: flow && bpName === 'phone' ? bp.cols : Math.min(bp.cols, flow ? Math.max(w, 6) : w), rowSpan: h, collapsed: false, order: ++maxOrder, placed: false };
    placeFrom(items, it, bp.cols, maxRow + 1);
    items.push(it);
  }
  if (!flow) { // each panel rises in its own columns into the room a hidden one left
    const done = [];
    for (const it of items.slice().sort((x, y) => x.row - y.row || x.col - y.col)) {
      const m = occ(done), h = it.collapsed ? 1 : it.rowSpan;
      while (it.row > 1 && free(m, it.col, it.row - 1, it.colSpan, h)) it.row--;
      done.push(it);
    }
  }
  items.push(...kept);
  return { name: bp.name, bp: bpName, cols: bp.cols, rowHeight: bpName === 'overlay' ? 40 : (layout.rowHeight || 48), flow, items, unknown, off };
}

// The entries of a plan as a breakpoint's panels array (what the editor saves), order from row then column.
export function toPanels(items, bands) {
  return items.slice().sort((a, b) => a.row - b.row || a.col - b.col).map((it, i) => {
    const o = { id: it.id, band: (bands && bands[it.id]) || 'grid', col: it.col, row: it.row, colSpan: it.colSpan, rowSpan: it.rowSpan, collapsed: !!it.collapsed, order: i };
    if (it.off) o.hidden = true; // only a switched-off panel carries the key, so other entries are as the mockup editor writes them
    return o;
  });
}

// ---- the editor's move: push down, cascade, close up (a dashboard grid) ----
const rows = (p) => (p.collapsed ? 1 : p.rowSpan);
const hits = (a, b) => a !== b && a.col < b.col + b.colSpan && b.col < a.col + a.colSpan && a.row < b.row + rows(b) && b.row < a.row + rows(a);

// moveItem(items, id, rect, cols): the layout after panel id takes rect ({col, row, colSpan, rowSpan, collapsed}; any
// part may be left out). It never changes items: the result is a copy, so a preview that is dropped costs nothing and
// the layout before the drag is still there to go back to. The moved panel lands exactly on its cells; every panel it
// covers is pushed below it, the ones those cover are pushed in turn; then each panel rises in its columns as far as
// there is room (the moved one first where two start level), which is what the normal page does too. Hidden panels
// neither block nor move. A move onto free cells changes nothing else.
export function moveItem(items, id, rect, cols) {
  const out = items.map((i) => Object.assign({}, i));
  const it = out.find((i) => i.id === id && !i.hidden);
  if (!it) return out;
  Object.assign(it, rect);
  it.colSpan = Math.max(1, Math.min(cols, it.colSpan | 0));
  it.col = Math.max(1, Math.min(cols - it.colSpan + 1, it.col | 0));
  it.row = Math.max(1, it.row | 0);
  it.rowSpan = Math.max(1, it.rowSpan | 0);
  const vis = out.filter((i) => !i.hidden);
  const queue = [it];
  for (let guard = 0; queue.length && guard < 20000; guard++) {
    const a = queue.shift();
    for (const b of vis.slice().sort((x, y) => x.row - y.row || x.col - y.col)) {
      if (b !== it && hits(a, b)) { b.row = a.row + rows(a); queue.push(b); }
    }
  }
  rise(vis, it);
  return out;
}

function rise(vis, first) {
  const done = [];
  for (const x of vis.slice().sort((p, q) => p.row - q.row || (p === first ? -1 : q === first ? 1 : 0) || p.col - q.col)) {
    const m = occ(done);
    while (x.row > 1 && free(m, x.col, x.row - 1, x.colSpan, rows(x))) x.row--;
    done.push(x);
  }
}

// setOff(items, id, on, cols): the layout after the user hides (on) or shows a panel. A hidden panel keeps its cells in
// the file and reserves nothing; the others rise into its room. Shown again it takes its cells back and pushes down.
export function setOff(items, id, on, cols) {
  const out = items.map((i) => Object.assign({}, i));
  const it = out.find((i) => i.id === id);
  if (!it || (it.hidden && !it.off)) return out; // a panel the page itself hides is not the user's to switch
  if (on) { it.off = true; it.hidden = true; rise(out.filter((i) => !i.hidden)); return out; }
  delete it.off; delete it.hidden;
  return moveItem(out, id, {}, cols);
}

// Whether two layouts put every panel on the same cells (the editor: is there anything to commit).
export function sameCells(a, b) {
  const key = (i) => [i.id, i.col, i.row, i.colSpan, i.rowSpan, !!i.collapsed, !!i.off].join();
  return a.length === b.length && a.map(key).sort().join('|') === b.map(key).sort().join('|');
}
