# SPDX-License-Identifier: GPL-3.0-or-later
"""verifytele: classify game.exe oracle answers against the ROOM rule I read in
FUN_00554ea0 / FUN_00463740: accept needs tele != 0 and the target inside the caster's room
(40x40 subtiles, caster room (2,2) = [80,120)^2) or one of the 8 touching rooms ([40,160)^2).
Also reports: refusals inside the neighbourhood (snap found no spot), and a pure
distance rule (Chebyshev <= 40) as the competing hypothesis."""
import sys

cases = {}
for ln in open(sys.argv[1]):
    f = list(map(int, ln.split()))
    cases[f[0]] = f[1:6]
game = {}
for ln in open(sys.argv[2]):
    p = ln.split()
    game[int(p[0])] = int(p[1])
acc = ref = 0
tp = fp = fn = 0           # room rule vs game: pred accept / game
dist_bad = 0
tele0_acc = 0
inside_refused = 0
outside_accepted = 0
by = {}
for cid, (tele, cx, cy, tx, ty) in cases.items():
    g = game[cid]
    inroom = 40 <= tx < 160 and 40 <= ty < 160
    if tele == 0 and g:
        tele0_acc += 1
    if g:
        acc += 1
        if not inroom:
            outside_accepted += 1
    else:
        ref += 1
        if tele and inroom:
            inside_refused += 1
    dist = max(abs(tx - cx), abs(ty - cy)) <= 40
    if (dist and tele != 0) != bool(g) and not (tele and inroom and not g):
        dist_bad += 1
    k = (tele, inroom)
    a = by.setdefault(k, [0, 0])
    a[1 if g else 0] += 1
print("cases", len(cases), "accepted", acc, "refused", ref)
print("tele0 accepted (must be 0):", tele0_acc)
print("accepted with target outside 3x3 rooms (must be 0):", outside_accepted)
print("refused though tele!=0 and inside 3x3 rooms (snap found no spot):", inside_refused)
print("cases where distance<=40 hypothesis disagrees with game beyond snap fails:", dist_bad)
for k in sorted(by):
    print("tele", k[0], "inside3x3", k[1], "refused/accepted", by[k])
