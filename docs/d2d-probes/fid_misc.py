#!/usr/bin/env python3
"""Does `debug stat 3 N` (vitality) take effect, and does it change the potion double-fill chance? (check 1b follow-up)"""
import json, fid
d = fid.start("Mi", 4, seed=2); d.q("debug stat 12 12")
print("before", json.loads(d.q("dump stats")[0])["stats"]["base"])
print(d.q("debug stat 3 150"), d.q("debug stat 1 150")); d.step(5)
st = json.loads(d.q("dump stats")[0])["stats"]; print("after", st["base"], st["raw"][:6], d.state()["life"], d.state()["mana"])
d.q("debug stat 7 %d" % (400 * 256)); d.q("debug stat 6 %d" % (50 * 256)); d.step(3)
print(d.q("debug attack")[-1][:80])
fid.stop()
