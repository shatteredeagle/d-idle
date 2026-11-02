#!/usr/bin/env python3
"""verifynmlevel: compare the rule grid (vnl_grid.py, game.exe) with d2d base and head (vnl_d2d G mode).

    python3 vnl_gridcmp.py GAMEGRID BASEGRID HEADGRID [--list N]

Counts per difficulty and per kind of row (ratio, noRatio, boss) and per kind of level id (0, a town, 1..136
not a town, past the table), the cases where d2d's start level differs from game.exe's. d2d clamps a level
under 1 to 1 (monster_stats); game.exe stores FUN_0061dca0's value as it is, so a 0 there is listed apart.
"""
import collections
import sys

TOWNS = {1, 40, 75, 103, 109}


def lid_kind(lid):
    return "lid0" if lid == 0 else "town" if lid in TOWNS else "past" if lid > 136 else "level"


def main():
    game = {}
    for l in open(sys.argv[1]):
        p = l.split()
        if p[0] == "G": game[(int(p[1]), int(p[2]), int(p[3]))] = (int(p[4]), int(p[5]), int(p[6]))
    nlist = int(sys.argv[sys.argv.index("--list") + 1]) if "--list" in sys.argv else 0
    for name, path in (("base", sys.argv[2]), ("head", sys.argv[3])):
        c = collections.Counter()
        shown = collections.Counter()
        for l in open(path):
            p = l.split()
            if p[0] != "G": continue
            key = (int(p[1]), int(p[2]), int(p[3]))
            col, flags, start = game[key]
            row = "boss" if flags & 0x40 else "noRatio" if flags & 4 else "ratio"
            k = (key[1], row, lid_kind(key[2]))
            c[k + ("cases",)] += 1
            if p[4] == "no-type": c[k + ("notype",)] += 1; continue
            got = int(p[4])
            if got != start:
                zero = start < 1 and got == max(start, 1)
                c[k + ("clamp0" if zero else "diff",)] += 1
                if not zero and shown[k] < nlist:
                    shown[k] += 1
                    print(f"  {name} cls {key[0]} diff {key[1]} lid {key[2]} ({row}): game {start} (col {col}), d2d {got}")
        print(f"== {name}")
        for k in sorted({k[:3] for k in c}):
            print(f"{name} diff {k[0]} {k[1]:8} {k[2]:6} cases {c[k + ('cases',)]:6} differ {c[k + ('diff',)]:6} clamp0 {c[k + ('clamp0',)]:5} notype {c[k + ('notype',)]}")


if __name__ == "__main__":
    main()
