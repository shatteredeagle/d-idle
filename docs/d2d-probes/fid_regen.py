#!/usr/bin/env python3
"""Check 1: natural mana/life/stamina regen, town vs field. usage: fid_regen.py OUTDIR
Sets maxmana via stat 9 (8.8 fixed: value*256), mana via stat 8, then steps and samples every 25 ticks."""
import sys, json, time, fid, math
out = sys.argv[1]; res = []
d = fid.start("Reg", 4, seed=3)
d.q("debug stat 12 12"); d.step(5)
def run(area, M, secs=60, life_max=0):
    d.q("debug stat 9 %d" % (M * 256)); d.q("debug stat 8 0"); d.q("debug stat 7 %d" % (100 * 256)); d.q("debug stat 6 %d" % (10 * 256)); d.step(2)
    d.q("debug stat 11 %d" % (100 * 256)); d.q("debug stat 10 %d" % (20 * 256))
    raw0 = json.loads(d.q("dump stats")[0])["stats"]["raw"]
    m0, l0, s0 = raw0[8], raw0[6], raw0[10]
    d.step(25 * secs)
    raw = json.loads(d.q("dump stats")[0])["stats"]["raw"]
    r = dict(area=area, maxmana=M, secs=secs, mana_gain=(raw[8] - m0) / 256, life_gain=(raw[6] - l0) / 256, stam_gain=(raw[10] - s0) / 256,
             mana_per_s=(raw[8] - m0) / 256 / secs, expected_floor=(math.floor(256 * M / (25 * 120)) * 25 / 256))
    res.append(r); print(r, flush=True)
for M in (35, 60, 100, 150, 250):
    run("town", M)
d.q("debug goto 2"); d.step(50); fid.prime(d); d.q("debug kill"); d.step(10)
for M in (35, 100, 250):
    run("bloodmoor", M)
fid.stop()
json.dump(res, open(out + "/regen.json", "w"), indent=1)
