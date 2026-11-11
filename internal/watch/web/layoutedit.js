// Edit layout mode of /watch (docs/ui-design-system.md section 6.4; the mockup editor docs/mockups/ui/editor.html in
// the real page). Loaded by shell.js only when the user presses Edit layout. Drag a cell by its handle bar, resize by
// the gold corner, arrows move the focused cell, Shift+arrows resize, C collapses, Delete sends it to the end. The
// moves snap to the grid and push an overlapped cell down (the editor's packer, layout.js). Save posts the manifest
// to POST /api/layout; Reset to default drops the stored copy. Edits apply to the breakpoint of the window width.
import { occ, free, placeNear, toPanels } from '/watch/layout.js';

export function start(api) {
  const main = api.main, el = api.el;
  const p0 = api.plan();
  const bp = p0.bp, cols = p0.cols;
  const work = JSON.parse(JSON.stringify(api.layout));
  let items = p0.items.map((it) => Object.assign({}, it)); // every cell on the page, also the ones the file did not name
  const bands = {};
  ((api.layout.breakpoints[p0.name] || {}).panels || []).forEach((e) => { bands[e.id] = e.band; });
  const byId = (id) => items.find((it) => it.id === id);
  const added = [];
  let dirty = false, ghost = null;

  function commit() {
    work.breakpoints[bp] = { width: (work.breakpoints[bp] || {}).width || window.innerWidth, cols, panels: toPanels(items, bands) };
    items = api.preview(work).items.map((it) => Object.assign({}, it));
    labels();
  }
  function labels() {
    for (const it of items) {
      const n = api.targets.get(it.id), pos = n && n.querySelector(':scope > .d-edithandle .pos');
      if (pos) pos.textContent = 'c' + it.col + ' w' + it.colSpan + ' r' + it.row + ' h' + it.rowSpan;
      if (n) n.setAttribute('aria-label', it.id + ', column ' + it.col + ' span ' + it.colSpan + ', row ' + it.row + ' span ' + it.rowSpan);
    }
  }
  function change(it, fn) {
    fn(it);
    it.colSpan = Math.max(1, Math.min(cols, it.colSpan));
    it.col = Math.max(1, Math.min(cols - it.colSpan + 1, it.col));
    it.row = Math.max(1, it.row);
    it.rowSpan = Math.max(1, it.rowSpan);
    placeNear(items, it);
    dirty = true;
    commit();
  }

  // ---- pointer drag and resize, snapped to the grid ----
  function metrics() {
    const r = main.getBoundingClientRect(), cs = getComputedStyle(main);
    const gap = parseFloat(cs.columnGap) || 12, pad = parseFloat(cs.paddingLeft) || 0, rowh = p0.rowHeight;
    return { left: r.left + pad, top: r.top, gap, rowh, colw: (r.width - 2 * pad - gap * (cols - 1)) / cols };
  }
  const cellAt = (m, x, y) => ({ c: Math.floor((x - m.left + m.gap / 2) / (m.colw + m.gap)) + 1, r: Math.floor((y - m.top + m.gap / 2) / (m.rowh + m.gap)) + 1 });
  function drag(ev, it, n, mode) {
    if (ev.button !== 0) return;
    ev.preventDefault();
    n.focus({ preventScroll: true });
    const m = metrics(), from = cellAt(m, ev.clientX, ev.clientY), o = Object.assign({}, it), next = Object.assign({}, it);
    let moved = false;
    n.dataset.drag = '1';
    n.setPointerCapture(ev.pointerId);
    ghost = el('div', 'd-editghost');
    main.appendChild(ghost);
    const paint = () => {
      ghost.style.gridColumn = next.col + ' / span ' + next.colSpan;
      ghost.style.gridRow = next.row + ' / span ' + (next.collapsed ? 1 : next.rowSpan);
      if (free(occ(items, it), next.col, next.row, next.colSpan, next.collapsed ? 1 : next.rowSpan)) delete ghost.dataset.bad; else ghost.dataset.bad = '1';
    };
    const mv = (e) => {
      const c = cellAt(m, e.clientX, e.clientY), dc = c.c - from.c, dr = c.r - from.r;
      if (dc || dr) moved = true;
      if (mode === 'move') { next.col = Math.max(1, Math.min(cols - o.colSpan + 1, o.col + dc)); next.row = Math.max(1, o.row + dr); }
      else { next.colSpan = Math.max(1, Math.min(cols - o.col + 1, o.colSpan + dc)); next.rowSpan = Math.max(1, o.rowSpan + dr); }
      paint();
    };
    const up = (e) => {
      n.releasePointerCapture(e.pointerId);
      n.removeEventListener('pointermove', mv); n.removeEventListener('pointerup', up); n.removeEventListener('pointercancel', up);
      delete n.dataset.drag;
      if (ghost) { ghost.remove(); ghost = null; }
      if (moved) change(it, (t) => Object.assign(t, next));
    };
    n.addEventListener('pointermove', mv); n.addEventListener('pointerup', up); n.addEventListener('pointercancel', up);
    paint();
  }
  function keys(ev, it) {
    if (ev.target !== ev.currentTarget) return; // typing inside a panel is not a move
    const k = ev.key, d = { ArrowLeft: [-1, 0], ArrowRight: [1, 0], ArrowUp: [0, -1], ArrowDown: [0, 1] }[k];
    if (!d && k !== 'Delete' && k !== 'c' && k !== 'C') return;
    ev.preventDefault();
    change(it, (t) => {
      if (k === 'Delete') { t.col = 1; t.row = items.reduce((mx, x) => Math.max(mx, x.row + x.rowSpan), 1); }
      else if (!d) t.collapsed = !t.collapsed;
      else if (ev.shiftKey) { t.colSpan += d[0]; t.rowSpan += d[1]; }
      else { t.col += d[0]; t.row += d[1]; }
    });
  }

  // ---- the cells and the bar ----
  for (const it0 of items) {
    const n = api.targets.get(it0.id);
    if (!n) continue;
    const it = () => byId(it0.id);
    const handle = el('div', 'd-edithandle');
    handle.append(el('b', '', it0.id), el('span', 'pos'));
    const rs = el('i', 'd-editrs');
    rs.title = 'Resize';
    const had = n.getAttribute('tabindex'), label = n.getAttribute('aria-label');
    n.dataset.editcell = '1';
    n.tabIndex = 0;
    n.append(handle, rs);
    const onDown = (ev) => drag(ev, it(), n, 'move'), onRs = (ev) => { ev.stopPropagation(); drag(ev, it(), n, 'resize'); }, onKey = (ev) => keys(ev, it());
    handle.addEventListener('pointerdown', onDown);
    rs.addEventListener('pointerdown', onRs);
    n.addEventListener('keydown', onKey);
    added.push(() => {
      handle.remove(); rs.remove(); n.removeEventListener('keydown', onKey);
      delete n.dataset.editcell;
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
    el('span', 'keys', 'Drag a title bar, resize by the gold corner. Tab to a panel: arrows move, Shift+arrows resize, C collapses, Delete sends it to the end.'),
    save, reset, done);
  main.parentElement.insertBefore(bar, main);
  main.dataset.editing = '1';

  let kept = api.layout;
  function stop() {
    added.forEach((f) => f());
    bar.remove();
    delete main.dataset.editing;
    api.done(kept);
  }
  save.addEventListener('click', async () => {
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
  commit();
  dirty = false;
  return { stop };
}
