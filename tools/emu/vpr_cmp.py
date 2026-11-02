# Verifier VERIFYPRICES: compare game.exe oracle lines with d2d probe lines (card verifyprices).
#   vpr_cmp.py types                       ItemTypes Repair = 0 rows and their bases' durability columns (from the MPQ)
#   vpr_cmp.py cmp CASES GAME BASE HEAD    per-claim counts: M (mults) and G (repair gate) base vs head vs game
import sys
from collections import Counter

import vpr_oracle


def types():
    rows = vpr_oracle.txt("itemtypes")
    norep = {r["Code"] for r in rows if r.get("Code") and r.get("Repair", "") != "1"}
    print("ItemTypes Repair!=1:", len(norep), sorted(norep))
    for name in ("weapons", "armor", "misc"):
        for r in vpr_oracle.txt(name):
            if r.get("type") in norep and r.get("code"):
                print(name, r["code"], r["type"], "nodur", r.get("nodurability", ""), "dur", r.get("durability", ""), "stack", r.get("stackable", ""))


def load(path):
    out = {}
    for line in open(path):
        f = line.split()
        if f and f[0][0] in "mg": out[f[0]] = f[1:]
    return out


def cmp(cases, game_path, base_path, head_path):
    g, b, h = load(game_path), load(base_path), load(head_path)
    meta = {f[1]: f for f in (line.split() for line in open(cases)) if f and f[0] in "MG"}
    tally = Counter()
    examples = {}
    for cid, gv in g.items():
        f = meta[cid]
        if cid[0] == "m":
            p, diff = int(f[2]), int(f[4])
            gb, gs, gr, cap = (int(x) for x in gv)
            want = (gb, max(min(gs, cap), 1), max(gr, 1) if gr > 0 else gr)
            big = "big" if p > 0xffff else "small"
            for side, d in (("base", b), ("head", h)):
                db, ds, dr = (int(x) for x in d[cid])
                for k, (x, y) in zip(("buy", "sell", "rep"), zip((db, ds, dr), want)):
                    tally[(side, "M", big, k, "differ" if x != y else "same")] += 1
                    if x != y: examples.setdefault((side, big, k), (cid, f, (db, ds, dr), want))
        else:
            _, _, code, dur, maxd, qty, ident, eth, indes, chg, npc, diff = f
            gate = gv[0] == "1"
            damaged = (int(maxd) > 0 and int(dur) < int(maxd)) or chg == "2"
            for side, d in (("base", b), ("head", h)):
                cost = int(d[cid][0])
                cls = "damaged" if damaged else "whole"
                tally[(side, "G", cls, "agree" if (cost > 0) == gate else ("d2d>0,game-refuses" if cost > 0 else "d2d0,game-allows"))] += 1
                if (cost > 0) != gate and damaged: examples.setdefault((side, "G", cost > 0), (cid, f, cost, gate))
    for k in sorted(tally): print(*k, tally[k])
    for k, v in examples.items(): print("example", k, v)


if __name__ == "__main__":
    if sys.argv[1] == "types": types()
    else: cmp(*sys.argv[2:6])
