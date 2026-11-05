# SPDX-License-Identifier: GPL-3.0-or-later
"""COVMONSKILLS: the calc cells AND the skill's own physical min / max (game.exe FUN_00647bc0 / FUN_00647d00, 256ths,
as skbarb_calc.py) of the 147 monster / common rows vs d2d's eval_calc / skill_phys (skbarb_test --calc).

    python3 cms_phys.py gen N SEED > cases.tsv       (cms_calc.gen: same grid and random cases)
    python3 cms_phys.py game cases.tsv > game.tsv
    skbarb_test --calc cases.tsv > d2d.tsv
Output columns: the 25 of cms_show.py then phys_min phys_max (weapon share off on both sides)."""
import sys

import cms_calc
import skbarb_calc
import skill_damage


def main():
    a = sys.argv[1:]
    if a[0] == "gen":
        cms_calc.gen(int(a[1]), int(a[2]))
    elif a[0] == "game":
        o = skill_damage.Oracle()
        with open(a[1]) as f:
            for line in f:
                cid, s, lvl, clvl, m, pairs = skill_damage.parse(line)
                print(cid, *skbarb_calc.run(o, s, lvl, clvl, m, pairs))


if __name__ == "__main__":
    main()
