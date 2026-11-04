#!/usr/bin/env python3
"""esverify: list the head's life / mana differences on shield cases whose no-shield control agrees exactly.
    python3 esv_fail.py RUN_DIR [max]"""
import sys

import esv_cmp

R = sys.argv[1]
cap = int(sys.argv[2]) if len(sys.argv) > 2 else 40
cases = {}
for line in open(f"{R}/cases.tsv"):
    v = list(map(int, line.split()))
    cases[v[0]] = v
game, base, head = (esv_cmp.table(f"{R}/{n}.tsv") for n in ("game", "base", "head"))
kinds = {}
shown = 0
for k, v in sorted(cases.items()):
    c = k + esv_cmp.CTRL
    if k % 10 ** 7 >= esv_cmp.CTRL or c not in game or not (game[c][0] == base[c][0] == head[c][0]):
        continue
    g, h = game[k], head[k]
    if (g[0] >> 8) != (h[0] >> 8) or g[1] != h[1]:
        blows = [v[11 + 5 * i: 16 + 5 * i] for i in range(v[10])]
        big = any(x > 4096 for b in blows for x in b)
        key = ("big field" if big else "small") + (", dr%" if v[4] else "") + (", flat/mdr" if v[5] or v[6] else "")
        kinds[key] = kinds.get(key, 0) + 1
        if shown < cap:
            print(k, v[1:], "game", g, "head", h, "base", base[k])
            shown += 1
print("kinds:", kinds)
