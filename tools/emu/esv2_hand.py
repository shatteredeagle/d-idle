#!/usr/bin/env python3
"""esverify2 hand cases: the no-shield order (flat / mdr before the resistance, a negative field summed) and poison.

    python3 esv2_hand.py SEED N > cases.tsv      (SEED and N ignored; ids 9 * 10**7 + k, control twins as esv_cases.py)

Each row: (note, level, mana256, dr_pct, flat, mdr, res_fire, res_light, res_cold, blows of 6) -- printed to stderr as an index.
"""
import sys

CLS = 1
CTRL = 5 * 10 ** 6
BIG = 600 * 256
ROWS = [
    ("fire 100 res 50 mdr 10 (PR: game 45)", 0, 0, 0, 0, 10, 50, 0, 0, [(0, 100, 0, 0, 0, 0)]),
    ("same under a shield", 40, BIG, 0, 0, 10, 50, 0, 0, [(0, 100, 0, 0, 0, 0)]),
    ("fire 100 res 75 mdr 10", 0, 0, 0, 0, 10, 75, 0, 0, [(0, 100, 0, 0, 0, 0)]),
    ("fire 100 res -100 mdr 10", 0, 0, 0, 0, 10, -100, 0, 0, [(0, 100, 0, 0, 0, 0)]),
    ("fire 5 + cold 100, mdr 6 (fire -1 summed?)", 0, 0, 0, 0, 6, 0, 0, 0, [(0, 5, 0, 100, 0, 0)]),
    ("fire 5 res -100 + cold 100 res -100, mdr 10 (resistance on a negative?)", 0, 0, 0, 0, 10, -100, 0, -100, [(0, 5, 0, 100, 0, 0)]),
    ("phys 10 flat 15 + fire 50 (phys -5 summed?)", 0, 0, 0, 15, 0, 0, 0, 0, [(10, 50, 0, 0, 0, 0)]),
    ("phys 10 flat 15 dr 50 + fire 50", 0, 0, 50, 15, 0, 0, 0, 0, [(10, 50, 0, 0, 0, 0)]),
    ("phys 100 flat 10 dr 50", 0, 0, 50, 10, 0, 0, 0, 0, [(100, 0, 0, 0, 0, 0)]),
    ("fire 3 mdr 10 alone (total under 0)", 0, 0, 0, 0, 10, 0, 0, 0, [(0, 3, 0, 0, 0, 0)]),
    ("magic 50 mdr 10", 0, 0, 0, 0, 10, 0, 0, 0, [(0, 0, 0, 0, 50, 0)]),
    ("magic 5 + light 100 mdr 10", 0, 0, 0, 0, 10, 0, 0, 0, [(0, 0, 100, 0, 5, 0)]),
    ("poison 100 alone, shield", 40, BIG, 0, 0, 0, 0, 0, 0, [(0, 0, 0, 0, 0, 100)]),
    ("poison 100 alone, shield, mana 0", 40, 0, 0, 0, 0, 0, 0, 0, [(0, 0, 0, 0, 0, 100)]),
    ("poison 100 + fire 100, shield", 40, BIG, 0, 0, 0, 0, 0, 0, [(0, 100, 0, 0, 0, 100)]),
    ("fire 100 mdr 10 res 50, shield, mana 0 (untouched field)", 40, 0, 0, 0, 10, 50, 0, 0, [(0, 100, 0, 0, 0, 0)]),
    ("phys 200 + fire 100 mdr 10 res 50, mana for the phys only", 40, 300, 0, 0, 10, 50, 0, 0, [(200, 100, 0, 0, 0, 0)]),
]


def main():
    for k, (note, level, mana, dr, flat, mdr, rf, rl, rc, blows) in enumerate(ROWS):
        flat_blows = [x for b in blows for x in b]
        row = (CLS, level, mana, dr, flat, mdr, rf, rl, rc, len(blows), *flat_blows)
        cid = 9 * 10 ** 7 + k
        print(cid, *row, sep="\t")
        print(f"{cid}\t{note}", file=sys.stderr)
        if level > 0:
            print(cid + CTRL, CLS, 0, *row[2:], sep="\t")


if __name__ == "__main__":
    main()
