#!/usr/bin/env python3
"""Checks 3+4: drops and XP per kill over a full sweep of levels, lockstep.
usage: fid_sweep.py OUT.jsonl SEED CLVL CLASS LEVEL [LEVEL...]    (LEVEL = Levels.txt id; 8 = Den of Evil via Blood Moor)
For every level: warp over a 10-cell grid, wait, `debug kill` at each stop, tally new monsters (by unit id), the xp of each stop
(character level and exp are reset before every kill so the clvl stays fixed), ground gold piles (amount from the label) and
ground items; every non-gold item is picked up and dumped to read quality/ilvl/code, then the inventory is cleared.
One JSON line per level: monsters, stops (comp + dxp), gold piles, items."""
import sys, json, time, fid
XP = {1: 0, 2: 500, 3: 1500, 4: 3750, 5: 7875, 6: 14175, 7: 22680, 8: 32886, 9: 44396, 10: 57715, 11: 72144, 12: 90180, 13: 112725, 14: 140906}
out, seed, clvl, cls = sys.argv[1], int(sys.argv[2]), int(sys.argv[3]), int(sys.argv[4]); levels = [int(x) for x in sys.argv[5:]]
d = fid.start("Sw%d" % seed, cls, seed=seed)
def setup():
    d.q("debug stat 12 %d" % clvl); d.q("debug stat 13 %d" % XP[clvl])
    for k, v in ((7, 600), (6, 600), (9, 300), (8, 300)): d.q("debug stat %d %d" % (k, v * 256))
def goto(lvl):
    if lvl == 8:
        d.q("debug goto 2"); d.step(100); fid.prime(d)
        for l in d.q("debug warps"):
            p = l.split("\t")
            if len(p) >= 4 and p[3] == "8": d.q("debug enter %s" % p[0]); d.step(150); fid.prime(d); return
        raise SystemExit("no den warp")
    d.q("debug goto %d" % lvl); d.step(100); fid.prime(d)
def num_exp(): return int(d.state()["exp"])
setup(); d.step(5)
f = open(out, "a")
for lvl in levels:
    goto(lvl); t0 = time.time()
    info = d.q("debug level")[0].split(); w, h = int(info[3 + 1]), int(info[5])
    seen, killed, stops, gold, items, known = {}, set(), [], [], [], set()
    for y in range(8, h - 6, 10):
        for x in range(8, w - 6, 10):
            if d.q("debug blocked %d %d" % (x, y))[-1].endswith("1"): continue
            d.q("debug warp %d %d" % (x, y)); d.step(30); fid.prime(d)
            if d.state()["level"] != str(clvl): setup()
            ms = d.monsters()
            for m in ms: seen.setdefault(m["unit"], (m["name"], m["lvl"], m["boss"], m["mods"]))
            alive = [m for m in ms if m["mode"] not in ("DD", "DT") and m["unit"] not in killed]
            setup(); d.step(1); x0 = num_exp()
            d.q("debug kill"); d.step(25)
            x1 = num_exp()
            killed |= {m["unit"] for m in alive}
            if alive: stops.append(dict(x=x, y=y, comp=[(m["name"], m["lvl"], m["boss"]) for m in alive], dxp=x1 - x0))
            for l in d.q("ground"):
                p = l.split("\t")
                if len(p) < 5 or p[4] in known: continue
                known.add(p[4]); u = int(p[4][1:])
                if p[0] == "gld": gold.append(dict(unit=u, label=p[1], x=x, y=y)); continue
                before = {i["id"] for i in json.loads(d.q("dump items")[0])["items"]}
                d.q("debug warp %s %s" % (p[5], p[6])); d.step(8); fid.prime(d)
                d.q("cmd pickup %d" % u); d.step(100)
                new = [i for i in json.loads(d.q("dump items")[0])["items"] if i["id"] not in before]
                items.append(dict(unit=u, code=p[0], label=p[1], got=[dict(code=i["code"], q=i["quality"], ilvl=i["ilvl"], name=i["name"], sock=i["sockets"]) for i in new]))
                d.q("debug clearinv")
    f.write(json.dumps(dict(level=lvl, seed=seed, clvl=clvl, w=w, h=h, monsters={str(k): v for k, v in seen.items()}, nmon=len(seen), killed=len(killed),
                            stops=stops, gold=gold, items=items, wall_s=round(time.time() - t0, 1))) + "\n"); f.flush()
    print("level", lvl, "monsters", len(seen), "killed", len(killed), "gold", len(gold), "items", len(items), "wall", round(time.time() - t0), flush=True)
fid.stop()
