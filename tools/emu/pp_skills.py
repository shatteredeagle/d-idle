"""prprep2: player skills whose missile rows have LastCollide, no NextHit, no Pierce and SrvDoFunc 1 (and 3), from Skills.txt + Missiles.txt (stdin-free: reads the two dumps named on the command line)."""
import csv
import sys

def table(path):
    r = list(csv.reader(open(path, encoding="latin1"), delimiter="\t"))
    h = r[0]
    return h, [dict(zip(h, x)) for x in r[1:]]

mh, missiles = table(sys.argv[1])
sh, skills = table(sys.argv[2])
print([c for c in mh if "do" in c.lower() or "func" in c.lower()][:8])
print([c for c in sh if "missile" in c.lower()][:10])
rows = {m["Missile"]: m for m in missiles if m.get("Missile")}
for want in ("1", "3"):
    names = {n for n, m in rows.items() if m.get("LastCollide") == "1" and not m.get("NextHit") and not m.get("Pierce") and not m.get("CollideKill") and m.get("pSrvDoFunc") == want}
    print("srvdofunc", want, "rows:", len(names))
    for s in skills:
        used = {s.get(c) for c in ("srvmissile", "srvmissilea", "srvmissileb", "srvmissilec")} & names
        if used and s.get("charclass") not in ("",):
            print("  ", want, s.get("skill"), s.get("charclass"), sorted(used))

for n in ("furylightning", "lightningbolt", "infernoflame1", "bonespear", "arcticblast1"):
    m = rows.get(n)
    print(n, {k: m.get(k) for k in ("LastCollide", "NextHit", "Pierce", "CollideKill", "pSrvDoFunc")} if m else None)
