#!/usr/bin/env python3
"""O-part-only case generator for the D1 differential (off-weapon enhanced damage), job shielddiff.

    python shield_diff_gen.py REPS SEED [fold] > cases.tsv
5 off values (0 15 40 100 300) x REPS cases, all tag O, line format of sk_melee2.py.
"fold": the off-weapon ED is added to the damage-percent column (stat 25, which make_fighter adds into phys_pct, the one
percentage the game also adds 17 / 18 to) and off set to 0. That is d2d's input under the PROPOSED rule (phys_pct +=
the other worn ED), so d2d's unchanged make_fighter on the folded file is the "after" count (I: the rule emulated
through the input, not a code change; a weapon_sum fold would be wrong: it applies to the base range only).
"""
import random
import sys

import sk_melee2 as M

if __name__ == "__main__":
    reps, seed = int(sys.argv[1]), int(sys.argv[2])
    fold = len(sys.argv) > 3 and sys.argv[3] == "fold"
    rng = random.Random(seed & M.M32)
    i = 0
    for off in (0, 15, 40, 100, 300):
        for _ in range(reps):
            f = M.dmg_fields(rng); f["off"] = off
            if fold: f["dp"] += off; f["off"] = 0
            print(M.dline("O", i, f)); i += 1
