#!/usr/bin/env python3
"""Check 2: monster wake distance. usage: fid_aggro.py OUT.jsonl SEED LEVEL [LEVEL...]
Natural experiment on the sweep grid: at each warp stop, every monster seen for the first time is recorded with its distance d0 to the
player (cells), then 25 ticks (1 s) pass and we note whether it approached (distance shrank > 0.5 cell), moved at all, and its mode.
All monsters near the stop are then killed so later stops stay clean. The player has 600 life so deaths do not interfere.
Second part (hit test): in Blood Moor one asleep monster at distance 9..14 is hit with Fire Bolt (id 36) and watched."""
import sys, json, math, time, fid
out, seed = sys.argv[1], int(sys.argv[2]); levels = [int(x) for x in sys.argv[3:]]
d = fid.start("Ag%d" % seed, 4, seed=seed)
def setup():
    d.q("debug stat 12 12"); d.q("debug stat 13 90180")
    for k, v in ((7, 600), (6, 600), (9, 300), (8, 300)): d.q("debug stat %d %d" % (k, v * 256))
def goto(lvl):
    if lvl == 8:
        d.q("debug goto 2"); d.step(100); fid.prime(d)
        for l in d.q("debug warps"):
            p = l.split("\t")
            if len(p) >= 4 and p[3] == "8": d.q("debug enter %s" % p[0]); d.step(150); fid.prime(d); return
    d.q("debug goto %d" % lvl); d.step(100); fid.prime(d)
setup(); d.step(2); f = open(out, "a")
for lvl in levels:
    goto(lvl); info = d.q("debug level")[0].split(); w, h = int(info[4]), int(info[5]); seen = set(); rows = []
    for y in range(8, h - 6, 10):
        for x in range(8, w - 6, 10):
            if d.q("debug blocked %d %d" % (x, y))[-1].endswith("1"): continue
            d.q("debug warp %d %d" % (x, y)); d.step(3); fid.prime(d)
            px, py = d.player_xy(); new = [m for m in d.monsters() if m["unit"] not in seen and m["mode"] not in ("DD", "DT")]
            if not new: continue
            a = {m["unit"]: m for m in new}; seen |= set(a)
            d.step(25); px2, py2 = d.player_xy()
            for m in d.monsters():
                if m["unit"] in a:
                    m0 = a[m["unit"]]; d0 = math.hypot(m0["x"] - px, m0["y"] - py); d1 = math.hypot(m["x"] - px2, m["y"] - py2)
                    rows.append(dict(name=m["name"], lvl=m["lvl"], boss=m["boss"], d0=round(d0, 1), d1=round(d1, 1), moved=round(math.hypot(m["x"] - m0["x"], m["y"] - m0["y"]), 2),
                                     mode0=m0["mode"], mode=m["mode"], hp0=m0["hp"], hp=m["hp"]))
            d.q("debug kill"); d.step(10); setup()
    f.write(json.dumps(dict(level=lvl, seed=seed, rows=rows)) + "\n"); f.flush(); print("level", lvl, len(rows), flush=True)
fid.stop()
