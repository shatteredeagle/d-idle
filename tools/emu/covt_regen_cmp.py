# SPDX-License-Identifier: GPL-3.0-or-later
"""covtreasure2: compare two covt_regen files (game, d2d) line by line; count the lines that differ by cause.

    covt_regen_cmp.py GAME D2D
"""
import sys


def main():
    g = open(sys.argv[1]).read().split("\n")
    d = [x for x in open(sys.argv[2]).read().split("\n") if " ::: " in x]
    g = [x for x in g if " ::: " in x]
    assert len(g) == len(d), (len(g), len(d))
    causes = {"flat < 0 (the game can take mana)": 0, "mana above max (the game pulls it down)": 0, "other": 0}
    shown = 0
    n = bad = 0
    for a, b in zip(g, d):
        n += 1
        if a == b:
            continue
        bad += 1
        cls, mx, mana, bonus, flat = (int(v) for v in a.split(" ::: ")[0].split())
        if mana > mx:
            causes["mana above max (the game pulls it down)"] += 1
        elif flat < 0:
            causes["flat < 0 (the game can take mana)"] += 1
        else:
            causes["other"] += 1
            if shown < 5:
                print("other:", a, "|", b)
                shown += 1
    print(f"cases {n}, differ {bad}: {causes}")


if __name__ == "__main__":
    main()
