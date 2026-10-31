#!/usr/bin/env python3
"""Check 7: attack / cast frame counts. usage: fid_timing.py OUT.json
Sorceress spells: period between mana drops while the cast command is re-issued every tick (= frames per cast).
Melee (Barbarian / Paladin with several weapons): one `cmd skill 0` next to a monster, count ticks with pmode A1 (single swing);
repeat 6 times (hits can kill the target: a fresh monster is chosen each time).
Also logs `debug attack` (wsm, ias, fcr-like fields) so the weapon speed modifier d2d uses can be compared with Arreat's table."""
import sys, json, fid
res = []
def prep(d, cls):
    d.q("debug stat 12 12")
    for k, v in ((7, 900), (6, 900), (9, 300), (8, 300)): d.q("debug stat %d %d" % (k, v * 256))
    d.step(3); d.q("debug goto 2"); d.step(100); fid.prime(d)
def target(d, near):
    for y in range(8, 50, 10):
        for x in range(10, 90, 10):
            d.q("debug warp %d %d" % (x, y)); d.step(8); fid.prime(d)
            ms = [m for m in d.monsters() if m["mode"] in ("NU", "WL") and m["hp"].split("/")[0] != "0"]
            if ms:
                m = ms[0]
                if near: d.q("debug warp %.1f %.1f" % (m["x"] + 1.0, m["y"] + 0.5)); d.step(4); fid.prime(d)
                return m
    raise SystemExit("no monster")
d = fid.start("Tim", 4, seed=7); prep(d, 4)
for sk, name in ((36, "Fire Bolt"), (38, "Charged Bolt"), (39, "Ice Bolt?"), (45, "Ice Blast?"), (49, "Lightning?"), (53, "Chain Lightning?"), (47, "Fire Ball?"), (54, "Teleport?")):
    d.q("debug points %d 3" % sk); d.step(3); ok = d.q("debug skill left %d" % sk); m = target(d, False)
    d.q("debug stat 8 %d" % (300 * 256)); mana = []
    for t in range(150):
        d.q("cmd skill %d %.1f %.1f %d left" % (sk, m["x"], m["y"], m["unit"])); d.step(1); mana.append(int(d.state()["mana"].split("/")[0]))
    edges = [i for i in range(1, len(mana)) if mana[i] < mana[i - 1]]
    per = [b - a for a, b in zip(edges, edges[1:])]
    r = dict(who="sorc", skill=sk, name=name, ok=ok, mana_cost=[mana[i - 1] - mana[i] for i in edges[:4]], periods=per[:8], lskill=d.state()["lskill"]); res.append(r); print(r, flush=True)
    d.q("debug kill")
d.q("debug attack"); print(d.q("debug attack")[-1], flush=True)
fid.stop()
if len(sys.argv) > 2: raise SystemExit
for cls, cn in ((0, "barbarian"), (2, "paladin")):
    for code in ("ssd", "scm", "whm", "lsd", "spr"):
        d = fid.start("Tm%d" % cls, cls, seed=8); prep(d, cls)
        print(d.q("debug equip %s 4" % code)); att = d.q("debug attack")[-1]
        counts = []
        for rep in range(6):
            m = target(d, True)
            d.q("cmd skill 0 %.1f %.1f %d left" % (m["x"], m["y"], m["unit"])); run = 0; seen = False
            for t in range(60):
                d.step(1); pm = d.state()["pmode"]
                if pm.startswith("A"): run += 1; seen = True
                elif seen: break
            counts.append(run)
        r = dict(who=cn, weapon=code, a1_ticks=counts, attack=att); res.append(r); print(r, flush=True); fid.stop()
json.dump(res, open(sys.argv[1], "w"), indent=1)
