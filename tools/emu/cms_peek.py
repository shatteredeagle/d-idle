# SPDX-License-Identifier: GPL-3.0-or-later
"""COVMONSKILLS: print the game.tsv / d2d.tsv column files of a run side by side (rows only, d2d log lines dropped).
    python3 cms_peek.py RUNDIR"""
import re
import sys

d = sys.argv[1]
g = [l.split() for l in open(d + "/game.tsv")]
x = [l.split() for l in open(d + "/d2d.tsv") if re.match(r"^\d+ -?\d", l)]
print("game rows", len(g), "d2d rows", len(x), "differ", sum(1 for a, b in zip(g, x) if a != b))
print("id st do p1 p2 p3 mana lvlmana manashift minmana m0 m1 m2 m3")
for a in g:
    print(" ".join(a))
