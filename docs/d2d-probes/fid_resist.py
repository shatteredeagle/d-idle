#!/usr/bin/env python3
"""Check 6: difficulty resist penalty via `debug attack` (res=f/l/c/p) and `dump stats` (res + res_cap), per class. usage: fid_resist.py OUT.json"""
import sys, json, fid
res = []
for cls in (4, 2):
    d = fid.start("Rs%d" % cls, cls, seed=9); d.q("debug stat 12 12")
    for k, v in ((7, 900), (6, 900)): d.q("debug stat %d %d" % (k, v * 256))
    d.step(3)
    for diff in (0, 1, 2, 0):
        if diff or res: d.q("debug difficulty %d" % diff); d.step(200)
        a = d.q("debug attack")[-1]; st = json.loads(d.q("dump stats")[0])["stats"]
        r = dict(cls=cls, difficulty=diff, attack=[t for t in a.split() if t.startswith(("res=", "dr=", "mdr=", "def="))], res=st["res"], cap=st["res_cap"],
                 state_diff=d.state()["difficulty"], area=d.q("debug level")[0])
        res.append(r); print(r, flush=True)
    fid.stop()
json.dump(res, open(sys.argv[1], "w"), indent=1)
