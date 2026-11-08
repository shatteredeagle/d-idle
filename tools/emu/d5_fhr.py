"""decomp5: FHR breakpoints from the hit_len.py oracle output (game.exe) and from d2d, per AnimData get-hit record.
usage: python3 d5_fhr.py CASES GAME D2D  -- prints per (speed, frames) of kind 0 the FHR values where the hold (ticks) shortens."""
import sys
cases, game, d2d = sys.argv[1:4]
def load(path):
    out = {}
    for line in open(path):
        f = line.split()
        if f and not f[0].startswith('#'): out[int(f[0])] = tuple(map(int, f[1:]))
    return out
G, D = load(game), load(d2d)
rows = {}
for line in open(cases):
    f = line.split()
    if not f or f[0].startswith('#'): continue
    i, kind, speed, frames, fhr = map(int, f[:5])
    if kind == 0 and fhr <= 300 and i < 2 * 18 * 301: rows.setdefault((speed, frames), []).append((fhr, i))
for key, lst in sorted(rows.items(), key=lambda kv: (-kv[0][0], kv[0][1])):
    lst.sort()
    def bps(T):
        out, prev = [], None
        for fhr, i in lst:
            t = T[i][1]
            if prev is None or t < prev: out.append(fhr)
            prev = t
        return out
    g, d = bps(G), bps(D)
    t0 = G[lst[0][1]][1]
    print(f"speed {key[0]} frames {key[1]}: ticks at FHR 0 = {t0}; game.exe breakpoints {g}; d2d {'EQUAL' if g == d else d}")
