#!/usr/bin/env python3
"""Check 8: death gold/XP rules over a grid. usage: fid_death.py OUT.json
Per case: set clvl, purse, stash, exp (stat 12/14/15/13), `debug die`, dismiss the screen, read purse, ground pile, exp, corpses.
Expected (rule from mechanics-items-drops.md 9 / d2d notes): toll = min(clvl,20)% of (purse+stash), but the total never drops below
clvl*500, taken from the purse; the pile on the ground = purse - toll; stash untouched; Normal xp loss 0."""
import sys, json, fid
XP = {10: 57715, 11: 72144, 12: 90180, 13: 112725, 14: 140906, 20: 537513}
def expect(lvl, purse, stash):
    tot = purse + stash; toll = int(min(lvl, 20) * tot / 100); floor = lvl * 500
    if tot - toll < floor: toll = max(0, tot - floor)
    toll = min(toll, purse); return toll, purse - toll
d = fid.start("De", 4, seed=5); res = []
d.q("debug stat 7 %d" % (600 * 256)); d.q("debug stat 6 %d" % (600 * 256))
def die():
    d.q("debug die"); d.step(100); d.q("click 400 300"); d.step(10); d.q("key Escape"); d.step(60)
    if d.state()["gamemenu"].split(":")[0] != "0": d.q("click 400 308"); d.step(10)
for diff in (0, 1, 2):
    if diff: d.q("debug difficulty %d" % diff); d.step(200); fid.prime(d)
    for lvl in (10, 12, 14, 20):
        for purse, stash in ((0, 0), (400, 0), (1100, 0), (3000, 0), (6000, 0), (6000, 5000), (9000, 3000), (20000, 0), (50000, 0), (50000, 20000)):
            d.q("debug stat 12 %d" % lvl); d.q("debug stat 13 %d" % (XP[lvl] + 5000 if lvl < 20 else XP[lvl] + 20000)); d.q("debug stat 14 %d" % purse); d.q("debug stat 15 %d" % stash)
            d.q("debug stat 7 %d" % (600 * 256)); d.q("debug stat 6 %d" % (600 * 256)); d.step(3)
            e0 = int(d.state()["exp"]); die()
            ground = [l for l in d.q("ground") if l.startswith("gld")]; pile = sum(int(l.split("\t")[1].split()[0]) for l in ground)
            s = d.state(); toll, epile = expect(lvl, purse, stash)
            r = dict(diff=diff, clvl=lvl, purse=purse, stash=stash, purse_after=int(s["gold"]), pile=pile, expect_pile=epile, exp_lost=e0 - int(s["exp"]), exp0=e0,
                     corpses=len(d.q("debug corpses")))
            res.append(r); print(r, flush=True)
            d.q("debug clearinv")
fid.stop(); json.dump(res, open(sys.argv[1], "w"), indent=1)
