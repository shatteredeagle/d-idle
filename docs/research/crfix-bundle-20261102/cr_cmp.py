# cr_cmp.py CASES GAME D2D... : per-case join of the game.exe base part (cr_repair_game.py game) and d2d probe output(s)
import sys
from collections import Counter

def load(path):
    return {f[0]: f[1:] for f in (ln.split() for ln in open(path)) if f}

cases = {f[1]: f for f in (ln.split() for ln in open(sys.argv[1])) if f}
game = load(sys.argv[2])
for path in sys.argv[3:]:
    d2d = load(path)
    tally, ex = Counter(), []
    for cid, g in game.items():
        _, _, code, dur, maxd, indes, base, npc, diff = cases[cid]
        durable = int(maxd) > 0 and not int(indes)         # shape only, for the report
        cls = "no-dur" if g[0] == base else ("scaled" if int(g[0]) != int(base) else "?")
        want = int(g[1])
        if cid not in d2d: tally[(cls, "no-d2d-row")] += 1; continue
        got = int(d2d[cid][0])
        tally[(cls, "same" if got == want else "differ")] += 1
        if got != want and len(ex) < 5: ex.append((cid, code, dur, maxd, indes, base, npc, "game", want, "d2d", got))
    print(path, dict(sorted(tally.items())))
    for e in ex: print("  example", *e)
