# SPDX-License-Identifier: GPL-3.0-or-later
"""SKDEFSHIELDD2D: split the player's effres mismatches (v != 0) by cause; python skdd_ana.py cases game real"""
import sys

def cases(p):
    return {int(l.split()[0]): tuple(int(x) for x in l.split()[1:]) for l in open(p) if len(l.split()) == 12}

def table(p):
    return {int(l.split()[0]): l.split()[1:] for l in open(p) if l.split() and l.split()[0].isdigit() and len(l.split()) == 6}

cs, g, d = cases(sys.argv[1]), table(sys.argv[2]), table(sys.argv[3])
PEN = ((0, -40, -100), (0, -20, -50))
cat = {}
for cid, c in cs.items():
    if cid not in g or cid not in d:
        continue
    kind, diff, exp, elem, s, m, v, p, aux, roll, ln = c
    if kind != 0 or elem in (1,) or v == 0:
        continue
    gv, dv = g[cid][1], d[cid][1]
    ok = gv == dv
    pen = PEN[0 if exp else 1][diff] if elem >= 2 else 0
    cap = (min(75 + m, 95) if elem >= 2 else 50)
    panel = max(-100, min(cap, s + pen)) if elem >= 2 else min(s, 50)
    key = ("v>0" if v > 0 else "v<0", "elem" if elem >= 2 else "phys", "panel at floor" if panel <= -100 else "panel at cap" if panel >= cap else "panel inside", "equal" if ok else "DIFFER")
    cat[key] = cat.get(key, 0) + 1
    if not ok and cat[key] <= 2:
        print("  ex", key, c, "game", gv, "d2d", dv, "panel", panel)
for k in sorted(cat):
    print(k, cat[k])
# reachable values only: the player's physical resistance from items is never below 0 in 1.14d (I), an aura raises an
# elemental resistance (Salvation, Resist Fire / Cold / Lightning, v > 0), and the monster curses are -100 / -50 on 0x24
reach = {}
for cid, c in cs.items():
    if cid not in g or cid not in d:
        continue
    kind, diff, exp, elem, s, m, v, p, aux, roll, ln = c
    if kind != 0 or elem == 1 or v == 0:
        continue
    if (elem == 0 and v in (-100, -50) and s >= 0) or (elem >= 2 and 0 < v <= 150 and s >= 0):
        k = ("curse on 0x24" if elem == 0 else "aura on elem", "equal" if g[cid][1] == d[cid][1] else "DIFFER")
        reach[k] = reach.get(k, 0) + 1
print("reachable-value subset (raw >= 0, curse -100 / -50 on phys, aura > 0 on elements):", sorted(reach.items()))
