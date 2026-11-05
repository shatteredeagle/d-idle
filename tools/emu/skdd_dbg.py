# SPDX-License-Identifier: GPL-3.0-or-later
"""debug aid: list shield cases of one category with game and d2d damage. python skdd_dbg.py cases game real"""
import sys
cs = {}
for line in open(sys.argv[1]):
    f = line.split()
    if len(f) == 16:
        cs[int(f[0])] = tuple(int(x) for x in f[1:])
g = {int(l.split()[1]): l.split() for l in open(sys.argv[2]) if l.startswith("M ")}
d = {int(l.split()[1]): l.split() for l in open(sys.argv[3]) if l.startswith("M ")}
n = 0
for cid, c in cs.items():
    d0, d1, armed, bm, bM, ed, fmin, fmax, dp, strn, dex, sb, db, off, broken = c
    if armed and broken and dp and not off and not fmin and not fmax and d[cid][2] != "0":
        print(cid, "bm", bm, bM, "ed", ed, "dp", dp, "str", strn, dex, "sb", sb, db, "game", int(g[cid][2]) >> 8, "d2d", d[cid][3])
        n += 1
        if n > 14:
            break
