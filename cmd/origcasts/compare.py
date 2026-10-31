#!/usr/bin/env python3
"""our own helper. usage: compare.py orig-spawns.csv d2d-spawns.csv  -> markdown table (orig vs d2d) on stdout"""
import csv, sys, statistics as st, collections
AREA = {2: "Blood Moor", 3: "Cold Plains", 4: "Stony Field", 5: "Dark Wood", 8: "Den of Evil"}
def load(p):
    d = collections.defaultdict(list)
    for r in csv.reader(open(p)):
        if r and r[0].isdigit() and not r[-1].startswith('toarea'): d[int(r[0])].append(r)
    return d
o, d = load(sys.argv[1]), load(sys.argv[2])
def m(rs, i): return st.mean(int(r[i]) for r in rs)
def rate(rs, i): return sum(1 for r in rs if int(r[i]) > 0) / len(rs)
print("| Area | n orig / d2d | total mean orig | total mean d2d | orig min-max | d2d min-max | champion/visit orig | d2d | rare (unique col) per visit orig | d2d | superunique visits orig | d2d |")
print("|---|---|---|---|---|---|---|---|---|---|---|---|")
for a in [2, 3, 8, 4, 5]:
    if a not in o and a not in d: continue
    ro, rd = o.get(a, []), d.get(a, [])
    def g(rs, f):
        return f(rs) if rs else float("nan")
    t = lambda rs: [int(r[7]) for r in rs]
    print("| %s (%d) | %d / %d | %.1f | %.1f | %s | %s | %.2f | %.2f | %.2f | %.2f | %s | %s |" % (
        AREA[a], a, len(ro), len(rd), g(ro, lambda r: m(r, 7)), g(rd, lambda r: m(r, 7)),
        "%d-%d" % (min(t(ro)), max(t(ro))) if ro else "-", "%d-%d" % (min(t(rd)), max(t(rd))) if rd else "-",
        g(ro, lambda r: m(r, 4)), g(rd, lambda r: m(r, 4)), g(ro, lambda r: m(r, 5)), g(rd, lambda r: m(r, 5)),
        "%d/%d" % (sum(1 for r in ro if int(r[6]) > 0), len(ro)), "%d/%d" % (sum(1 for r in rd if int(r[6]) > 0), len(rd))))
