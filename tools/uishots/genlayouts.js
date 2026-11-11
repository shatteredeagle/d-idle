// Runs the mockup editor's own preset code (docs/mockups/ui/editor.html) and writes the three presets as the editor
// exports them, plus web/layout.json (the mockup preset with the real shell's extra panels placed).
const fs = require('fs'), vm = require('vm');
const src = fs.readFileSync('docs/mockups/ui/editor.html', 'utf8').split('\n');
const a = src.findIndex((l) => l.startsWith('var LIB = ')), b = src.findIndex((l) => l.startsWith('function load()'));
const code = src.slice(a, b).filter((l) => !/^var \$|^var canvas/.test(l)).join('\n');
const ctx = {}; vm.runInNewContext(code + '\nthis.x={blank,fromPreset,BPS,byId,PRESETS,LIB,place};', ctx);
const { blank, BPS, PRESETS, byId } = ctx.x;
// the editor's presets overlap in places (it resolves an overlap only on a drop); the shipped files carry the same
// push-down the editor and the shell apply, so tools/uilint's no-overlap rule holds
const occ = (ps, skip) => { const m = {}; ps.forEach((p) => { if (p === skip) return; for (let r = p.row; r < p.row + p.rowSpan; r++) for (let c = p.col; c < p.col + p.colSpan; c++) m[r + ':' + c] = 1; }); return m; };
const fromPreset = (name, bp) => {
  const out = [];
  for (const p of ctx.x.fromPreset(name, bp)) {
    for (let m = occ(out), ok = false; !ok; p.row++) {
      ok = true;
      for (let r = p.row; r < p.row + p.rowSpan && ok; r++) for (let c = p.col; c < p.col + p.colSpan; c++) if (m[r + ':' + c]) { ok = false; break; }
      if (ok) break;
    }
    out.push(p);
  }
  return out;
};
const out = 'internal/watch/web/';
fs.mkdirSync(out + 'layouts', { recursive: true });
// argv[3]: also write the raw exports (overlaps and all) there, as the editor's Export JSON gives them (compat fixtures)
if (process.argv[3]) for (const name of Object.keys(PRESETS)) {
  const o = blank(); Object.keys(BPS).forEach((bp) => { o.breakpoints[bp].panels = ctx.x.fromPreset(name, bp); });
  const f = process.argv[3] + '/layout-v1-' + name + '.json';
  if (!fs.existsSync(f)) fs.writeFileSync(f, JSON.stringify(o, null, 2) + '\n'); // fixtures are only added, never rewritten
}
for (const name of Object.keys(PRESETS)) {
  const o = blank(); Object.keys(BPS).forEach((bp) => { o.breakpoints[bp].panels = fromPreset(name, bp); });
  fs.writeFileSync(out + 'layouts/' + name + '.json', JSON.stringify(o, null, 2) + '\n');
}
// the default: the mockup preset, the real page's extra panels in (argv[2] = rows JSON overrides for iteration)
const rows = JSON.parse(fs.readFileSync(process.argv[2], 'utf8'));
for (const r of rows) if (!byId[r[0]]) byId[r[0]] = { id: r[0], band: r[5] || 'grid' };
PRESETS.real = rows.map((r) => r.slice(0, 5));
const o = blank(); Object.keys(BPS).forEach((bp) => { o.breakpoints[bp].panels = fromPreset('real', bp); });
// tablet: a half-width panel left alone on its row (the next one is full width) takes the whole row
{
  const ps = o.breakpoints.tablet.panels;
  let half = null;
  for (const p of ps) {
    if (p.colSpan === 12) { if (half) half.colSpan = 12; half = null; } else half = half ? null : p;
  }
  if (half) half.colSpan = 12;
  ps.forEach((p) => { p.col = 1; p.row = 1; });
  const done = [];
  for (const p of ps) { ctx.x.place(done, p); done.push(p); }
}
fs.writeFileSync(out + 'layout.json', JSON.stringify(o, null, 2) + '\n');
