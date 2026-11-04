#!/usr/bin/env python3
"""esverify: shield cases the head gets wrong where the base was right (life whole points or mana 256ths), by kind.
    python3 esv_regress.py RUN_DIR..."""
import sys

import esv_cmp

for R in sys.argv[1:]:
    cases = {}
    for line in open(f"{R}/cases.tsv"):
        v = list(map(int, line.split()))
        cases[v[0]] = v
    game, base, head = (esv_cmp.table(f"{R}/{n}.tsv") for n in ("game", "base", "head"))
    out = {}
    for k, v in cases.items():
        if k % 10 ** 7 >= esv_cmp.CTRL or v[2] == 0:
            continue
        g, b, h = game[k], base[k], head[k]
        for name, gv, bv, hv in (("life_pt", g[0] >> 8, b[0] >> 8, h[0] >> 8), ("mana_256", g[1], b[1], h[1])):
            if bv == gv and hv != gv:
                key = (name, "flat/mdr" if v[5] or v[6] else "no flat/mdr")
                out[key] = out.get(key, 0) + 1
            if bv != gv and hv == gv:
                out[(name, "fixed")] = out.get((name, "fixed"), 0) + 1
    print(R, "shield cases", sum(1 for k, v in cases.items() if k % 10 ** 7 < esv_cmp.CTRL and v[2] > 0), dict(sorted(out.items())))
