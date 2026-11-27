// Town plan card (town-planner section 8): the excursion and the verdict chip, the shopping list, the budget row, the
// stops and the one waiting sentence, from GET /api/townplan (watch.townplan/1). textContent only.

// stopsText is the stops in order with the ticked ones marked: "akara (done), stash".
export function stopsText(stops, done) {
  return (stops || []).map((s, i) => (i < done ? s + ' (done)' : s)).join(', ');
}

// lineText is one shopping row in words: "healing potions: have 3, need 4, buy 1 for 30 gold".
export function lineText(l) {
  let s = l.label + ': have ' + l.have + ', need ' + l.min;
  if (l.target > l.min) s += ' (up to ' + l.target + ')';
  if (l.buy > 0) s += ', buy ' + l.buy + ' for ' + l.cost + ' gold';
  if (l.status === 'dropped' || l.status === 'short') s += ', dropped';
  if (l.status === 'blocked') s += ', cannot be bought here';
  return s;
}

export function budgetText(b) {
  return 'Purse ' + b.purse + ', stash ' + b.stash + ', spending ' + b.spent + ', left ' + b.left + ' gold';
}

export function mount(root, ctx) {
  const head = ctx.el('p', 'tp-head');
  const list = ctx.el('ul', 'tp-lines');
  const budget = ctx.el('p', 'd-hint');
  const stops = ctx.el('p', 'd-hint');
  const wait = ctx.el('p', 'tp-wait');
  wait.setAttribute('role', 'status');
  ctx.body.append(head, list, budget, stops, wait);
  return {
    update(v) {
      if (!v || v.schema !== 'watch.townplan/1') { ctx.hide(true); return; }
      ctx.hide(false);
      head.textContent = v.known ? (v.for ? 'For ' + v.for + ': ' : '') + v.chip : 'No plan yet';
      head.dataset.verdict = v.verdict || '';
      ctx.ui.clear(list);
      for (const l of v.lines || []) {
        const li = ctx.el('li', 'tp-line', lineText(l));
        if (l.why) li.title = l.why;
        li.dataset.status = l.status;
        list.append(li);
      }
      budget.textContent = v.known ? budgetText(v.budget) : '';
      stops.textContent = (v.stops || []).length ? 'Stops: ' + stopsText(v.stops, v.stopsDone) : '';
      wait.textContent = v.waiting || '';
      ctx.state('ready');
    },
  };
}
