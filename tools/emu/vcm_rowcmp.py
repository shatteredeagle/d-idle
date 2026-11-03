# SPDX-License-Identifier: GPL-3.0-or-later
"""verifycovmissiles: the record game.exe holds in memory (read by vcm_flight.py's unperturbed grid rows) against the patched
Missiles.txt, Range / LevRange / Vel / VelLev / MaxVel / Accel of all 684 rows.  usage: vcm_rowcmp.py FLIGHT_DIR MISSILES_TSV"""
import os
import sys

d, tsv = sys.argv[1], sys.argv[2]


def num(s):
    try:
        return int(s)
    except ValueError:
        return 0


L = [l.rstrip("\n").split("\t") for l in open(tsv, encoding="latin-1") if "\t" in l]
h = next(i for i, x in enumerate(L) if x[0] == "Missile")
T = [dict(zip(L[h], x)) for x in L[h + 1:] if x and x[0]]
seen, bad = {}, {}
for line in open(f"{d}/game.tsv"):
    f = line.split("\t")
    mid, lvl = int(f[0]), int(f[1])
    if mid in seen or lvl != 0:
        continue
    seen[mid] = 1
    r = T[mid]
    got = {"Vel": int(f[2]), "VelLev": int(f[3]), "Range": int(f[4]), "LevRange": int(f[5]), "MaxVel": int(f[9]) >> 8, "Accel": int(f[10])}
    for c, v in got.items():
        if v != num(r[c]):
            bad.setdefault(c, []).append((mid, r["Missile"], v, r[c]))
print(f"rows read {len(seen)} of {len(T)}; disagreements with the table: { {c: len(v) for c, v in bad.items()} }")
for c, v in bad.items():
    print(" ", c, v[:5])
