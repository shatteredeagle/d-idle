# SPDX-License-Identifier: GPL-3.0-or-later
"""COVMONSKILLS: join cases.tsv, game.tsv, d2d.tsv of one run; print per-row stats and sample values.
    python3 cms_show.py RUNDIR"""
import sys

d = sys.argv[1]
cs = [l.split() for l in open(d + "/cases.tsv")]
gs = [l.split() for l in open(d + "/game.tsv")]
ds = [l.split() for l in open(d + "/d2d.tsv")]
assert len(cs) == len(gs) == len(ds)
bad = sum(1 for g, x in zip(gs, ds) if g != x)
print("cases", len(cs), "mismatch", bad)
rows = {}
for c, g in zip(cs, gs):
    rows.setdefault(int(c[1]), []).append((c, g[1:]))
nonzero = 0
dist = {}
for s, lst in sorted(rows.items()):
    cells = set()
    for c, g in lst:
        for i, v in enumerate(g):
            if v not in ("0", "x"):
                cells.add(i)
    dist[s] = sorted(cells)
    if cells:
        nonzero += 1
print("rows with a nonzero calc cell:", nonzero, "of", len(rows))
names = "calc1 calc2 calc3 calc4 petmax sk1 sk2 sk3 sk4 sk5 delay range len perdelay a1 a2 a3 a4 a5 a6 p1 p2 p3 p4 p5 physmin physmax".split()
for s in sorted(dist):
    if dist[s]:
        lst = rows[s]
        # sample: level 10, clvl 30, no blizzard
        smp = [g for c, g in lst if c[2] == "10" and c[3] == "30" and c[8] == "0"]
        print(s, [names[i] for i in dist[s]], "L10:", [smp[0][i] for i in dist[s]] if smp else "-")
