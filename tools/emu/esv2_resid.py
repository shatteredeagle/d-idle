#!/usr/bin/env python3
"""esverify2: where the head still differs from game.exe on shield cases, by cause.

    python3 esv2_resid.py RUN_DIR...

Per run: shield cases (life compared: no poison), head life differs (whole points, 256ths), split by the control twin
(ctrl-exact: game, base and head agree on the same frame without the shield, so a difference is the shield's own;
ctrl-differs: the no-shield path already differs) and, for ctrl-exact, by kind: C2 (a physical roll above 4,096 with a
percent reduction) or other (listed). Also: head worse than base / better than base, mana 256ths differences, and
no-shield frames where base != head (must be 0).
"""
import sys

import esv_cmp

CTRL = esv_cmp.CTRL


def blows(v):
    w = esv_cmp.width(v)
    return [v[11 + w * k: 11 + w * (k + 1)] for k in range(v[10])]


tot = {}
for R in sys.argv[1:]:
    cases = {}
    for line in open(f"{R}/cases.tsv"):
        v = list(map(int, line.split()))
        cases[v[0]] = v
    game, base, head = (esv_cmp.table(f"{R}/{n}.tsv") for n in ("game", "base", "head"))
    c = dict.fromkeys(("shield", "life_cmp", "head_pt", "head_256", "exact_pt", "exact_256", "exact_c2", "exact_other", "differs_pt",
                       "base_pt", "worse", "better", "mana_head", "mana_base", "noshield_base_ne_head", "poison_cases", "exact_cases", "exact_other_mana_out", "exact_base_pt", "exact_base_256"), 0)
    other = []
    for k, v in cases.items():
        if k % 10 ** 7 >= CTRL or v[2] == 0:
            if k in base and base[k] != head[k]:
                c["noshield_base_ne_head"] += 1
            continue
        c["shield"] += 1
        g, b, h = game[k], base[k], head[k]
        c["mana_head"] += h[1] != g[1]
        c["mana_base"] += b[1] != g[1]
        if any(len(x) > 5 and x[5] > 0 for x in blows(v)):
            c["poison_cases"] += 1
            continue
        c["life_cmp"] += 1
        hp, bp, gp = h[0] >> 8, b[0] >> 8, g[0] >> 8
        c["head_pt"] += hp != gp
        c["head_256"] += h[0] != g[0]
        c["base_pt"] += bp != gp
        c["worse"] += bp == gp and hp != gp
        c["better"] += bp != gp and hp == gp
        t = k + CTRL
        exact = t in game and game[t][0] == base[t][0] == head[t][0]
        if exact:
            c["exact_cases"] += 1
            c["exact_base_pt"] += bp != gp
            c["exact_base_256"] += b[0] != g[0]
            c["exact_pt"] += hp != gp
            c["exact_256"] += h[0] != g[0]
            if h[0] != g[0]:
                if v[4] and any(x[0] > 4096 for x in blows(v)):
                    c["exact_c2"] += 1
                else:
                    c["exact_other"] += 1
                    # the mana ran out inside the frame (game leaves under 1 point): fields after it are on the no-shield path,
                    # whose known differences (flat / mdr order, 256ths) can cancel in the control twin
                    c["exact_other_mana_out"] += g[1] < 256
                    if len(other) < 5:
                        other.append((k, v[1:], g, b, h))
        else:
            c["differs_pt"] += hp != gp
    print(R, c)
    for o in other:
        print("   other:", o)
    for key, val in c.items():
        tot[key] = tot.get(key, 0) + val
if len(sys.argv) > 2:
    print("total", tot)
