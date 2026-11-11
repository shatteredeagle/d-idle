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
    if (p === skip) continue;
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
// page has no dead bands; with edit set (the editor) hidden targets keep their cells and nothing rises.
export function plan(layout, width, mode, conf, targets, edit) {
  const bpName = breakpointFor(width, mode);
  const bp = pickBreakpoint(layout, bpName);
  if (!bp) return null;
  const aliases = (conf && conf.aliases) || {}, reserved = (conf && conf.reserved) || [];
  const known = {};
  targets.forEach((t) => { known[t.id] = t; });
  const items = [], used = {}, unknown = [];
  let maxRow = 0, maxOrder = -1;
  for (const e of bp.panels) {
    const id = known[e.id] ? e.id : aliases[e.id];
    if (!id || !known[id]) { if (!reserved.includes(e.id)) unknown.push(e.id); continue; }
    if (used[id]) continue; // the first entry wins when an alias and its target are both named
    used[id] = true;
    if (known[id].hidden && !edit) continue;
    const colSpan = Math.max(1, Math.min(bp.cols, e.colSpan | 0)), col = Math.max(1, Math.min(bp.cols - colSpan + 1, e.col | 0));
    const it = { id, col, row: Math.max(1, e.row | 0), colSpan, rowSpan: Math.max(1, e.rowSpan | 0), collapsed: !!e.collapsed, order: e.order | 0, placed: true };
    placeNear(items, it); // a file with overlapping entries (the editor's presets have some) still loads: push down
    items.push(it);
    maxRow = Math.max(maxRow, it.row + (it.collapsed ? 1 : it.rowSpan) - 1);
    maxOrder = Math.max(maxOrder, it.order);
  }
  // panels the layout does not mention are packed after the placed ones, in page order
  const flow = bpName === 'phone' || bpName === 'tablet';
  for (const t of targets) {
    if (used[t.id] || (t.hidden && !edit)) continue;
    const [w, h] = SPAN[t.size] || SPAN.s;
    const it = { id: t.id, col: 1, row: maxRow + 1, colSpan: flow && bpName === 'phone' ? bp.cols : Math.min(bp.cols, flow ? Math.max(w, 6) : w), rowSpan: h, collapsed: false, order: ++maxOrder, placed: false };
    placeFrom(items, it, bp.cols, maxRow + 1);
    items.push(it);
  }
  if (!edit && !flow) { // each panel rises in its own columns into the room a hidden one left
    const done = [];
    for (const it of items.slice().sort((x, y) => x.row - y.row || x.col - y.col)) {
      const m = occ(done), h = it.collapsed ? 1 : it.rowSpan;
      while (it.row > 1 && free(m, it.col, it.row - 1, it.colSpan, h)) it.row--;
      done.push(it);
    }
  }
  return { name: bp.name, bp: bpName, cols: bp.cols, rowHeight: bpName === 'overlay' ? 40 : (layout.rowHeight || 48), flow, items, unknown };
}

// The entries of a plan as a breakpoint's panels array (what the editor saves), order from row then column.
export function toPanels(items, bands) {
  return items.slice().sort((a, b) => a.row - b.row || a.col - b.col).map((it, i) => ({
    id: it.id, band: (bands && bands[it.id]) || 'grid', col: it.col, row: it.row, colSpan: it.colSpan, rowSpan: it.rowSpan, collapsed: !!it.collapsed, order: i,
  }));
}
