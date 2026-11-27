// web/onboarding.js in the stub DOM (node test, card onboardtour): triggers from real state, Back/Skip/resume, seen kept in a store with a
// throwing fallback, the Now line from explain, the Help sheet.
'use strict';
const fs = require('fs');
const path = require('path');
const { makeDoc } = require('./nav_dom.cjs');
const O = require(path.join(process.argv[2], 'onboarding.cjs'));
const Sheet = require(path.join(process.argv[2], 'sheet.cjs'));
const T = JSON.parse(fs.readFileSync(path.join(process.argv[2], 'onboarding.json'), 'utf8'));
const fails = [];
const ok = (c, m) => { if (!c) fails.push(m); };
const mkStore = () => { const m = {}; return { getItem: (k) => (k in m ? m[k] : null), setItem: (k, v) => { m[k] = v; }, m }; };
const btns = (n) => n.all().filter((x) => x.tag === 'button');
const press = (n, key) => btns(n).find((b) => b.dataset.key === key).click();
const mk = (ls) => { const doc = makeDoc(); doc.N.prototype.remove = function () { if (this.parent) { this.parent.children = this.parent.children.filter((x) => x !== this); this.parent = null; } };
  const sheets = []; const tour = O.create(doc, { texts: T, store: O.store(ls), sheet: { open: (d, o) => { const r = Sheet.open(d, o); sheets.push(r); return r; } }, toast: () => {}, CustomEvent: function (t, i) { this.type = t; this.detail = i.detail; this.cancelable = i.cancelable; this.defaultPrevented = false; this.preventDefault = () => {}; } }); return { doc, tour, sheets }; };

// triggers: only real state raises a step
ok(JSON.stringify(O.triggered(null, T.townAreas)) === '["launch"]', 'first launch alone with no readings');
const all = O.triggered({ uptime: { deaths: 1 }, hero: { area: 'Rogue Encampment' }, vault: { total: 2 }, quests: { quests: [{ state: 'done' }] } }, T.townAreas);
ok(['launch', 'town', 'vault', 'quest', 'death'].every((x) => all.includes(x)), 'town, vault item, quest hand-in and death each trigger: ' + all);
ok(!O.triggered({ uptime: { deaths: 0 }, hero: { area: 'Blood Moor' }, vault: { total: 0 }, quests: { quests: [{ state: 'started' }] } }, T.townAreas).some((x) => x !== 'launch'), 'nothing else without the event');
ok(T.steps.map((s) => s.id).sort().join() === 'death,launch,quest,town,vault', 'five steps in the json');

// tour: one at a time, Got it persists, Back, Skip, resume
const ls = mkStore();
let { tour } = mk(ls);
tour.feed({ hero: { area: 'Harrogath' }, uptime: { deaths: 0 } });
ok(!tour.card.hidden && tour.state().cur === 'launch', 'the first launch tip shows first');
ok(btns(tour.card).every((b) => b.dataset.key !== 'tour-back'), 'no Back on the first tip');
press(tour.card, 'tour-next');
ok(tour.state().cur === 'town', 'Got it brings the next due tip');
ok(JSON.parse(ls.m[O.KEY]).seen.launch === 1, 'seen is written to localStorage');
press(tour.card, 'tour-back');
ok(tour.state().cur === 'launch', 'Back returns to the previous tip');
press(tour.card, 'tour-skip');
ok(tour.card.hidden && JSON.parse(ls.m[O.KEY]).off === true, 'Skip tour hides it and stores off');
tour.feed({ vault: { total: 1 } });
ok(tour.card.hidden, 'a skipped tour stays away when more events come');
// resume: a new page with the same store keeps what was seen; an unseen step comes back
const ls2 = mkStore();
({ tour } = mk(ls2)); tour.feed({ uptime: { deaths: 1 } }); press(tour.card, 'tour-next');
ok(tour.state().cur === 'death', 'launch seen, death next');
({ tour } = mk(ls2)); tour.feed({ uptime: { deaths: 1 } });
ok(tour.state().cur === 'death', 'after a reload the unseen step resumes, the seen one does not return');
// a dismissed step does not come back while still true
press(tour.card, 'tour-next');
tour.feed({ uptime: { deaths: 2 } });
ok(tour.card.hidden, 'a seen step stays gone');

// storage that throws: in-memory fallback, nothing breaks
const bad = { getItem() { throw new Error('denied'); }, setItem() { throw new Error('denied'); } };
({ tour } = mk(bad)); tour.feed({}); press(tour.card, 'tour-next');
ok(tour.card.hidden, 'with a throwing store Got it still works for this visit');
({ tour } = mk(null)); tour.feed({}); ok(!tour.card.hidden, 'no storage at all still shows the tip');
const corrupt = mkStore(); corrupt.m[O.KEY] = '{nope'; ({ tour } = mk(corrupt)); tour.feed({}); ok(!tour.card.hidden, 'a corrupt value reads as empty');

// the Now line
const ex = { enabled: true, explain: { decisions: [{ code: 'fight' }, { code: 'unknown_code' }], refusals: [{ what: 'buy potions', need: 'gold 500' }] } };
ok(O.nowLine({ enabled: false }, T.doing) === null, 'explain off: no line');
const nl = O.nowLine(ex, T.doing);
ok(nl.text === T.doing._default && nl.wait === 'gold 500', 'an unknown code uses the default sentence; the refusal names the need');
ok(O.nowLine({ enabled: true, explain: { decisions: [{ code: 'fight' }], refusals: [] } }, T.doing).text === 'Fighting', 'a known code has its sentence');
({ tour } = mk(mkStore())); ok(tour.now.hidden, 'the line is hidden until explain answers'); tour.paintNow(ex);
ok(!tour.now.hidden && tour.now.all().some((n) => n.textContent === T.doing._default), 'paintNow shows it'); tour.paintNow({ enabled: false }); ok(tour.now.hidden, 'and hides it again');

// Help sheet: lines, Replay, link to the guide
const H = mk(mkStore()); H.tour.feed({}); press(H.tour.card, 'tour-skip');
const sh = H.tour.openHelp();
ok(sh && H.doc.all().some((n) => n.textContent === T.help.title), 'the Help sheet opens with its title');
let guide = 0; H.doc.addEventListener('d2:nav', (e) => { if (e.detail.id === 'help') guide++; });
press(sh.sheet, 'help-guide'); ok(guide === 1, 'the guide button asks the shell to open the Help drawer (welcome card, first 30 minutes)');
const H2 = mk(mkStore()); H2.tour.feed({}); press(H2.tour.card, 'tour-skip'); const sh2 = H2.tour.openHelp(); press(sh2.sheet, 'help-replay');
ok(!H2.tour.card.hidden && H2.tour.state().cur === 'launch', 'Replay shows the tour again');
const H3 = mk(null); const sh3 = H3.tour.openHelp(); ok(H3.doc.all().some((n) => n.textContent === T.help.storeOff), 'without storage the sheet says tips may repeat');
console.log(JSON.stringify({ fails }));
