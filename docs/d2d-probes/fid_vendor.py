#!/usr/bin/env python3
"""Check 5 helper: open a vendor's trade window in lockstep and dump what is readable. usage: fid_vendor.py NPC [OUTPNG]
Prints the NPC menu, `state store=`, the sell values the tooltips of our own items show with the window open, and takes a screenshot."""
import os, sys, re, fid
npc = sys.argv[1]; png = sys.argv[2] if len(sys.argv) > 2 else os.path.expanduser("~/d2d-lab/fidelity/out/vendor.png")
d = fid.start("Ven", 4, seed=3, lockstep=False); d.q("debug stat 12 12"); d.q("debug stat 14 20000"); d.step(10); fid.prime(d)
def npcs():
    r = []
    for l in d.q("npcs"):
        f = l.split("\t")
        if len(f) >= 7: r.append((f[0], int(f[4]), float(f[5]), float(f[6])))
    return r
print(npcs())
for _ in range(25):
    c = [n for n in npcs() if n[0].lower().startswith(npc.lower())]
    if not c: d.step(50); continue
    d.q("cmd interact %d" % c[0][1]); d.step(75)
    m = d.q("menu")
    if m: break
print("menu", m)
import math
for _ in range(12):
    n = [x for x in npcs() if x[0].lower().startswith(npc.lower())][0]; px, py = d.player_xy()
    print("dist to npc", round(math.hypot(px - n[2], py - n[3]), 1))
    if math.hypot(px - n[2], py - n[3]) < 3.5: break
    d.q("cmd interact %d" % n[1]); d.step(100)
d.step(30); m = d.q("menu")
d.step(200); m = d.q("menu"); print("menu again", m)
for l in m:
    p = l.split("\t")
    if p[0].lower().startswith("trade") or p[0].lower().startswith("repair"):
        d.q("move %s %s" % (p[1], p[2])); d.step(10); d.q("click %s %s" % (p[1], p[2])); d.step(10); d.q("click %s %s" % (p[1], p[2])); d.step(40); break
print("state store=", d.state().get("store"), "gold", d.state().get("gold"))
print(d.q("screenshot %s" % png))
for l in d.q("items"): print(l)
fid.stop()
