#!/usr/bin/env python3
"""esverify cases: one frame of monster blows on a player, with and without Energy Shield.

    python3 esv_cases.py SEED N > cases.tsv

Line: id cls level mana256 dr_pct dr_flat mdr res_fire res_light res_cold nblows {phys fire light cold magic} x nblows
id = seed * 10**7 + k. Edge grid first (seed 0 ids), then N random cases (Python's random.Random(seed)).
Kinds (id // 10**6 % 10 of the random cases is not used: the kind is in the comparison's own classification).
"""
import random
import sys

CLS = 1   # Sorceress


def edge():
    rows = []
    # single field, the edges the claim must survive: zero, tiny, MulDiv's 0x100000 (4096 points) boundary, huge;
    # mana 0, 1/256, exactly the cap, huge; resistance -100, 0, 99, 100; levels 1, 18, 40, 99
    for kind in range(5):
        for res in (-100, -1, 0, 1, 50, 99, 100):
            for roll in (1, 2, 3, 99, 4095, 4096, 4097, 12000):
                for level in (1, 18, 40, 99):
                    for mana in (0, 1, 255, 256, 257, 10000, 0x10000, 0x100001, 0x7FFFFF00):
                        b = [0, 0, 0, 0, 0]
                        b[kind] = roll
                        r = [0, 0, 0]
                        if kind == 0:
                            dr = min(max(res, 0), 50)
                        else:
                            dr = 0
                            if 1 <= kind <= 3:
                                r[kind - 1] = res
                            elif res != 0:
                                continue          # magic: no percent resistance on this path (claim's scope)
                        rows.append((CLS, level, mana, dr, 0, 0, *r, 1, *b))
    # all four elements and the physical in one record, the mana running out inside the record
    for mana in (0, 300, 3000, 30000, 300000):
        for level in (1, 40):
            rows.append((CLS, level, mana, 20, 3, 2, 75, -50, 30, 1, 300, 200, 150, 120, 80))
            rows.append((CLS, level, mana, 0, 0, 0, 0, 0, 0, 3, 100, 0, 0, 0, 0, 0, 100, 0, 0, 0, 0, 0, 0, 100, 0))
    return rows


def rand_case(rng):
    level = 0 if rng.random() < 0.15 else rng.choice((rng.randint(1, 40), rng.randint(1, 40), rng.randint(1, 99)))
    mana = rng.choice((0, rng.randint(0, 600 * 256), rng.randint(0, 600 * 256), rng.randint(0, 40 * 256), rng.randint(0, 255)))
    dr_pct = 0 if rng.random() < 0.6 else rng.randint(1, 50)
    dr_flat = 0 if rng.random() < 0.8 else rng.randint(1, 15)
    mdr = 0 if rng.random() < 0.8 else rng.randint(1, 15)
    res = [rng.choice((0, rng.randint(-100, 95), rng.randint(-100, 75), 100)) for _ in range(3)]
    n = rng.choice((1, 1, 1, 2, 3))
    blows = []
    for _ in range(n):
        phys = 0 if rng.random() < 0.3 else rng.choice((rng.randint(1, 60), rng.randint(1, 600), rng.randint(1, 2500)))
        el = []
        for p in (0.35, 0.35, 0.35, 0.15):
            el.append(rng.choice((rng.randint(1, 60), rng.randint(1, 800), rng.randint(1, 2000))) if rng.random() < p else 0)
        blows += [phys, *el]
    return (CLS, level, mana, dr_pct, dr_flat, mdr, *res, n, *blows)


def main():
    seed, n = int(sys.argv[1]), int(sys.argv[2])
    rows = edge() if seed == 0 else []
    rng = random.Random(seed)
    rows += [rand_case(rng) for _ in range(n)]
    for k, r in enumerate(rows):
        print(seed * 10 ** 7 + k, *r, sep="\t")


if __name__ == "__main__":
    main()
