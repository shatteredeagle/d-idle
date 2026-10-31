#!/usr/bin/env python3
"""Check 2b: do hits wake monsters, pack rally, leash. usage: fid_hit.py OUT.jsonl SEED
A: sleeping monster 9..13 cells away (checked asleep for 25 ticks) is hit with Fire Bolt from there; 75 ticks later: target mode/moved and
   every monster within 7 cells of it.
B: player warped to 6 cells of a sleeping pack: per tick count of monsters awake (mode != NU) for 150 ticks (rally speed).
C: after B, player warped 30 cells away: modes and distance 100 ticks later (give-up / leash)."""
import sys, json, math, fid
out, seed = sys.argv[1], int(sys.argv[2]); res = []
d = fid.start("Hit%d" % seed, 4, seed=seed)
d.q("debug stat 12 12"); d.q("debug points 36 1"); d.step(3); d.q("debug skill left 36")
for k, v in ((7, 900), (6, 900), (9, 300), (8, 300)): d.q("debug stat %d %d" % (k, v * 256))
d.q("debug goto 2"); d.step(100); fid.prime(d); d.step(40)
def refill():
    for k, v in ((6, 900), (8, 300)): d.q("debug stat %d %d" % (k, v * 256))
def dist(a, b): return math.hypot(a[0] - b[0], a[1] - b[1])
trials = 0
for lvl in (2, 3, 4):
    if trials >= 8: break
    d.q('debug goto %d' % lvl); d.step(100); fid.prime(d); d.step(40)
    for y in range(8, 50, 5):
        for x in range(8, 90, 5):
            d.q("debug warp %d %d" % (x, y)); d.step(3); fid.prime(d); d.step(2); p = d.player_xy()
            ms = {m["unit"]: m for m in d.monsters() if m["mode"] == "NU"}
            d.step(25); ms2 = {m["unit"]: m for m in d.monsters()}
            asleep = [m for u, m in ms.items() if u in ms2 and ms2[u]["mode"] == "NU" and dist((m["x"], m["y"]), (ms2[u]["x"], ms2[u]["y"])) < 0.3 and 7.4 <= dist((m["x"], m["y"]), p) <= 9.6 and int(m["hp"].split("/")[1]) >= 6]
            if not asleep: d.q("debug kill"); d.step(10); refill(); continue
            t = asleep[0]; pp = d.player_xy()
            d.q("cmd skill 36 %.1f %.1f %d left" % (t["x"], t["y"], t["unit"])); d.step(75)
            now = {m["unit"]: m for m in d.monsters()}; tt = now[t["unit"]]
            near = [dict(name=m["name"], mode=m["mode"], moved=round(dist((m["x"], m["y"]), (ms2[u]["x"], ms2[u]["y"])), 2), hp=m["hp"]) for u, m in now.items()
                    if u in ms2 and u != t["unit"] and dist((ms2[u]["x"], ms2[u]["y"]), (t["x"], t["y"])) < 7]
            res.append(dict(part="A", target=t["name"], d=round(dist((t["x"], t["y"]), pp), 1), t_mode=tt["mode"], t_hp=tt["hp"], t_hp0=ms2[t["unit"]]["hp"],
                            t_moved=round(dist((tt["x"], tt["y"]), (t["x"], t["y"])), 2), near=near)); print(res[-1], flush=True)
            d.q("debug kill"); d.step(10); refill(); trials += 1
            if trials >= 8: break
        if trials >= 8: break

# part B and C: Dark Wood (first visit), a pack of >= 4 monsters asleep within 8 cells of each other
d.q("debug goto 5"); d.step(100); fid.prime(d); d.step(40)
best = None
for y in range(8, 70, 8):
    for x in range(8, 70, 8):
        d.q("debug warp %d %d" % (x, y)); d.step(3); fid.prime(d); d.step(2)
        ms = [m for m in d.monsters() if m["mode"] == "NU"]
        for t in ms:
            pack = [m for m in ms if dist((m["x"], m["y"]), (t["x"], t["y"])) < 8]
            if len(pack) >= 4 and dist((t["x"], t["y"]), d.player_xy()) > 9: best = (t, pack); break
        if best: break
        d.q("debug kill"); d.step(5)
    if best: break
if best:
    t, pack = best; pu = {m["unit"] for m in pack}
    d.q("debug warp %.1f %.1f" % (t["x"] + 6.5, t["y"])); d.step(1); refill(); series = []
    for tick in range(200):
        d.step(1)
        if tick % 10 == 9: series.append(sum(1 for m in d.monsters() if m["unit"] in pu and m["mode"] != "NU"))
    res.append(dict(part="B", n=len(pack), spread=[round(dist((m["x"], m["y"]), (t["x"], t["y"])), 1) for m in pack], awake_every_10_ticks=series)); print(res[-1], flush=True)
    d.q("debug warp %.1f %.1f" % (t["x"] + 30.0, t["y"])); fs = []
    for tick in range(300):
        d.step(1)
        if tick % 50 == 49:
            cur = [m for m in d.monsters() if m["unit"] in pu and m["mode"] not in ("DD", "DT")]; pxy = d.player_xy()
            fs.append(dict(t=tick + 1, awake=sum(1 for m in cur if m["mode"] != "NU"), mean_dist=round(sum(dist((m["x"], m["y"]), pxy) for m in cur) / max(1, len(cur)), 1)))
    res.append(dict(part="C", series=fs)); print(res[-1], flush=True)
fid.stop()
open(out, "a").write("\n".join(json.dumps(r) for r in res) + "\n")
