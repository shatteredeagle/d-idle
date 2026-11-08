#!/usr/bin/env python3
"""stats2.py MINUTES_DIR: per class (tag prefix) over seeds: clvl/kills at virtual minute 20/45/60/90 as mean and range, minutes in
Recovery, town chores and deaths. Minute lines are one per virtual minute (extract.py of classpacing)."""
import json, sys, glob, os, statistics as st, collections
d = sys.argv[1]
runs = collections.defaultdict(dict)
for f in sorted(glob.glob(os.path.join(d, "*.jsonl"))):
    tag = os.path.basename(f)[:-6]; cls, seed = tag.rsplit("-s", 1)
    rows = [json.loads(l) for l in open(f)]
    if rows: runs[cls][int(seed)] = rows
def at(rows, m):
    i = m - 1
    if i >= len(rows) and m == 90 and len(rows) >= 82: i = len(rows) - 1  # v90 = last sample when the run stopped at 82..89
    return rows[i] if i < len(rows) else None
def fmt(vals):
    vals = [v for v in vals if v is not None]
    if not vals: return "-"
    return "%.1f (%s-%s)" % (st.mean(vals), min(vals), max(vals)) if isinstance(vals[0], int) else "-"
print("| class | seeds | n | v20 clvl / kills | v45 | v60 | v90 | recovery min | deaths |")
print("|---|---|---|---|---|---|---|---|---|")
for cls in sorted(runs):
    rs = dict(sorted(runs[cls].items())); cells = []
    for m in (20, 45, 60, 90):
        pts = [at(r, m) for r in rs.values()]
        pts = [p for p in pts if p]
        cells.append("%s / %s" % (fmt([p["clvl"] for p in pts]), fmt([p["kills"] for p in pts])))
    rec = [sum(1 for x in r[:90] if str(x.get("step") or "").startswith("Recovery")) for r in rs.values()]
    dead = [max(x.get("deaths") or 0 for x in r) for r in rs.values()]
    xp = [at(r, 90)["xp"] for r in rs.values() if at(r, 90)]
    print("| %s | %s | %s | %s | %s | %s |" % (cls, ",".join(map(str, sorted(rs))), ",".join(str(len(r)) for r in rs.values()), " | ".join(cells), fmt(rec), "%d" % sum(dead)) + " xp@v90 " + fmt(xp))

print()
print("per run (clvl/kills at 20, 45, 60, last; n minutes; recovery min)")
for cls in sorted(runs):
    for sd, r in sorted(runs[cls].items()):
        g = lambda m: (lambda p: "%s/%s/%s" % (p["clvl"], p["kills"], p["xp"]) if p else "-")(at(r, m))
        print(cls, sd, g(20), g(45), g(60), g(90), len(r), sum(1 for x in r[:90] if str(x.get("step") or "").startswith("Recovery")))
