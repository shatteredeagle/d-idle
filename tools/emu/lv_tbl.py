# SPDX-License-Identifier: GPL-3.0-or-later
"""Levels.txt (patched 1.14d stack) sanity census of the columns d2d consumes (agent levelsareas).

    python3 lv_tbl.py levels.txt     # file from tools/emu/mpq-table.sh levels

Prints counts and the odd rows: area level per difficulty (MonLvl1Ex..3Ex, classic MonLvl1), density cap 10000
(d2d objgroups.cpp:716), umin > umax, density with no monster slot, wander, NumMon.
"""
import sys

rows = [l.rstrip("\n").split("\t") for l in open(sys.argv[1], encoding="latin-1")]
start = next(i for i, r in enumerate(rows) if r[:2] == ["Name", "Id"])   # skip wrapper banner lines
rows = rows[start:]
hdr = rows[0]
ix = {h: i for i, h in enumerate(hdr)}


def g(r, k):
    v = r[ix[k]] if ix[k] < len(r) else ""
    return int(v) if v.lstrip("-").isdigit() else 0


data = [r for r in rows[1:] if len(r) > 2 and r[ix["Id"]].isdigit()]
print("rows", len(data), "ids", data[0][ix["Id"]], "..", data[-1][ix["Id"]])
pop = [r for r in data if g(r, "MonDen") or g(r, "MonDen(N)") or g(r, "MonDen(H)")]
print("rows with density >0 in any difficulty:", len(pop))
for r in pop:
    i = int(r[ix["Id"]])
    n = [g(r, f"MonLvl{d}Ex") for d in (1, 2, 3)]
    den = [g(r, "MonDen"), g(r, "MonDen(N)"), g(r, "MonDen(H)")]
    um = [(g(r, "MonUMin"), g(r, "MonUMax")), (g(r, "MonUMin(N)"), g(r, "MonUMax(N)")), (g(r, "MonUMin(H)"), g(r, "MonUMax(H)"))]
    slots = sum(1 for k in range(1, 11) if ix[f"mon{k}"] < len(r) and r[ix[f"mon{k}"]])
    flags = []
    if not (n[0] <= n[1] <= n[2]): flags.append("lvl-not-monotone")
    if any(d > 10000 for d in den): flags.append("density>cap")
    if any(a > b for a, b in um): flags.append("umin>umax")
    if slots == 0: flags.append("no-mon-slot")
    if g(r, "MonLvl1Ex") != g(r, "MonLvl1"): flags.append("ex!=classic")
    print(i, r[ix["Name"]], "lvl", n, "classic", g(r, "MonLvl1"), "den", den, "u", um, "wndr", g(r, "MonWndr"), "num", g(r, "NumMon"), "slots", slots, *flags)
zero = [r[ix["Id"]] for r in data if g(r, "MonLvl1Ex") == 0 and g(r, "MonDen") > 0]
print("populated Normal rows with MonLvl1Ex 0:", zero)
