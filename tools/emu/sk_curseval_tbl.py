# SPDX-License-Identifier: GPL-3.0-or-later
"""SKCURSEVAL: print the game.exe values of the curse rows by level from a finished run
(sk_curseval_run.sh output: $D2_SCRATCH/curse/cases.tsv + game.tsv; grid cases only: clvl 30, masteries 0, no synergy).

    tools/emu/run-on-worker.sh sk_curseval_tbl.py > $D2_SCRATCH/tbl.txt

Columns per row and level: aurarange, auralen (frames), aurastatcalc1..6 (x = slot unused), calc1..3 (Iron Maiden / Life Tap)."""
import os

D = os.path.join(os.environ.get("D2_SCRATCH", "/tmp"), "curse")
LEVELS = (1, 2, 5, 10, 15, 20, 24, 25, 26, 30, 40, 60, 99)
cases = {}
with open(os.path.join(D, "cases.tsv")) as f:
    for line in f:
        t = line.split()
        cases[t[0]] = (int(t[1]), int(t[2]), int(t[3]), t[4:8], int(t[8]))
rows = {}
with open(os.path.join(D, "game.tsv")) as f:
    for line in f:
        t = line.split()
        s, lvl, clvl, m, n = cases[t[0]]
        if clvl == 30 and m == ["0"] * 4 and n == 0 and lvl in LEVELS:
            # t: id calc1..4 petmax sumsk1..5 delay range len perdelay a1..a6 p1..p5
            rows.setdefault(s, {})[lvl] = (t[12], t[13], t[15:21], t[1:4])
# synergy check: the grid cases (n = 0, or n = 11 = every curse row at 20 or 60) of one (row, level, clvl, mastery)
# must print the same cells if no calc reads another skill's level or a mastery.
groups = {}
with open(os.path.join(D, "game.tsv")) as f:
    for line in f:
        t = line.split()
        s, lvl, clvl, m, n = cases[t[0]]
        if n in (0, 11) and len(set(m)) == 1 and m[0] in ("0", "50"):
            groups.setdefault((s, lvl, clvl, m[0]), set()).add(" ".join(t[1:]))
varying = sorted({k[0] for k, v in groups.items() if len(v) > 1})
print("synergy check: %d grid groups, %d with differing cells; rows with any: %s" % (
    len(groups), sum(1 for v in groups.values() if len(v) > 1), varying))
print("row level range len aurastat1..6 calc1..3")
for s in sorted(rows):
    for lvl in LEVELS:
        r, ln, a, c = rows[s][lvl]
        print(s, lvl, r, ln, ",".join(a), ",".join(c))
