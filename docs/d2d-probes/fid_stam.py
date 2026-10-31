#!/usr/bin/env python3
"""Check 1c: stamina regen standing in town, and drain while running (Sorceress, clvl 12). Expected (mechanics-combat.md 7.2):
standing recovers max/256 per tick?? (doc wording), run drain 25*(40*(1+floor(armorSpeed/10))/256)/s = 3.91/s in light armor."""
import json, fid
d = fid.start("St", 4, seed=4); d.q("debug stat 12 12"); d.step(3)
raw = lambda: json.loads(d.q("dump stats")[0])["stats"]["raw"]
mx = raw()[11] / 256; print("max stamina", mx)
d.q("debug stat 10 0"); d.step(1); s0 = raw()[10]; ser = []
for t in range(10): d.step(25); ser.append(round((raw()[10]) / 256, 2))
print("standing, per second from 0:", ser)
# running: toggle run with key? use cmd move far (the character runs when run is on)
d.q("debug stat 10 %d" % int(mx * 256)); d.step(1); d.q("cmd move 45 30"); ser = []
for t in range(10): d.step(25); ser.append((round(raw()[10] / 256, 2), d.state()["pmode"], d.player_xy()))
print("moving:", ser)
fid.stop()
