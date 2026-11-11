// Edit layout mode of /watch (docs/ui-design-system.md sections 6.4 and 15.8; the mockup editor in the real page).
// Loaded by shell.js only when the user presses Edit layout. It works like a dashboard grid: while a panel is dragged
// (handle bar) or resized (gold corner) the page shows the result live, the panels it covers are pushed down, the push
// cascades and everything closes up (layout.js moveItem), with a short slide. Releasing the button on the grid commits
// that picture; Escape, a release outside the grid, or moving back to where nothing overlaps gives the layout from
// before the drag. The keyboard does the same: arrows move the focused panel, Shift+arrows resize, as a preview that
// Enter commits and Escape (or leaving the panel) drops; C collapses, Delete sends the panel to the end.
// A preview only sets grid positions on the page. The layout object changes on a commit, and the stored layout only
// on Save (POST /api/layout); Reset to default drops the stored copy. Edits apply to the breakpoint of the window.
import { moveItem, sameCells, toPanels } from '/watch/layout.js';

export function start(api) {
  const main = api.main, el = api.el;
  const p0 = api.plan();
  const bp = p0.bp, cols = p0.cols;
  const work = JSON.parse(JSON.stringify(api.layout));
  let base = p0.items.map((it) => Object.assign({}, it)); // the committed cells (hidden panels come along, flagged)
  let shown = base;                                       // what the page shows: base, or a preview
  let pending = null;                                     // a keyboard preview: { id, rect }
  let active = null;                                      // a pointer drag in progress
  const bands = {};
  ((api.layout.breakpoints[p0.name] || {}).panels || []).forEach((e) => { bands[e.id] = e.band; });
  const added = [];
  let dirty = false;
  const still = typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches;
  const cellOf = (id) => api.targets.get(id);
  const get = (items, id) => items.find((it) => it.id === id && !it.hidden);

  // ---- painting: grid positions only, with a slide from the old place (FLIP; none under reduced motion) ----
  function paint(items, skipFlip) {
    const cells = items.filter((it) => !it.hidden).map((it) => [it, cellOf(it.id)]).filter((c) => c[1]);
    const from = still || skipFlip === true ? null : new Map(cells.map(([, n]) => [n, n.getBoundingClientRect()]));
    for (const [it, n] of cells) {
      n.style.gridColumn = it.col + ' / span ' + it.colSpan;
      n.style.gridRow = it.row + ' / span ' + (it.collapsed ? 1 : it.rowSpan);
      if (it.collapsed) n.dataset.collapsed = '1'; else delete n.dataset.collapsed;
      const pos = n.querySelector(':scope > .d-edithandle .pos');
      if (pos) pos.textContent = 'c' + it.col + ' w' + it.colSpan + ' r' + it.row + ' h' + it.rowSpan;
      n.setAttribute('aria-label', it.id + ', column ' + it.col + ' span ' + it.colSpan + ', row ' + it.row + ' span ' + it.rowSpan);
    }
    shown = items;
    if (!from) return;
    const moved = [];
    for (const [it, n] of cells) {
      if (skipFlip === it.id) continue; // the panel under the pointer snaps; the ones it pushes slide
      const a = from.get(n), b = n.getBoundingClientRect(), dx = a.left - b.left, dy = a.top - b.top;
      if (!dx && !dy) continue;
      n.dataset.flip = 'from';
      n.style.transform = 'translate(' + dx + 'px, ' + dy + 'px)';
      moved.push(n);
    }
    if (!moved.length) return;
    void main.offsetWidth; // the start positions must be laid out before they are released, or nothing slides
    requestAnimationFrame(() => moved.forEach((n) => { n.dataset.flip = 'go'; n.style.transform = ''; }));
  }
  function preview(id, rect) {
    const next = moveItem(base, id, rect, cols);
    if (sameCells(next, base)) { delete main.dataset.preview; paint(base, id); return base; }
    main.dataset.preview = id;
    paint(next, id);
    return next;
  }
  function revert() {
    pending = null;
    delete main.dataset.preview;
    if (shown !== base) paint(base);
  }
  function commit(next) {
    pending = null;
    delete main.dataset.preview;
    if (sameCells(next, base)) { if (shown !== base) paint(base); return; }
    base = next;
    work.breakpoints[bp] = { width: (work.breakpoints[bp] || {}).width || window.innerWidth, cols, panels: toPanels(base, bands) };
    dirty = true;
    if (shown !== base) paint(base);
  }

  // ---- pointer drag and resize, snapped to the grid ----
  function metrics() {
    const r = main.getBoundingClientRect(), cs = getComputedStyle(main);
    const gap = parseFloat(cs.columnGap) || 12, pad = parseFloat(cs.paddingLeft) || 0, rowh = p0.rowHeight;
    return { rect: r, left: r.left + pad, top: r.top, gap, rowh, colw: (r.width - 2 * pad - gap * (cols - 1)) / cols };
  }
  const cellAt = (m, x, y) => ({ c: Math.floor((x - m.left + m.gap / 2) / (m.colw + m.gap)) + 1, r: Math.floor((y - m.top + m.gap / 2) / (m.rowh + m.gap)) + 1 });
  function drag(ev, id, n, mode) {
    if (ev.button !== 0 || active) return;
    const o = get(base, id);
    if (!o) return;
    ev.preventDefault();
    revert(); // a keyboard preview does not survive a grab
    n.focus({ preventScroll: true });
    const m = metrics(), from = cellAt(m, ev.clientX, ev.clientY);
    let next = base;
    n.dataset.drag = '1';
    n.setPointerCapture(ev.pointerId);
    const mv = (e) => {
      const c = cellAt(m, e.clientX, e.clientY), dc = c.c - from.c, dr = c.r - from.r;
      const rect = mode === 'move' ? { col: Math.max(1, Math.min(cols - o.colSpan + 1, o.col + dc)), row: Math.max(1, o.row + dr) }
        : { colSpan: Math.max(1, Math.min(cols - o.col + 1, o.colSpan + dc)), rowSpan: Math.max(1, o.rowSpan + dr) };
      next = preview(id, rect);
    };
    const end = (keep) => {
      n.releasePointerCapture(active.pointer);
      n.removeEventListener('pointermove', mv); n.removeEventListener('pointerup', up); n.removeEventListener('pointercancel', cancel);
      delete n.dataset.drag;
      active = null;
      if (keep) commit(next); else revert();
    };
    // a release outside the grid is not a drop: the layout from before the drag comes back
    const up = (e) => { const q = main.getBoundingClientRect(); end(e.clientX >= q.left && e.clientX <= q.right && e.clientY >= q.top && e.clientY <= q.bottom); };
    const cancel = () => end(false);
    active = { pointer: ev.pointerId, cancel };
    n.addEventListener('pointermove', mv); n.addEventListener('pointerup', up); n.addEventListener('pointercancel', cancel);
  }

  // ---- keyboard: a preview that Enter commits and Escape drops ----
  function keys(ev, id) {
    if (ev.target !== ev.currentTarget) return; // typing inside a panel is not a move
    const k = ev.key;
    if (k === 'Escape') { if (active) { ev.preventDefault(); active.cancel(); } else if (pending) { ev.preventDefault(); revert(); } return; }
    if (active) return;
    if (k === 'Enter') { if (pending) { ev.preventDefault(); commit(shown); } return; }
    const d = { ArrowLeft: [-1, 0], ArrowRight: [1, 0], ArrowUp: [0, -1], ArrowDown: [0, 1] }[k];
    const o = get(base, id);
    if (!o || (!d && k !== 'Delete' && k !== 'c' && k !== 'C')) return;
    ev.preventDefault();
    if (!d) { // collapse and send-to-end are one step: no preview to confirm
      revert();
      const last = base.filter((x) => !x.hidden).reduce((mx, x) => Math.max(mx, x.row + x.rowSpan), 1);
      commit(moveItem(base, id, k === 'Delete' ? { col: 1, row: last } : { collapsed: !o.collapsed }, cols));
      return;
    }
    const r = pending && pending.id === id ? pending.rect : { col: o.col, row: o.row, colSpan: o.colSpan, rowSpan: o.rowSpan };
    if (ev.shiftKey) { r.colSpan = Math.max(1, Math.min(cols - r.col + 1, r.colSpan + d[0])); r.rowSpan = Math.max(1, r.rowSpan + d[1]); }
    else { r.col = Math.max(1, Math.min(cols - r.colSpan + 1, r.col + d[0])); r.row = Math.max(1, r.row + d[1]); }
    pending = { id, rect: r };
    preview(id, r);
  }

  // ---- the cells and the bar ----
  for (const it0 of base) {
    const n = cellOf(it0.id);
    if (!n || it0.hidden) continue;
    const handle = el('div', 'd-edithandle');
    handle.append(el('b', '', it0.id), el('span', 'pos'));
    const rs = el('i', 'd-editrs');
    rs.title = 'Resize';
    const had = n.getAttribute('tabindex'), label = n.getAttribute('aria-label');
    n.dataset.editcell = '1';
    n.tabIndex = 0;
    n.append(handle, rs);
    const onDown = (ev) => drag(ev, it0.id, n, 'move'), onRs = (ev) => { ev.stopPropagation(); drag(ev, it0.id, n, 'resize'); };
    const onKey = (ev) => keys(ev, it0.id), onBlur = () => { if (pending && pending.id === it0.id && !active) revert(); };
    const onFlip = () => { if (n.dataset.flip === 'go') delete n.dataset.flip; };
    handle.addEventListener('pointerdown', onDown);
    rs.addEventListener('pointerdown', onRs);
    n.addEventListener('keydown', onKey);
    n.addEventListener('blur', onBlur);
    n.addEventListener('transitionend', onFlip);
    added.push(() => {
      handle.remove(); rs.remove();
      n.removeEventListener('keydown', onKey); n.removeEventListener('blur', onBlur); n.removeEventListener('transitionend', onFlip);
      delete n.dataset.editcell; delete n.dataset.flip; delete n.dataset.drag;
      n.style.transform = '';
      if (had === null) n.removeAttribute('tabindex'); else n.setAttribute('tabindex', had);
      if (label === null) n.removeAttribute('aria-label'); else n.setAttribute('aria-label', label);
    });
  }
  const bar = el('div', 'd-editbar');
  bar.setAttribute('role', 'toolbar');
  bar.setAttribute('aria-label', 'Layout editor');
  const mk = (text, variant) => { const b = el('button', 'd-btn', text); b.type = 'button'; b.dataset.size = 'sm'; if (variant) b.dataset.variant = variant; return b; };
  const save = mk('Save', 'primary'), reset = mk('Reset to default'), done = mk('Done', 'ghost');
  bar.append(el('span', 'what', 'Editing the ' + bp + ' layout'),
    el('span', 'keys', 'Drag a title bar or the gold corner: the other panels make room, Escape puts everything back. Keys on a focused panel: arrows move, Shift+arrows resize, Enter keeps, Escape drops, C collapses, Delete sends it to the end.'),
    save, reset, done);
  main.parentElement.insertBefore(bar, main);
  main.dataset.editing = '1';

  let kept = api.layout;
  function stop() {
    if (active) active.cancel();
    revert();
    added.forEach((f) => f());
    bar.remove();
    delete main.dataset.editing;
    delete main.dataset.preview;
    api.done(kept);
  }
  save.addEventListener('click', async () => {
    revert();
    const r = await api.save(work);
    if (r && r.ok) { kept = r.layout || work; dirty = false; api.toast('good', 'Layout saved.'); }
    else if (r) api.toast('alarm', 'Not saved: ' + (r.error || 'the answer route refused it'));
  });
  reset.addEventListener('click', async () => {
    const r = await api.reset();
    if (!r || !r.ok || !r.layout) { if (r) api.toast('alarm', 'Not reset: ' + (r.error || 'the answer route refused it')); return; }
    kept = r.layout;
    api.toast('good', 'Layout reset to the default.');
    stop();
  });
  done.addEventListener('click', () => {
    if (dirty) api.toast('info', 'Unsaved layout changes were dropped.');
    stop();
  });
  paint(base, true); // the cells are where the shell put them: nothing to slide
  // ?drag=<id>:<col>:<row> shows a drag in progress (a link for a bug report, and the screenshot suite); nothing is committed
  const demo = /^([a-z0-9-]+):(\d+):(\d+)$/.exec(api.demo || '');
  if (demo && get(base, demo[1])) { pending = { id: demo[1], rect: { col: +demo[2], row: +demo[3] } }; preview(demo[1], pending.rect); }
  return { stop };
}
