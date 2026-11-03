# SPDX-License-Identifier: GPL-3.0-or-later
"""verifyskdruid: read back the game.exe side of a vsd_run.sh run (scratch dir): the values the claims rest on.
    vsd_check.py DIR SEED"""
import sys

d, seed = sys.argv[1], sys.argv[2]
NAMES = (["calc1", "calc2", "calc3", "calc4", "petmax"] + ["sumsk%d" % k for k in range(1, 6)]
         + ["delay", "aurarange", "auralen", "perdelay"] + ["aura%d" % k for k in range(1, 7)]
         + ["passive%d" % k for k in range(1, 6)])
cases = {int(l.split()[0]): [int(x) for x in l.split()] for l in open("%s/cases-%s.tsv" % (d, seed))}
game = {int(l.split()[0]): l.split()[1:] for l in open("%s/game-%s.tsv" % (d, seed))}
distinct = {}
for i, v in game.items():
    s = cases[i][1]
    for name, x in zip(NAMES, v):
        if x != "x":
            distinct.setdefault((s, name), set()).add(int(x))
print("cells with a value, and how many distinct values the game produced over all cases:")
for s in range(221, 251):
    print(s, " ".join("%s:%d" % (n, len(distinct[(s, n)])) for n in NAMES if (s, n) in distinct))
print("fixed cases (skill lvl clvl neighbours) -> non-x game cells:")
for i in sorted(game):
    c = cases[i]
    if i >= 840:
        break
    if c[2] in (1, 5, 20, 40):
        cells = ["%s=%s" % (n, x) for n, x in zip(NAMES, game[i]) if x != "x" and not (n.startswith("sumsk") and x == "x")]
        print(c[1], "lvl", c[2], "clvl", c[3], "nb" if c[4] else "alone", " ".join(cells))
