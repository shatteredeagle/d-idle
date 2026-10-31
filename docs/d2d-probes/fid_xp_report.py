#!/usr/bin/env python3
"""XP per kill from fid_sweep.py stops. usage: fid_xp_report.py drops.jsonl...
Per clvl: least squares of stop xp over the monster types killed at that stop (ridge 1e-6), then compare with the original
level-difference table (clvl < 25): implied_base = xp / pen(mlvl - clvl); a stable implied_base over clvl 10..14 means d2d applies the table."""
import sys, json, collections, math
PEN = {10: 1.95, 9: 14.8, 8: 35.9, 7: 68.0, 6: 87.9, -6: 80.9, -7: 62.1, -8: 43.0, -9: 23.8}   # exact /256 values below
PEN256 = {10: 5, 9: 38, 8: 92, 7: 174, 6: 225, -6: 207, -7: 159, -8: 110, -9: 61}
def pen(mlvl, clvl):
    d = mlvl - clvl
    if d >= 10: return 5 / 256
    if d <= -10: return 13 / 256
    if -5 <= d <= 5: return 1.0
    return PEN256[d] / 256
rows = [json.loads(l) for fn in sys.argv[1:] for l in open(fn)]
per = collections.defaultdict(list)       # clvl -> list of (comp counter, dxp)
for r in rows:
    for s in r["stops"]:
        c = collections.Counter((n, l, b) for n, l, b in s["comp"]); per[r["clvl"]].append((c, s["dxp"]))
def solve(A, b):
    n = len(A)
    for i in range(n):
        p = max(range(i, n), key=lambda k: abs(A[k][i])); A[i], A[p] = A[p], A[i]; b[i], b[p] = b[p], b[i]
        for k in range(i + 1, n):
            f = A[k][i] / A[i][i]
            for j in range(i, n): A[k][j] -= f * A[i][j]
            b[k] -= f * b[i]
    x = [0.0] * n
    for i in reversed(range(n)): x[i] = (b[i] - sum(A[i][j] * x[j] for j in range(i + 1, n))) / A[i][i]
    return x
est = {}
for cl, st in sorted(per.items()):
    types = sorted({t for c, _ in st for t in c}); ix = {t: i for i, t in enumerate(types)}; n = len(types)
    AtA = [[0.0] * n for _ in range(n)]; Atb = [0.0] * n
    for c, y in st:
        for t1, k1 in c.items():
            Atb[ix[t1]] += k1 * y
            for t2, k2 in c.items(): AtA[ix[t1]][ix[t2]] += k1 * k2
    for i in range(n): AtA[i][i] += 1e-6
    x = solve(AtA, Atb); kills = sum(sum(c.values()) for c, _ in st)
    for t, v in zip(types, x): est[(cl, t)] = v
    print("clvl %d: %d stops, %d kills, total xp %d" % (cl, len(st), kills, sum(y for _, y in st)))
print("\n| monster (name, mlvl, kind) | " + " | ".join("clvl %d xp/kill (pen%%)" % c for c in sorted(per)) + " | implied base xp |")
allt = sorted({t for (_, t) in est}, key=lambda t: (t[1], t[0]))
for t in allt:
    cells, bases = [], []
    for c in sorted(per):
        v = est.get((c, t))
        if v is None: cells.append("-"); continue
        p = pen(t[1], c); cells.append("%.2f (%.0f%%)" % (v, 100 * p)); 
        if p >= 0.5 and v > 0.5: bases.append(v / p)
    print("| %s %d %s | %s | %s |" % (t[0], t[1], t[2], " | ".join(cells), ("%.2f..%.2f" % (min(bases), max(bases))) if bases else "-"))

AREA = {2: "Blood Moor", 3: "Cold Plains", 4: "Stony Field", 5: "Dark Wood", 8: "Den of Evil"}
print("\nAverage xp per kill by area and clvl (all monsters incl. uniques/champions; kills_to_level = span of clvl / avg):")
agg = collections.defaultdict(lambda: [0, 0])
for r in rows:
    for s in r["stops"]:
        a = agg[(r["level"], r["clvl"])]; a[0] += s["dxp"]; a[1] += len(s["comp"])
TAB = {1: 0, 2: 500, 3: 1500, 4: 3750, 5: 7875, 6: 14175, 7: 22680, 8: 32886, 9: 44396, 10: 57715, 11: 72144, 12: 90180, 13: 112725, 14: 140906, 15: 176132}
for (lvl, cl), (xp, k) in sorted(agg.items()):
    avg = xp / k; span = TAB[cl + 1] - TAB[cl]
    print("  %-12s clvl %d: %5d kills, %.2f xp/kill, kills to next level %d" % (AREA.get(lvl, lvl), cl, k, avg, span / avg if avg else 0))
