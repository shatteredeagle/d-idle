import sys, math, fid
d = fid.start("Ven2", 4, seed=3, lockstep=False); d.q("debug stat 12 12"); d.q("debug stat 14 20000"); d.step(25); fid.prime(d)
def npc():
    for l in d.q("npcs"):
        f = l.split("\t")
        if f[0].startswith("Akara"): return int(f[4]), float(f[5]), float(f[6])
i, nx, ny = npc()
for _ in range(10):
    d.q("cmd interact %d" % i); d.step(100)
    px, py = d.player_xy()
    if d.q("menu") and math.hypot(px - nx, py - ny) < 3: break
m = d.q("menu"); print(m, d.state()["store"], d.player_xy(), (nx, ny))
x, y = [l.split("\t") for l in m if l.startswith("trade")][0][1:3]
for variant in range(4):
    if variant == 0: d.q("click %s %s" % (x, y)); d.step(40)
    elif variant == 1: d.q("move %s %s" % (x, y)); d.step(10); d.q("click %s %s" % (x, y)); d.step(10); d.q("click %s %s" % (x, y)); d.step(40)
    elif variant == 2: d.q("move %s %s" % (x, int(y) + 3)); d.step(10); d.q("click %s %s" % (x, int(y) + 3)); d.step(40)
    elif variant == 3: d.q("key Return"); d.step(40)
    print(variant, d.state()["store"], d.q("menu")[:1])
    if d.state()["store"] != "-1": break
fid.stop()
