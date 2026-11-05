# SPDX-License-Identifier: GPL-3.0-or-later
"""COVMONSKILLS: every calc cell of the 147 monster / common skill rows (Skills.txt charclass blank:
ids 0-5, 156-220, 281-356) by game.exe's own calc machine (FUN_00646ca0, as nec_calc.py) vs d2d's eval_calc.

    python3 cms_calc.py gen N SEED > cases.tsv
    python3 cms_calc.py game cases.tsv > game.tsv       # nec_calc.run: calc1..4, petmax, sumsk, delay, range, len, perdelay, aura, passive
    test_sk_rest_calc --calc cases.tsv > d2d.tsv
Grid: row x level 1..60 x clvl {1, 30, 99} x (Blizzard 59 level 0 / 20 / 60: ShamanFire-style cross-skill cell skill('Blizzard'.blvl)).
Then N random cases (levels 1..99, clvl 1..99, 0..8 other skill levels 0..60 on rows 59, 36..65).
Limit (as nec_calc.py): the stat getter returns 0 for every stat but clvl and the masteries; par1..par7 come from the skill record.
"""
import random
import sys

import nec_calc
import skill_damage

IDS = list(range(0, 6)) + list(range(156, 221)) + list(range(281, 357))
CROSS = [59] + list(range(36, 66))


def gen(n, seed):
    rng = random.Random(seed)
    out = []
    for s in IDS:
        for lvl in range(1, 61):
            for clvl in (1, 30, 99):
                for b in (0, 20, 60):
                    out.append((s, lvl, clvl, [0] * 4, [(59, b)] if b else []))
    for _ in range(n):
        s = rng.choice(IDS)
        pairs = [(rng.choice(CROSS), rng.randint(0, 60)) for _ in range(rng.randint(0, 8))]
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
