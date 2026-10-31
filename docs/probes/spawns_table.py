#!/usr/bin/env python3
"""our own helper. usage: spawns_table.py d2dspawn.out > d2d-spawns.csv ; also prints a markdown summary to stderr.
Columns (the original-game side fills the same): area,seed,normal,minion,champion,unique,superunique,total,names,boss_names
normal = pack-less or pack members without the minion flag; minion = followers of a champion/unique; unique = d2d's boss column
"unique" (the game's rare monsters); superunique = Bishibosh, Corpsefire etc."""
import csv, sys, statistics as st, collections
HDR = "area,seed,normal,minion,champion,unique,superunique,total,names,boss_names"
AREA = {2: "Blood Moor", 3: "Cold Plains", 4: "Stony Field", 5: "Dark Wood", 8: "Den of Evil"}
rows = [r for r in csv.reader(open(sys.argv[1])) if r and r[0].isdigit()]
print(HDR)
for r in rows: print(",".join(r[:8]) + "," + ",".join('"%s"' % x for x in r[8:]))
by = collections.defaultdict(list)
for r in rows: by[int(r[0])].append(r)
w = sys.stderr.write
w("| Area | seeds | total mean (min-max) | normal mean | minion | champion | unique (rare) | superunique | top names (mean per seed) | superuniques seen |\n|---|---|---|---|---|---|---|---|---|---|\n")
for a, rs in sorted(by.items()):
    n = len(rs); col = lambda i: [int(r[i]) for r in rs]
    names = collections.Counter(); sup = collections.Counter()
    for r in rs:
        for t in filter(None, r[8].split(";")):
            nm, _, k = t.rpartition(" x"); names[nm] += int(k)
        for t in filter(None, r[9].split(";")):
            if t.startswith("superunique:"): sup[t.split(":", 1)[1]] += 1
    top = ", ".join("%s %.1f" % (k, v / n) for k, v in names.most_common(5))
    sp = ", ".join("%s %d/%d" % kv + "" for kv in [(k, v, n) for k, v in sup.items()]) or "-"
    f = lambda i: "%.1f" % st.mean(col(i))
    t = col(7)
    w("| %s (%d) | %d | %.1f (%d-%d) | %s | %s | %s | %s | %s | %s | %s |\n" % (AREA.get(a, a), a, n, st.mean(t), min(t), max(t), f(2), f(3), f(4), f(5), f(6), top, sp))
