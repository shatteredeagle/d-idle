#!/usr/bin/env python3
"""Check 1b: potion heal curve. usage: fid_potion.py OUT.json CLASS_ID [VIT]
Fresh character per 4 potions (the start belt holds 4 Minor Healing Potions). Life is set low (stat 6), max 400, one potion is
drunk (`cmd belt 0`) and life is sampled every tick (raw stat 6 in 1/256) until it stops changing. Reports total, first-tick
delta, ticks to settle, and the double-fill flag (total ~ 2x base)."""
import sys, json, fid
out, cls = sys.argv[1], int(sys.argv[2]); vit = int(sys.argv[3]) if len(sys.argv) > 3 else None
res = []
def raw(d): return json.loads(d.q("dump stats")[0])["stats"]["raw"]
for rnd in range(6):
    d = fid.start("Po%d" % rnd, cls, seed=30 + rnd)
    d.q("debug stat 12 12")
    if vit is not None: d.q("debug stat 3 %d" % vit)
    d.q("debug stat 7 %d" % (400 * 256)); d.step(2)
    for k in range(4):
        d.q("debug stat 6 %d" % (50 * 256)); d.step(5)
        l0 = raw(d)[6]; d.q("cmd belt %d" % k); series = []
        for t in range(400):
            d.step(1); series.append(raw(d)[6])
        tot = (series[-1] - l0) / 256
        first = next((i for i, v in enumerate(series) if v != l0), None)
        last = max(i for i, v in enumerate(series) if v != series[-1]) + 1 if series[-1] != l0 else 0
        res.append(dict(total=tot, first_tick=first, last_tick=last, ticks=last - (first or 0), series=series[::10]))
        print(res[-1]["total"], first, last, flush=True)
    fid.stop()
json.dump(res, open(out, "w"))
