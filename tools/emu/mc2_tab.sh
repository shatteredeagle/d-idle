#!/bin/bash
# MERC2: summarise game_hire_1.txt / game_revive.txt of mc2_run.sh: Act 1 offer cost windows per difficulty and character level,
# and the resurrection price by merc level. (hire line: seed act diff clvl: id level life str dex cost exp def dmin dmax)
cd "$D2_SCRATCH"
head -2 game_hire_1.txt
python3 - <<'PY'
import collections
d = collections.defaultdict(list)
for l in open("game_hire_1.txt"):
    a, b = l.split(":")
    p = a.split(); act, diff, clvl = int(p[1]), int(p[2]), int(p[3])
    f = b.split()
    d[(act, diff, clvl)].append((int(f[0]), int(f[1]), int(f[5])))
for k in sorted(d):
    v = d[k]
    ids = sorted({x[0] for x in v})
    print(k, "n", len(v), "ids", ids, "level", min(x[1] for x in v), "-", max(x[1] for x in v), "cost", min(x[2] for x in v), "-", max(x[2] for x in v))
PY
awk 'NR>=2 && NR<=41 {printf "%s:%s ", $1, $2} END{print ""}' game_revive.txt
