#!/usr/bin/env python3
# verifycovmissiles: compare game.tsv (game.exe, vcm_flight.py) with d2d.txt (vcm_test, the real rules header). usage: vcm_cmp.py DIR
import sys

d = sys.argv[1]
game = [l.split("\t") for l in open(f"{d}/game.tsv") if l.strip()]
made = [r for r in game if r[6] == "1"]
d2d = [l.split() for l in open(f"{d}/d2d.txt") if l.strip()]
assert len(made) == len(d2d), (len(made), len(d2d))
bad_v = bad_r = 0
ex = []
for r, x in zip(made, d2d):
    dv, dr = int(x[0]), int(x[1])
    gv, gr = int(r[7]), int(r[8])
    if dv != gv:
        bad_v += 1
        ex.append(("vel", r[:6], gv, dv))
    if dr != gr:
        bad_r += 1
        ex.append(("range", r[:6], gr, dr))
print(f"cases {len(game)} made {len(made)} not made {len(game) - len(made)}; d2d vs game: velocity differ {bad_v}, range differ {bad_r}")
mb = sum(1 for r in made if int(r[11]))
print(f"game vs own model from the loaded record: {mb} cases with any mismatch")
for e in ex[:12]:
    print("  ", *e)
for r in game:
    if r[6] == "0":
        print("   not made:", r[:6])
        break
