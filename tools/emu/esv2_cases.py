#!/usr/bin/env python3
"""esverify2 cases (re-verify of the Energy Shield head 229aeca7): one frame of monster blows on a player, six numbers a blow.

    python3 esv2_cases.py SEED N > cases.tsv       SEED 0: the edge grid below, then N random cases of rand_case2

Line: id cls level mana256 dr_pct dr_flat mdr res_fire res_light res_cold nblows {phys fire light cold magic poison} x nblows
Same id scheme and control twins (id + 5,000,000, level 0) as esv_cases.py. Seeds 21/22/23 of the first generator
(esv_cases.py) are the comparable runs; this file adds what the first grid left out: flat AND mdr both non-zero, a field
the reduction takes under 0 (alone and next to another field), resistance -100 / 75 / 100, all six damage types (poison
has no shield field), shield level 0 (the twins) and mana 0. rand_case2 draws flat and mdr far more often than the first.
"""
import random
import sys

CLS = 1   # Sorceress
CTRL = 5 * 10 ** 6
REDS = ((0, 0), (15, 0), (0, 10), (15, 10), (3, 6), (40, 40))   # (flat, mdr)


def blow(kind, roll):
    b = [0] * 6
    b[kind] = roll
    return b


def edge2():
    rows = []
    for kind in range(6):                       # physical, fire, lightning, cold, magic, poison
        for res in (-100, 0, 50, 75, 100):
            if kind in (4, 5) and res != 0:
                continue                         # no percent magic resistance on a monster blow; poison resistance left at 0
            for roll in (1, 5, 10, 50, 632):
                for flat, mdr in REDS:
                    for level in (1, 40, 99):
                        for mana in (0, 1, 300 * 256):
                            r = [0, 0, 0]
                            dr = 0
                            if kind == 0:
                                dr = min(max(res, 0), 50)
                            elif kind <= 3:
                                r[kind - 1] = res
                            rows.append((CLS, level, mana, dr, flat, mdr, *r, 1, *blow(kind, roll)))
    # two fields in one record: a small one the reduction takes under 0 next to a big one (the negative is summed)
    pairs = (((0, 3), (1, 200)), ((1, 4), (3, 120)), ((2, 2), (4, 90)), ((0, 8), (2, 300)), ((3, 1), (1, 1)), ((4, 5), (5, 40)))
    for (ka, ra), (kb, rb) in pairs:
        for flat, mdr in REDS:
            for res in (-100, 0, 50):
                for level in (1, 40):
                    for mana in (0, 50 * 256, 0x100001):
                        b = [0] * 6
                        b[ka] += ra
                        b[kb] += rb
                        rows.append((CLS, level, mana, 0, flat, mdr, res, res, res, 1, *b))
    return rows


def rand_case2(rng):
    level = 0 if rng.random() < 0.1 else rng.choice((rng.randint(1, 40), rng.randint(1, 99)))
    mana = rng.choice((0, rng.randint(0, 600 * 256), rng.randint(0, 40 * 256), rng.randint(0, 255)))
    dr_pct = 0 if rng.random() < 0.5 else rng.randint(1, 50)
    dr_flat = 0 if rng.random() < 0.5 else rng.choice((rng.randint(1, 15), rng.randint(1, 60)))
    mdr = 0 if rng.random() < 0.5 else rng.choice((rng.randint(1, 15), rng.randint(1, 60)))
    res = [rng.choice((0, -100, 75, 100, rng.randint(-100, 95))) for _ in range(3)]
    n = rng.choice((1, 1, 2, 3))
    blows = []
    for _ in range(n):
        phys = 0 if rng.random() < 0.3 else rng.choice((rng.randint(1, 20), rng.randint(1, 600), rng.randint(1, 2500)))
        el = []
        for p in (0.35, 0.35, 0.35, 0.2, 0.15):  # fire, lightning, cold, magic, poison
            el.append(rng.choice((rng.randint(1, 20), rng.randint(1, 800), rng.randint(1, 2000))) if rng.random() < p else 0)
        while sum(1 for x in el if x) > 3:      # three element slots a MonStats row (rules::MonStats::elements)
            el[max(i for i, x in enumerate(el) if x)] = 0
        blows += [phys, *el]
    return (CLS, level, mana, dr_pct, dr_flat, mdr, *res, n, *blows)


def main():
    seed, n = int(sys.argv[1]), int(sys.argv[2])
    rows = edge2() if seed == 0 else []
    rng = random.Random(seed)
    rows += [rand_case2(rng) for _ in range(n)]
    for k, r in enumerate(rows):
        print(seed * 10 ** 7 + k, *r, sep="\t")
        if r[1] > 0:
            print(seed * 10 ** 7 + CTRL + k, r[0], 0, *r[2:], sep="\t")


if __name__ == "__main__":
    main()
