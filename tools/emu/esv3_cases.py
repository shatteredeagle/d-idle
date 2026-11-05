#!/usr/bin/env python3
"""esverify3 edge grid (verify of the no-shield reduction order, fork head ca7872ab): one frame of monster blows on a
player, six numbers a blow (esv2 line format, control twins at id + 5,000,000 with level 0).

    python3 esv3_cases.py N > cases.tsv     the grid below, then N random cases of esv2_cases.rand_case2 (seed 3)

Axes: flat 0 / 3 / 15 / 60, mdr 0 / 10 / 50 / 100, resistance -100 / 0 / 75 / 100 / 110 (above the cap), all six damage types,
physical percent 0 / 20 / 50, shield level 0 or 99 with mana 0 or 300 points; two-field records with one field under the
reduction; physical above 4,096 points with a percent (C2, known pre-existing); zero-flat controls are the flat 0 / mdr 0 rows.
"""
import random
import sys

import esv2_cases

CLS = 1
CTRL = 5 * 10 ** 6
FLATS = (0, 3, 15, 60)
MDRS = (0, 10, 50, 100)
RESS = (-100, 0, 75, 100, 110)


def grid():
    rows = []
    for kind in range(6):
        for res in RESS:
            if kind in (0, 4, 5) and res != 0:
                continue
            for roll in (1, 9, 10, 11, 49, 51, 101, 333, 2000):
                for flat in FLATS:
                    for mdr in MDRS:
                        for pct in ((0, 20, 50) if kind == 0 else (0,)):
                            for level, mana in ((0, 0), (99, 0), (99, 300 * 256)):
                                r = [0, 0, 0]
                                if 1 <= kind <= 3:
                                    r[kind - 1] = res
                                b = [0] * 6
                                b[kind] = roll
                                rows.append((CLS, level, mana, pct, flat, mdr, *r, 1, *b))
    # two and three fields, one of them under the flat / mdr; mixed resistances
    combos = (((0, 10), (1, 50)), ((1, 5), (3, 100)), ((2, 8), (4, 200)), ((4, 3), (1, 30), (3, 30)), ((0, 2), (2, 2), (3, 2)),
              ((0, 300), (1, 5), (5, 40)), ((3, 60), (2, 60)), ((0, 40), (4, 40)))
    for combo in combos:
        for flat in FLATS:
            for mdr in MDRS:
                for res in ((0, 0, 0), (-100, 75, 100), (75, -100, 0), (110, 50, -50)):
                    for pct in (0, 30):
                        for level, mana in ((0, 0), (99, 0), (99, 300 * 256), (40, 20 * 256)):
                            b = [0] * 6
                            for k, v in combo:
                                b[k] += v
                            rows.append((CLS, level, mana, pct, flat, mdr, *res, 1, *b))
    # several blows in one frame
    for flat in FLATS:
        for mdr in MDRS:
            for level, mana in ((0, 0), (99, 300 * 256)):
                rows.append((CLS, level, mana, 10, flat, mdr, 50, 0, -100, 3,
                             5, 20, 0, 0, 0, 0, 0, 0, 7, 3, 0, 0, 100, 0, 0, 0, 6, 0))
    # C2: physical above 4,096 points with a percent reduction
    for phys in (4097, 6000, 12000):
        for pct in (1, 20, 50):
            for flat in (0, 15):
                for level, mana in ((0, 0), (99, 300 * 256)):
                    rows.append((CLS, level, mana, pct, flat, 0, 0, 0, 0, 1, phys, 0, 0, 0, 0, 0))
    return rows


def main():
    n = int(sys.argv[1]) if len(sys.argv) > 1 else 0
    rows = grid()
    rng = random.Random(3)
    rows += [esv2_cases.rand_case2(rng) for _ in range(n)]
    seed = 4
    for k, r in enumerate(rows):
        print(seed * 10 ** 7 + k, *r, sep="\t")
        if r[1] > 0:
            print(seed * 10 ** 7 + CTRL + k, r[0], 0, *r[2:], sep="\t")


if __name__ == "__main__":
    main()
