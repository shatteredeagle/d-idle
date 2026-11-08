"""decomp6: summarise the Lightning pierce differential: per-case T lines (hits per monster) game.exe against d2d.
    sk6_light_sum.py game.txt d2d.txt
"""
import sys

def totals(path):
    out = {}
    for line in open(path):
        if line.startswith("T"):
            f = line.split()
            out[f[0][1:]] = [int(x) for x in f[1:]]
    return out

g, d = totals(sys.argv[1]), totals(sys.argv[2])
cases = sorted(set(g) & set(d))
bad = [c for c in cases if g[c] != d[c]]
gh = sum(sum(g[c]) for c in cases); dh = sum(sum(d[c]) for c in cases)
print(f"cases {len(cases)}, differ {len(bad)}; total hits game.exe {gh}, d2d {dh} ({(gh - dh) / max(dh, 1) * 100:+.1f} %)")
more = sum(1 for c in bad if sum(g[c]) > sum(d[c])); less = sum(1 for c in bad if sum(g[c]) < sum(d[c]))
print(f"  of the differing: game.exe more hits {more}, fewer {less}, same total other split {len(bad) - more - less}")
for c in bad[:8]:
    print(f"  {c}: game {g[c]} d2d {d[c]}")
