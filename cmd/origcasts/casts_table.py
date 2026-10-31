#!/usr/bin/env python3
"""our own helper. usage: casts_table.py orig-casts.csv d2d-casts.csv -> markdown on stdout (casts per kill by monster and Charged Bolt level)"""
import csv, sys, statistics as st, collections
def load(p, col):
    d = collections.defaultdict(list)
    for r in csv.DictReader(open(p)):
        if r["area"] != "2": continue
        d[(int(r["cb_level"]), r["monster"])].append(int(r[col]))
    return d
o, d = load(sys.argv[1], "cast_cmds"), load(sys.argv[2], "cast_cmds")
def p90(x): x = sorted(x); return x[min(len(x) - 1, int(0.9 * len(x)))]
def cell(x): return "%d kills: **%.2f** (median %g, p90 %d)" % (len(x), st.mean(x), st.median(x), p90(x)) if x else "-"
print("| Charged Bolt level | monster | original | d2d |\n|---|---|---|---|")
for mon in ["Fallen", "Quill Rat", "Zombie"]:
    for cb in [1, 4, 8]:
        print("| %d | %s | %s | %s |" % (cb, mon, cell(o.get((cb, mon), [])), cell(d.get((cb, mon), []))))
for cb in [1, 4, 8]:
    x = o.get((cb, "Fallen"), [])
    if x: print("zero-cast Fallen kills at CB %d: %.0f%%" % (cb, 100 * sum(1 for v in x if v == 0) / len(x)), file=sys.stderr)
