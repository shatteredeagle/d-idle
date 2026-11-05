# SPDX-License-Identifier: GPL-3.0-or-later
"""covtreasure2, formulas F10: the per-hit experience award, game.exe's level-gap step FUN_0057e2f0 followed by the
Experience.txt ratio step FUN_0057e390 (the two steps of FUN_0057e480 that carry the arithmetic), run natively with the
real table getter FUN_00613e60 (no hook), as the oracle for d2d's rules::kill_award.

    covt_xp.py CASES SEED OUTFILE          one line per case: `exp clvl mlvl award`
    covt_xp.py ratios OUTFILE              `R level ratio` for level 0..99 (level 0 = the shift row), game.exe's getter

Cases: exp mixes small values (1..5000), the monster-table range (1..0x7fffff) and edge values; clvl 1..98 (99 is the
MaxLvl gate FUN_0057e480 reads before these steps, not run); mlvl 1..110 half at random, half within +-12 of clvl.
Not covered (V read only): the 0x7fffff cap on exp, the exp <= 0 return 1, the MaxLvl return 0, the item_addexperience
percentage, the merc cap.
"""
import random
import sys

import drlg


def s32(v):
    return v - (1 << 32) if v >= 1 << 31 else v


def main():
    e = drlg.boot()
    if sys.argv[1] == "ratios":
        with open(sys.argv[2], "w") as f:
            for level in range(0, 100):
                f.write(f"R {level} {s32(e.call(0x613e60, level))}\n")
        return
    cases, seed, path = int(sys.argv[1]), int(sys.argv[2]), sys.argv[3]
    rng = random.Random(seed)
    edges = (1, 2, 7, 18, 0x7fffff, 0x7ffffe, 0x100000, 0x100001)
    with open(path, "w") as f:
        for _ in range(cases):
            exp = rng.choice((rng.randint(1, 5000), rng.randint(1, 0x7fffff), rng.choice(edges)))
            clvl = rng.randint(1, 98)
            mlvl = rng.randint(1, 110)
            if rng.random() < 0.5:
                mlvl = max(1, min(110, clvl + rng.randint(-12, 12)))
            step = e.call(0x57e2f0, ecx=exp, edx=clvl, regs={"eax": mlvl})
            award = e.call(0x57e390, clvl, regs={"eax": step})
            f.write(f"{exp} {clvl} {mlvl} {s32(award)}\n")


if __name__ == "__main__":
    main()
