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

# per-frame view: frames where d2d hit two or more monsters (game.exe hands ONE unit per frame to FUN_005adf10)
def frames(path):
    out = {}
    for line in open(path):
        if line.startswith("H"):
            f = line.split()
            out[(f[0][1:], f[1])] = [tuple(x.split("x")) for x in f[2:]]
    return out
fg, fd = frames(sys.argv[1]), frames(sys.argv[2])
multi_d = sum(1 for v in fd.values() if len(v) > 1); multi_g = sum(1 for v in fg.values() if len(v) > 1)
print(f"frames with two or more monsters hit: game.exe {multi_g}, d2d {multi_d}")
rep = lambda fr: sum(1 for c in {k[0] for k in fr} if any(sum(1 for (cc, _), v in fr.items() if cc == c for m, h in v if m == mm) > 1 for mm in {m for (cc, _), v in fr.items() if cc == c for m, h in v}))
print(f"cases where a monster is hit in two different frames: game.exe {rep(fg)}, d2d {rep(fd)}")
