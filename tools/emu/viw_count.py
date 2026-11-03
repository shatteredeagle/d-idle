#!/usr/bin/env python3
"""viw_count.py: independent recount of weapons.txt facts claimed in docs/research/itemweapons-trace.md.
usage: python3 -I tools/emu/viw_count.py weapons.txt   (the file made by mpq-table.sh weapons)"""
import sys, collections
lines = open(sys.argv[1], encoding="latin-1").read().split("\n")
# skip a wrapper line if present
i = next(k for k, l in enumerate(lines) if l.startswith("name\t"))
hdr = lines[i].split("\t")
rows = [dict(zip(hdr, l.split("\t"))) for l in lines[i + 1:] if l.strip()]
print("rows with short field list", [(r.get("name"), len(r)) for r in rows if "mindam" not in r])
rows = [r for r in rows if r.get("name") and r.get("code") and "mindam" in r]
print("header cols", len(hdr), "rows", len(rows))
print("expansion marker rows", [r["name"] for r in rows if r["name"].lower() == "expansion"])
def n(v):
    return int(v) if v not in ("", None) else 0
cl = collections.Counter(r.get("type", "") for r in rows)
print("type", cl)
for col in ("wclass", "wclass2"):
    print(col, collections.Counter(r.get(col, "<nocol>") for r in rows))
print("mindam>maxdam", sum(1 for r in rows if n(r["mindam"]) > n(r["maxdam"])))
print("2hmin>2hmax", sum(1 for r in rows if n(r["2handmindam"]) > n(r["2handmaxdam"])))
print("2handed=1", sum(1 for r in rows if r["2handed"] == "1"))
print("2handed=1 rows with 1h pair nonzero", sum(1 for r in rows if r["2handed"] == "1" and (n(r["mindam"]) or n(r["maxdam"]))))
print("2handed=1 rows lacking 2h pair", sum(1 for r in rows if r["2handed"] == "1" and not n(r["2handmaxdam"])))
print("1or2handed=1", sum(1 for r in rows if r["1or2handed"] == "1"))
print("speed distinct", sorted(collections.Counter(n(r["speed"]) for r in rows).items()))
print("blank speed rows", sum(1 for r in rows if r["speed"] == ""))
print("str or dex set", sum(1 for r in rows if n(r["StrBonus"]) or n(r["DexBonus"])))
print("neither", [r["name"] for r in rows if not (n(r["StrBonus"]) or n(r["DexBonus"]))])
print("dex bonus nonzero", collections.Counter((r.get("wclass"), n(r["StrBonus"]), n(r["DexBonus"])) for r in rows if n(r["DexBonus"])))
print("nodurability=1", sum(1 for r in rows if r["nodurability"] == "1"), "durability 0", sum(1 for r in rows if n(r["durability"]) == 0))
print("nodura and dura0", sum(1 for r in rows if r["nodurability"] == "1" and n(r["durability"]) == 0))
for nm in ("Hand Axe", "Short Sword", "Katar", "Phase Blade", "Giant Thresher"):
    for r in rows:
        if r["name"] == nm:
            print(nm, {k: r[k] for k in ("code", "wclass", "mindam", "maxdam", "2handed", "2handmindam", "2handmaxdam", "speed", "StrBonus", "DexBonus", "reqstr", "reqdex", "durability", "nodurability") if k in r})
print("code 7wc", [ (r["name"], r["2handmindam"], r["2handmaxdam"]) for r in rows if r["code"] == "7wc"])
