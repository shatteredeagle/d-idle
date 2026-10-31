#!/usr/bin/env python3
"""Summarise fid_aggro.py output. woke = monster idle (NU) at first sight that moved > 0.3 cell or left mode NU within 1 s.
The distance columns are unreliable (the priming click walks the player ~1.8 cells), the monster displacement is not."""
import sys, json, collections
rows = [(r["level"], x) for fn in sys.argv[1:] for l in open(fn) for r in [json.loads(l)] for x in r["rows"]]
AREA = {2: "Blood Moor", 3: "Cold Plains", 4: "Stony Field", 8: "Den of Evil"}
def woke(x): return x["moved"] > 0.3 or x["mode"] != "NU"
B = [0, 3, 5, 7, 9, 11, 14, 18, 25, 99]
def bucket(d):
    for a, b in zip(B, B[1:]):
        if a <= d < b: return "%d-%d" % (a, b if b < 99 else 99)
for lvl in sorted({l for l, _ in rows}):
    print("\n==", AREA.get(lvl, lvl), "(cells; woke = moved or left NU within 25 ticks; last column = already walking at first sight)")
    t = collections.defaultdict(lambda: [0, 0, 0])
    for l, x in rows:
        if l != lvl or x["boss"] != "-" or x["mode0"] != "NU" and False: continue
        a = t[bucket(x["d0"])]; a[0] += 1; a[1] += woke(x); a[2] += (x["mode0"] != "NU")
    for k in sorted(t, key=lambda s: int(s.split("-")[0])):
        n, ap, mv = t[k]; print("  d0 %6s: n=%3d woke %3d (%3.0f%%), already walking %d" % (k, n, ap, 100 * ap / n, mv))
    print("  by name (approached/total, median d0 of approachers, max d0 of approachers):")
    nm = collections.defaultdict(list)
    for l, x in rows:
        if l == lvl and x["boss"] == "-": nm[x["name"]].append(x)
    for n, xs in sorted(nm.items()):
        ap = sorted(x["d0"] for x in xs if woke(x))
        non = sorted(x["d0"] for x in xs if not woke(x))
        print("   %-18s %3d  %3d woke; woke d0 max %s; never-woke d0 min %s" % (n, len(xs), len(ap), ap[-1] if ap else "-", non[0] if non else "-"))
