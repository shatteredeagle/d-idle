# SPDX-License-Identifier: GPL-3.0-or-later
"""SKDEFSHIELDD2D: game.exe's plain Attack damage (sk_melee2.py game, whole points = got >> 8, at least 1)
against d2d's REAL chain (skdd_test --shield), by what the shield carries.

    python -I skdd_shield_cmp.py d2dcases game.tsv real.tsv
"""
import sys


def main():
    cs = {}
    for line in open(sys.argv[1]):
        f = line.split()
        if len(f) == 16:
            cs[int(f[0])] = tuple(int(x) for x in f[1:])
    g = {int(l.split()[1]): l.split() for l in open(sys.argv[2]) if l.startswith("M ")}
    d = {int(l.split()[1]): l.split() for l in open(sys.argv[3]) if l.startswith("M ")}
    cat, n, miss = {}, 0, 0
    for cid, c in cs.items():
        if cid not in g or cid not in d:
            continue
        d0, d1, armed, bm, bM, ed, fmin, fmax, dp, strn, dex, sb, db, off, broken = c
        if d[cid][2] == "0":
            miss += 1                                     # d2d's to-hit draw missed: no damage to compare
            continue
        n += 1
        got, phys = max(int(g[cid][2]) >> 8, 1), int(d[cid][3])
        feats = [name for name, on in (("offED", off), ("flat", fmin or fmax), ("dmg%", dp)) if on]
        key = ("armed" if armed else "bare", "broken" if broken else "ok", "+".join(feats) or "none")
        k = cat.setdefault(key, [0, 0, 0])
        k[0] += 1
        if got != phys:
            k[1] += 1
            k[2] += got - phys
    print(f"cases {len(cs)}, d2d misses excluded {miss}, compared {n}")
    tot = sum(v[1] for v in cat.values())
    print(f"differ in whole points: {tot}")
    for key, v in sorted(cat.items()):
        print("  %-5s %-6s %-18s compared %5d differ %5d mean(game-d2d) %+.2f" % (*key, v[0], v[1], v[2] / max(v[1], 1)))


main()
