#!/usr/bin/env python3
"""D1 differential: game.exe FUN_0057b420 (off-weapon ED at the player's 17 / 18) vs d2d's real make_fighter + player_blow.

    run-on-worker.sh shield_diff_run.py REPS SEED...      (build first: shield_diff_build.sh)
Per seed: cases (5 off values x REPS, tag O) -> game -> d2d as is ("before") -> d2d on the folded file ("after",
the proposed rule emulated through the input) -> sk_melee2.cmp for both. Files under $D2_SCRATCH.
"""
import contextlib
import os
import subprocess
import sys

import sk_melee2 as M

S = os.environ.get("D2_SCRATCH", "/tmp/d2agent-pa-shielddiff")


def gen(reps, seed, fold, path):
    import random
    rng = random.Random(seed & M.M32)      # the same draws as shield_diff_gen.py
    i = 0
    with open(path, "w") as out:
        for off in (0, 15, 40, 100, 300):
            for _ in range(reps):
                f = M.dmg_fields(rng); f["off"] = off
                if fold: f["dp"] += off; f["off"] = 0
                out.write(M.dline("O", i, f) + "\n"); i += 1


def run(reps, seed):
    c, cf, g, d, df = (f"{S}/sd_{n}{seed}.tsv" for n in ("c", "cf", "g", "d", "df"))
    gen(reps, seed, False, c); gen(reps, seed, True, cf)
    print(f"== seed {seed}: {sum(1 for _ in open(c))} cases", flush=True)
    with open(g, "w") as out, contextlib.redirect_stdout(out):
        M.game(c)
    for case, res in ((c, d), (cf, df)):
        with open(res, "w") as out:
            subprocess.run([f"{S}/test_melee", "--melee", case], stdout=out, check=True)
    print("-- before (d2d as is)", flush=True); M.cmp(c, g, d)
    print("-- after (d2d with the off-weapon ED added to phys_pct via stat 25 = proposed rule)", flush=True); M.cmp(c, g, df)
    for tag, fn in (("first differing case before", d),):
        for a, b, x in zip(open(c), open(g), open(fn)):
            ga, da = b.split(), x.split()
            if len(da) > 4 and int(da[2]) and int(ga[2]) >> 8 != (int(da[4]) * 256 + int(da[5])) >> 8:
                print(tag, ":", a.strip(), "| game", ga[2], "| d2d", da[4]); break


if __name__ == "__main__":
    for s in sys.argv[2:]:
        run(int(sys.argv[1]), int(s))
