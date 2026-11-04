"""cmp.py CASES GAME BASE HEAD: per build, cases that differ from game.exe, by class (agent palverify)."""
import sys
from collections import Counter

def load(p):
    return {int(l.split()[0]): tuple(int(x) for x in l.split()[1:]) for l in open(p) if l.strip()}


def strip(d):
    """d2d lines carry a 5th column, targetable; keep id -> (lo, hi, life) and the out-of-domain ids."""
    return {i: v[:3] for i, v in d.items()}, {i for i, v in d.items() if len(v) > 3 and v[3] == 0}


cases = {int(l.split()[0]): [int(x) for x in l.split()] for l in open(sys.argv[1])}
game = load(sys.argv[2])
base, out_b = strip(load(sys.argv[3]))
head, out_h = strip(load(sys.argv[4]))
assert out_b == out_h
rows_out = sorted({cases[i][6] for i in out_h})
for i in out_h:
    cases.pop(i)
print(f"out of domain (not targetable in d2d): {len(out_h)} cases, {len(rows_out)} MonStats rows")
flags = {}
import os
fl = os.path.join(os.path.dirname(os.path.abspath(sys.argv[2])), "..", "flags-game.txt")
for l in open(fl):
    _, m, u, d = l.split()
    flags[int(m)] = (int(u), int(d))


def cls(c):
    u, d = flags[c[6]]
    k = "undead" if u else "demon" if d else "plain"
    return k + ("/conc" if c[5] else "/noconc")


print(f"cases {len(cases)} game {len(game)} base {len(base)} head {len(head)}")
for name, out in (("base", base), ("head", head)):
    total, minmax, life, by = 0, 0, 0, Counter()
    first = []
    for i, c in cases.items():
        g, o = game.get(i), out.get(i)
        if g != o:
            total += 1
            by[cls(c)] += 1
            if g is None or o is None or g[:2] != o[:2]: minmax += 1
            elif g[2] != o[2]: life += 1
            if len(first) < 5: first.append((i, c[1:], g, o))
    allby = Counter(cls(c) for c in cases.values())
    print(f"{name}: differ {total} (min/max {minmax}, life only {life}); distinct monsters differing {len({cases[i][6] for i in cases if game.get(i) != out.get(i)})}")
    for k in sorted(allby): print(f"  {k}: {by[k]} of {allby[k]}")
    for f in first: print("  first:", f)
