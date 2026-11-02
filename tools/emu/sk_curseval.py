# SPDX-License-Identifier: GPL-3.0-or-later
"""SKCURSEVAL: every calc cell of the curse rows (and Conviction) at EVERY skill level 1..99, by game.exe's own calc
machine (FUN_00646ca0, as nec_calc.py) vs d2d's eval_calc.

    python3 sk_curseval.py gen N SEED > cases.tsv
    python3 sk_curseval.py game cases.tsv > game.tsv
    test_sk_rest_calc --calc cases.tsv > d2d.tsv        # tools/emu/nec_test.cpp built as test_sk_rest_calc
    diff game.tsv d2d.tsv

Rows: Amplify Damage 66, Dim Vision 71, Weaken 72, Iron Maiden 76, Terror 77, Life Tap 82, Decrepify 87, Lower Resist 91
(srvdofunc 30, the curse callback), Conviction 123 (srvdofunc 66), Defense Curse 309, Blood Mana 310.
Grid: row x level 1..99 x clvl {1, 30, 60, 99} x (masteries 0 / 50) x (other curse rows at 0 / 20 / 60); then N random
cases (levels 1..99, clvl 1..99, masteries 0..300, 0..8 other skill levels 0..60), xorshift-free: random.Random(SEED).
Limit (as nec_calc.py): the stat getter returns 0 for every stat but clvl and the masteries.
"""
import random
import sys

import nec_calc
import skill_damage

CURSE = [66, 71, 72, 76, 77, 82, 87, 91, 123, 309, 310]


def gen(n, seed):
    rng = random.Random(seed)
    out = []
    for s in CURSE:
        for lvl in range(1, 100):
            for clvl in (1, 30, 60, 99):
                for m in (0, 50):
                    for o in (0, 20, 60):
                        out.append((s, lvl, clvl, [m] * 4, [(t, o) for t in CURSE if o]))
    for _ in range(n):
        s = rng.choice(CURSE)
        pairs = [(rng.choice(CURSE), rng.randint(0, 60)) for _ in range(rng.randint(0, 8))]
        out.append((s, rng.randint(1, 99), rng.randint(1, 99),
                    [rng.choice([0, rng.randint(0, 300)]) for _ in range(4)], pairs))
    for i, (s, lvl, clvl, m, pairs) in enumerate(out):
        print(i, s, lvl, clvl, *m, len(pairs), *(v for p in pairs for v in p))


def main():
    a = sys.argv[1:]
    if a[0] == "gen":
        gen(int(a[1]), int(a[2]))
    elif a[0] == "game":
        o = skill_damage.Oracle()
        with open(a[1]) as f:
            for line in f:
                cid, s, lvl, clvl, m, pairs = skill_damage.parse(line)
                print(cid, *nec_calc.run(o, s, lvl, clvl, m, pairs))


if __name__ == "__main__":
    main()
