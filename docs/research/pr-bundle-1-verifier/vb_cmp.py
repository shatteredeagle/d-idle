# vb_cmp.py GAME D2D : compare hit sets per (class, offset); prints per class: game hits, d2d hits, only-game, only-d2d
import sys
def load(p):
    d = {}
    for ln in open(p):
        f = ln.split()
        if len(f) == 6 and f[0] == "V" and f[5].lstrip("-").isdigit(): d[(int(f[1]), int(f[3]), int(f[4]))] = (int(f[2]), int(f[5]))
    return d
g = {}
for p in sys.argv[1].split(","): g.update(load(p))
d = load(sys.argv[2])
tot = [0, 0, 0]
for cls in sorted({k[0] for k in d}):
    keys = [k for k in d if k[0] == cls]
    if not any(k in g for k in keys): print(f"cls {cls} size {d[keys[0]][0]} NOGAME d2d {sum(1 for k in keys if d[k][1] > 0)}"); continue
    gh = {k for k in keys if k in g and g[k][1] > 0}; dh = {k for k in keys if d[k][1] > 0}
    tot[0] += len(keys); tot[1] += len(gh - dh); tot[2] += len(dh - gh)
    print(f"cls {cls} size {d[keys[0]][0]}/{g[keys[0]][0] if keys[0] in g else '?'} game {len(gh)} d2d {len(dh)} only_game {len(gh - dh)} only_d2d {len(dh - gh)}")
print(f"TOTAL cases {tot[0]} differ {tot[1] + tot[2]}")
