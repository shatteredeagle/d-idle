#!/usr/bin/env python3
"""verifydmg: MonStats rows whose A1 / A2 / S1 damage range hits FUN_0057b420's clamp, per difficulty.

    python3 -I vd_census.py monstats.txt        (the table from mpq-table.sh monstats; a labrun banner line is skipped)

Clamp (game.exe 0057b48a..0057b4f7, no weapon, stat 111 = 0): lo = 1 if min <= 0; hi = 2 if max <= 1; then in 256ths
lo < 1 -> 256, hi <= lo -> lo + 256. Mean of the game roll = (lo + hi) / 2 (top excluded, 256ths). Base d2d mean:
rng.range(min, max) inclusive whole points = (min + max) / 2, and min when max < min (the author's base edge rows, S).
A mode with a missile in MissA1 / MissA2 / MissS1 is listed apart: a missile's physical is FUN_005a89a0's (no clamp).
"""
import sys

MODES = ("A1", "A2", "S1")
DIFFS = (("N", ""), ("NM", "(N)"), ("H", "(H)"))


def game_mean(mn, mx):
    lo = mn if mn > 0 else 1
    hi = mx if mx > 1 else 2
    if hi <= lo:
        hi = lo + 1
    return (lo + hi) / 2


def base_mean(mn, mx):
    return (mn + mx) / 2 if mn <= mx else mn


def main():
    lines = [ln.rstrip("\n").split("\t") for ln in open(sys.argv[1]) if not ln.startswith("labrun")]
    hdr = lines[0]
    col = {h: i for i, h in enumerate(hdr)}
    rows = [r for r in lines[1:] if len(r) > 10 and r[col["Id"]] and r[col["Id"]] != "Expansion"]
    print("Id\tmode\tdiff\tmin\tmax\tkind\tmissile\tdelta_mean")
    tally = {}
    for r in rows:
        if r[col["enabled"]] != "1":
            continue
        for m in MODES:
            miss = r[col["Miss" + m]]
            for d, suf in DIFFS:
                smin, smax = r[col[f"{m}MinD{suf}"]], r[col[f"{m}MaxD{suf}"]]
                if smin == "" and smax == "":
                    continue
                mn, mx = int(smin or 0), int(smax or 0)
                if mn == 0 and mx == 0:
                    kind = "empty"
                elif mn > mx:
                    kind = "inverted"
                elif mn == mx:
                    kind = "equal"
                elif mn <= 0:
                    kind = "min0"
                else:
                    continue
                delta = game_mean(mn, mx) - base_mean(mn, mx)
                k = (m, d, kind, "missile" if miss else "melee")
                n, s = tally.get(k, (0, 0.0))
                tally[k] = (n + 1, s + delta)
                print(f"{r[col['Id']]}\t{m}\t{d}\t{mn}\t{mx}\t{kind}\t{miss or '-'}\t{delta:+.2f}")
    print("# summary: mode diff kind path rows mean_delta_hp")
    for k in sorted(tally):
        n, s = tally[k]
        print("#", *k, n, f"{s / n:+.2f}")


if __name__ == "__main__":
    main()
