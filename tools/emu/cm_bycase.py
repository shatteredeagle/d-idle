# covmissiles: per-skill count of differing lines between game.tsv and a d2d output, and of cases.
#   cm_bycase.py DIR     (DIR holds cases.tsv game.tsv before.tsv after.tsv)
import collections
import sys

d = sys.argv[1]
skill = {}
for line in open(f"{d}/cases.tsv"):
    f = line.split()
    skill[int(f[0])] = int(f[1])
game = {int(l.split()[0]): l.split() for l in open(f"{d}/game.tsv") if l[0].isdigit()}
for v in ("before", "after"):
    out = {int(l.split()[0]): l.split() for l in open(f"{d}/{v}.tsv")}
    tot, bad = collections.Counter(), collections.Counter()
    minmax = 0
    for i, g in game.items():
        tot[skill[i]] += 1
        o = out.get(i)
        if o != g:
            bad[skill[i]] += 1
        if o and o[1:3] != g[1:3]:
            minmax += 1
    print(v, "min/max differing:", minmax)
    for s in sorted(tot):
        print(" ", v, "skill", s, "cases", tot[s], "differing", bad[s])
