"""SHRINE (decomp17): count the shrine objects per level in a shr_dump.sh output (d2d's Act 1 normal generation), and the odds of each Shrines.txt row.
    python3 shr_count.py DUMP OBJECTS.txt SHRINES.txt LEVELS.txt     (the three tables from tools/emu/mpq-table.sh)
Shrine object = objects.txt row with InitFn 1 (OperateFn 2). Per level: mean and spread of the shrine objects per game, by parm0 class.
Then the odds of each Shrines.txt row for a shrine of parm0 P on level id L, from the roll rules of FUN_0054f9d0 as d2d ports them (S)."""
import sys, re, collections

def table(p):
    L = open(p, encoding="latin-1").read().splitlines()
    h = L[0].split("\t")
    return [dict(zip(h, l.split("\t"))) for l in L[1:]]

dump, objp, shrp, levp = sys.argv[1:5]
obj = {}
for r in table(objp):
    if r.get("Id", "").isdigit() and r.get("InitFn") == "1" and r.get("OperateFn") == "2":
        obj[int(r["Id"])] = int(r["Parm0"])
lv = {int(r["Id"]): r["Name"] for r in table(levp) if r.get("Id", "").isdigit()}
seeds = 0; per = collections.defaultdict(lambda: collections.Counter()); lvl = None
for ln in open(dump):
    if ln.startswith("seed"): seeds += 1; per_seed = collections.Counter(); continue
    if ln.startswith("level "): lvl = int(ln.split()[1]); continue
    if ln.startswith("room"):
        for m in re.finditer(r" (\d+)@(\d+),(\d+)", ln):
            o = int(m.group(1))
            if o in obj: per[lvl][(seeds, obj[o])] += 1
print("seeds", seeds)
print("level id name | mean shrines per game | parm0 1 (health) 2 (mana) 3 (boost) | games with 0 | max")
for l in sorted(per):
    tot = collections.Counter(); games = collections.Counter(); byp = collections.defaultdict(int)
    for (s, p), n in per[l].items(): games[s] += n; byp[p] += n
    vals = [games.get(s, 0) for s in range(1, seeds + 1)]
    print("%3d %-26s %5.2f | %4.2f %4.2f %4.2f | %3d | %d" % (l, lv.get(l, "?"), sum(vals) / seeds, byp[1] / seeds, byp[2] / seeds, byp[3] / seeds, vals.count(0), max(vals)))
# row odds
S = table(shrp)
rows = [dict(code=int(r["Code"]), cls=int(r["effectclass"]), lmin=int(r["LevelMin"]), name=r["Shrine name"]) for r in S]
def odds(parm0, level_id):
    res = collections.defaultdict(float)
    def pick(cls_list):
        # up to 8 tries, uniform over the class rows, stop at the first with level_id >= LevelMin; the last try stands
        n = len(cls_list); allowed = [i for i in cls_list if level_id >= rows[max(i, 1)]["lmin"]]
        out = collections.defaultdict(float)
        miss = (n - len(allowed)) / n
        stop = 0.0
        for k in range(8):
            for i in allowed: out[i] += (miss ** k) / n
        for i in cls_list:
            if i not in allowed: out[i] += (miss ** 7) / n   # eight low rolls: the last (low) row stands
        return out
    if parm0 == 1: cl = [(2, 1.0)]
    elif parm0 == 2: cl = [(3, 1.0)]
    else: cl = [(1, 0.1), (4, 0.9)]
    for cls, w in cl:
        lst = [i for i, r in enumerate(rows) if r["cls"] == cls]
        for i, p in pick(lst).items():
            fin = {5: 3, 4: 2, 16: 18}.get(i, i)
            res[fin] += w * p
    return res
for P in (3, 1, 2):
    print("parm0", P)
    for L in (2, 3, 4, 5, 6, 8, 10, 17, 21, 26, 33):
        o = odds(P, L)
        print("  level %2d: " % L + ", ".join("%s %.1f%%" % (rows[i]["name"], 100 * p) for i, p in sorted(o.items(), key=lambda x: -x[1]) if p > 0.0005))
