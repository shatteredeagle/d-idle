#!/usr/bin/env python3
"""Vendor session in real time (the NPC menu click needs hover + a few frames): usage: fid_vend3.py NPC CMD...
CMD = 'tab X Y' | 'buy X Y' | 'sell X Y' | 'shot NAME' | 'hover X Y' | 'inv' ; prints gold after each. Coordinates are game pixels (800x600).
Opens the NPC's trade window first (hover, click, click)."""
import os, sys, math, fid
npc = sys.argv[1]; cmds = sys.argv[2:]
d = fid.start("Vn", 4, seed=3, lockstep=False); d.q("debug stat 12 12"); d.q("debug stat 14 20000"); d.step(25); fid.prime(d)
def find():
    for l in d.q("npcs"):
        f = l.split("\t")
        if f[0].startswith(npc): return int(f[4]), float(f[5]), float(f[6])
i, nx, ny = find()
d.q("cmd move %.1f %.1f" % (nx + 2.0, ny + 1.0)); d.step(250); fid.prime(d)
for _ in range(14):
    d.q("cmd interact %d" % i); d.step(100); px, py = d.player_xy()
    if d.q("menu") and math.hypot(px - nx, py - ny) < 3: break
m = d.q("menu"); print(m)
x, y = [l.split("\t") for l in m if l.lower().startswith(("trade", "repair"))][0][1:3]
d.q("move %s %s" % (x, y)); d.step(10); d.q("click %s %s" % (x, y)); d.step(10); d.q("click %s %s" % (x, y)); d.step(40)
print("store", d.state()["store"], "gold", d.state()["gold"])
def hc(x, y, n=1):
    d.q("move %s %s" % (x, y)); d.step(8)
    for _ in range(n): d.q("click %s %s" % (x, y)); d.step(12)
for c in cmds:
    p = c.split()
    g0 = int(d.state()["gold"])
    if p[0] == "tab": hc(p[1], p[2])
    elif p[0] == "buy": d.q("move %s %s" % (p[1], p[2])); d.step(8); d.q("rclick %s %s" % (p[1], p[2])); d.step(15)
    elif p[0] == "sell":
        d.q("move %s %s" % (p[1], p[2])); d.step(8); d.q("click %s %s" % (p[1], p[2])); d.step(12)
        d.q("move 250 330"); d.step(8); d.q("click 250 330"); d.step(15)
    elif p[0] == "hover": d.q("move %s %s" % (p[1], p[2])); d.step(15)
    elif p[0] == "shot": print(d.q("screenshot " + os.path.expanduser("~/d2d-lab/fidelity/out/%s.png" % p[1])))
    elif p[0] == "inv":
        for l in d.q("items"): print(l)
    elif p[0] == "dbg": print(d.q(c[4:]))
    elif p[0] == "wait": d.step(int(p[1]))
    print(c, "gold", g0, "->", d.state()["gold"], "held", d.state().get("held"), "store", d.state()["store"], flush=True)
fid.stop()
