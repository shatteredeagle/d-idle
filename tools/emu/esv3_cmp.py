#!/usr/bin/env python3
"""esverify3 comparison: every frame (the control twins included) of game.exe's own code (esv_game.py) against the d2d
base and head probes. Unlike esv_cmp.py nothing is filtered by a control twin: the claim under test is the no-shield path.

    python3 esv3_cmp.py CASES GAME BASE HEAD [IDS_OUT]

Classes: shield / no-shield, x red (flat or mdr non-zero) / nored, plus c2 (physical above 4,096 points with a percent),
poison (life not compared: the emulator hooks the poison handler), res>100 (a resistance above the cap: d2d's player never has it).
IDS_OUT gets one line per frame where the head differs from game.exe: `id class life_game life_head mana_game mana_head`.
"""
import sys
from collections import defaultdict

from esv_cmp import table, width


def tags(v):
    level, mana, dr_pct, dr_flat, mdr, rf, rl, rc, nb = v[2:11]
    w = width(v)
    blows = [v[11 + w * k: 11 + w * (k + 1)] for k in range(nb)]
    t = ["shield" if level > 0 else "no-shield"]
    t.append("red" if dr_flat or mdr else "nored")
    if w == 6 and any(b[5] > 0 for b in blows):
        t.append("poison")
    if dr_pct > 0 and any(b[0] > 4096 for b in blows):
        t.append("c2")
    if max(rf, rl, rc) > 100:
        t.append("res>100")
    return t


def main():
    cases = {}
    for line in open(sys.argv[1]):
        if line.strip():
            v = list(map(int, line.split()))
            cases[v[0]] = v
    game, base, head = table(sys.argv[2]), table(sys.argv[3]), table(sys.argv[4])
    out = open(sys.argv[5], "w") if len(sys.argv) > 5 else None
    cnt = defaultdict(lambda: defaultdict(int))
    missing = 0
    for k, v in cases.items():
        if k not in game or k not in base or k not in head:
            missing += 1
            continue
        t = tags(v)
        g = game[k]
        keys = [t[0] + ":" + t[1], t[0] + ":all"] + [t[0] + ":" + x for x in t[2:]]
        clean = "poison" not in t and "c2" not in t and "res>100" not in t
        if clean:
            keys.append(t[0] + ":" + t[1] + ":clean")
        for key in keys + ["all"]:
            c = cnt[key]
            c["frames"] += 1
            for side, d in (("base", base[k]), ("head", head[k])):
                if "poison" not in t:
                    c[side + ":life_pt"] += (d[0] >> 8) != (g[0] >> 8)
                    c[side + ":life_256"] += d[0] != g[0]
                c[side + ":mana_256"] += d[1] != g[1]
            c["base!=head"] += base[k][:2] != head[k][:2]
        hd = head[k]
        if out and (("poison" not in t and hd[0] != g[0]) or hd[1] != g[1]):
            out.write(f"{k}\t{','.join(t)}\t{g[0]}\t{hd[0]}\t{g[1]}\t{hd[1]}\t{base[k][0]}\n")
    cols = ["frames", "base:life_pt", "base:life_256", "head:life_pt", "head:life_256", "base:mana_256", "head:mana_256", "base!=head"]
    print(f"cases {len(cases)} missing {missing}")
    print("class".ljust(26) + "".join(c.rjust(15) for c in cols))
    for key in sorted(cnt):
        print(key.ljust(26) + "".join(str(cnt[key][c]).rjust(15) for c in cols))


if __name__ == "__main__":
    main()
