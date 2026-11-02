#!/usr/bin/env python3
# verifycovmissiles: the rows d2d loads vs the patched Missiles.txt, column by column; the ledger claim list.
#   vcm_loaded.py MISSILES_TSV LOADED_TSV LEDGER_TSV
import sys


def table(p):
    r = [l.rstrip("\n").split("\t") for l in open(p, encoding="latin-1") if "\t" in l]
    h = next(i for i, x in enumerate(r) if x[0] == "Missile")
    return [dict(zip(r[h], x)) for x in r[h + 1:] if x and x[0]]


def num(s):
    try:
        return int(s)
    except ValueError:
        return 0


T = table(sys.argv[1])
byname = {r["Missile"]: (i, r) for i, r in enumerate(T)}
cols = ["vel:Vel", "vel_lev:VelLev", "max_vel:MaxVel", "accel:Accel", "activate:Activate", "range:Range", "lev_range:LevRange",
        "size:Size", "collide_kill:CollideKill", "pierce:Pierce", "next_hit:NextHit", "next_delay:NextDelay", "to_hit:ToHit",
        "always_explode:AlwaysExplode"]
bad = {}
loaded = set()
unknown = []
for l in open(sys.argv[2]):
    f = l.rstrip("\n").split("\t")
    if len(f) < 21 and len(f) != 20:
        continue
    name = f[0]
    if name not in byname:
        unknown.append(name)
        continue
    i, r = byname[name]
    loaded.add(i)
    for k, c in enumerate(cols):
        key, col = c.split(":")
        v = int(f[1 + k])
        want = num(r[col])
        if key in ("collide_kill", "pierce", "next_hit", "to_hit", "always_explode"):
            want = int(want == 1)
        if v != want:
            bad.setdefault(key, []).append((i, name, v, want))
print(f"d2d loads {len(loaded)} of {len(T)} rows; names not in the table {len(unknown)}")
print("loader vs table, columns differing:", {k: len(v) for k, v in bad.items()})
for k, v in bad.items():
    print(" ", k, v[:6])
claimed = set()
for l in open(sys.argv[3]):
    p = l.rstrip("\n").split("\t")
    if p[0] == "missiles" and "covmissiles-trace" in l:
        for t in p[1].split(","):
            if "-" in t:
                a, b = t.split("-")
                claimed.update(range(int(a), int(b) + 1))
            else:
                claimed.add(int(t))
print("author ledger ids", len(claimed), "of which d2d does not load", sorted(claimed - loaded))
print("loaded by d2d and not in the author ledger lines:", sorted(loaded - claimed))
