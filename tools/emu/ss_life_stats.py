#!/usr/bin/env python3
"""seedspawn: life of plain Normal monsters, game.exe (emulator) against d2d's make_monster on one world stream.

    python3 -I tools/emu/ss_life_stats.py life.txt

Input lines (ss_life_run.sh): `class statlvl game_life d2d_min d2d_max d2d_life`. Prints: units; game life outside d2d's
[min, max]; d2d life outside it; per (class, statlvl) group with >= 100 units: two-sample Kolmogorov-Smirnov D and its
asymptotic p (discrete data: conservative); the pooled normalised-life KS against uniform for each side and between them."""
import math
import sys
from collections import defaultdict


def ks2(a, b):
    a, b = sorted(a), sorted(b)
    i = j = 0
    d = 0.0
    while i < len(a) and j < len(b):
        x = min(a[i], b[j])
        while i < len(a) and a[i] <= x: i += 1
        while j < len(b) and b[j] <= x: j += 1
        d = max(d, abs(i / len(a) - j / len(b)))
    n = len(a) * len(b) / (len(a) + len(b))
    p = min(1.0, 2 * math.exp(-2 * n * d * d))
    return d, p


def ks_uniform(u):
    u = sorted(u)
    n = len(u)
    return max(max((i + 1) / n - x, x - i / n) for i, x in enumerate(u))


def main():
    rows = []
    other = 0
    for line in open(sys.argv[1]):
        f = line.split()
        if len(f) == 7 and all(v.lstrip("-").replace(".", "", 1).isdigit() for v in f[:6]) and all(c in "0123456789abcdef" for c in f[6]):
            if int(f[6], 16) & 0x1c == 0:                  # plain: not champion, unique or minion (ms_spawn_cmp.py's test)
                rows.append((int(f[0]), int(f[1]), float(f[2]), int(f[3]), int(f[4]), int(f[5])))
            else:
                other += 1
    print(f"units made {len(rows) + other}, plain {len(rows)} (flags & 0x1c == 0), others left out {other}")
    g_out = sum(not lo <= g <= hi for _, _, g, lo, hi, _ in rows)
    d_out = sum(not lo <= d <= hi for _, _, _, lo, hi, d in rows)
    print(f"game life outside d2d's [min,max]: {g_out}; d2d life outside: {d_out}")
    # units whose game life is outside d2d's range (another level or a life modifier: not traced, U) or whose range is a point
    # (the cow, 0..0) say nothing about the roll: left out of the distribution tests, counted above
    rows = [r for r in rows if r[4] > r[3] and r[3] <= r[2] <= r[4]]
    print(f"units in the distribution tests: {len(rows)}")
    groups = defaultdict(lambda: ([], [], []))
    for cls, lvl, g, lo, hi, d in rows:
        gr = groups[(cls, lvl, lo, hi)]
        gr[0].append(g); gr[1].append(d)
    ug, ud = [], []
    for (cls, lvl, lo, hi), (g, d, _) in groups.items():
        span = hi - lo + 1
        ug += [(x - lo + 0.5) / span for x in g]
        ud += [(x - lo + 0.5) / span for x in d]
    print(f"pooled normalised life: n game {len(ug)}, n d2d {len(ud)}")
    d, p = ks2(ug, ud)
    print(f"  two-sample KS game vs d2d: D {d:.4f}, p {p:.3f}")
    big = [(k, v) for k, v in groups.items() if len(v[0]) >= 100 and k[3] > k[2]]
    low = 0
    worst = []
    for (cls, lvl, lo, hi), (g, d, _) in sorted(big):
        dd, pp = ks2(g, d)
        low += pp < 0.01
        worst.append((pp, cls, lvl, lo, hi, len(g), sum(g) / len(g), sum(d) / len(d), dd))
    print(f"groups (class, level) with >= 100 units: {len(big)}; two-sample KS p < 0.01 in {low} (about {0.01 * len(big):.1f} expected by chance)")
    for pp, cls, lvl, lo, hi, n, mg, md, dd in sorted(worst)[:5]:
        print(f"  class {cls} lvl {lvl} life {lo}..{hi} n {n}: mean game {mg:.2f} d2d {md:.2f}, D {dd:.3f}, p {pp:.4f}")
    mean_g = sum(r[2] for r in rows) / len(rows)
    mean_d = sum(r[5] for r in rows) / len(rows)
    print(f"mean life over all units: game {mean_g:.3f}, d2d {mean_d:.3f} (ratio {mean_d / mean_g:.4f})")


main()
