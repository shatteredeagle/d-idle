# covmissiles: which missile ids earn a ledger row from the flight run (loaded by d2d, not claimed yet, not in MINE).
#   python3 cm_ledger_rows.py MISSILES_TSV D2D_FLIGHT_TSV LEDGER_TSV
import sys


def rows(p):
    r = [l.rstrip("\n").split("\t") for l in open(p, encoding="latin-1") if "\t" in l]
    h = next(i for i, x in enumerate(r) if x[0] == "Missile")
    return [dict(zip(r[h], x)) for x in r[h + 1:] if x and x[0]]


idof = {x["Missile"]: int(x["Id"]) for x in rows(sys.argv[1])}
loaded = set()
for l in open(sys.argv[2]):
    f = l.rstrip("\n").split("\t")
    if len(f) > 2 and f[0] in idof and f[2] != "-":
        loaded.add(idof[f[0]])
claimed = set()
for l in open(sys.argv[3]):
    p = l.rstrip("\n").split("\t")
    if p[0] == "missiles":
        for t in p[1].split(","):
            if "-" in t:
                a, b = t.split("-")
                claimed.update(range(int(a), int(b) + 1))
            else:
                claimed.add(int(t))
mine = {62, 96, 264, 166, 60, 233}
use = sorted(loaded - claimed - mine)
print(len(loaded), "loaded;", len(claimed), "claimed;", len(use), "to claim")
out, s, p = [], None, None
for i in use:
    if s is None:
        s = p = i
    elif i == p + 1:
        p = i
    else:
        out.append((s, p))
        s = p = i
out.append((s, p))
print(",".join(str(a) if a == b else f"{a}-{b}" for a, b in out))
print("claimed ids d2d does not load:", sorted(claimed - loaded))
