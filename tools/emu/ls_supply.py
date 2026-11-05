# SPDX-License-Identifier: GPL-3.0-or-later
"""How much XP one fresh game's Act 1 / 2 Normal monsters hold, per character level (agent LEVELSPAWN).

  python3 ls_supply.py experience.txt units.txt [diff]

units.txt is `ls_area.py ... units` output (every monster game.exe made: class, flags, level, experience stat 13 as set at
spawn). The award per kill is game.exe's (docs/research/xp-award-trace.md, V): FUN_0057e2f0 level step (table 0x6e1668 when the
character is at or above the monster, 0x6e1694 below it, clvl >= 25: exp x clvl / mlvl), then Experience.txt ExpRatio / 1024.
Prints, per clvl: the XP of killing EVERYTHING in the sets (outdoor Act 1 areas 2-7; all Act 1 areas run; Act 2 areas run),
mean per seed, and the XP the character needs for the next level (Sorceress column). An upper bound: whole level, no misses.
"""
import collections
import csv
import re
import sys

DOWN = [256, 256, 256, 256, 256, 256, 207, 159, 110, 61, 13]       # 0x6e1668, clvl - mlvl = 0..10
UP = [256, 256, 256, 256, 256, 256, 225, 174, 92, 38, 5]            # 0x6e1694, mlvl - clvl = 0..10


def award(exp, clvl, mlvl, ratio):
    if clvl >= mlvl: a = exp * DOWN[min(clvl - mlvl, 10)] // 256
    elif clvl >= 25: a = exp * clvl // mlvl
    else: a = exp * UP[min(mlvl - clvl, 10)] // 256
    return a * ratio // 1024


def main():
    rows = list(csv.reader(open(sys.argv[1]), delimiter="\t"))
    ix = {n: i for i, n in enumerate(rows[0])}
    thr, ratio = {}, {}
    for r in rows[2:]:
        if r[0].isdigit():
            thr[int(r[0])] = int(r[ix["Sorceress"]])
            ratio[int(r[0])] = int(r[ix["ExpRatio"]])
    diff = int(sys.argv[3]) if len(sys.argv) > 3 else 0
    units = collections.defaultdict(list)                               # (lid, seed) -> [(mlvl, exp)]
    for line in open(sys.argv[2]):
        m = re.match(r"seed (\d+) lvl (\d+) diff (\d): class \d+ f\w+ statlvl (\S+) stat12 \S+ exp (\d+)", line)
        if m and int(m.group(3)) == diff and m.group(4) != "None":
            units[(int(m.group(2)), int(m.group(1)))].append((int(m.group(4)), int(m.group(5))))
    seeds = sorted({s for _, s in units})
    sets = {"Act 1 outdoor (2-7)": range(2, 8),
            "Act 1 all run (2-12,17-19,21-38)": [l for l in range(2, 39) if l not in (13, 14, 15, 16, 20)],
            "Act 2 run (41-44,46-47,49,51,52,55-58,62,65,66)": [41, 42, 43, 44, 46, 47, 49, 51, 52, 55, 56, 57, 58, 62, 65, 66]}
    if len(sys.argv) > 4 and sys.argv[4] == "areas":                    # one column per area instead of the three sets
        sets = {f"area {l}": [l] for l in (2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 17)}
    print("seeds", seeds)
    print("clvl\tneed\t" + "\t".join(sets))
    for c in range(1, 99):
        need = thr[c] - thr[c - 1]                                     # row c = total XP at which clvl c + 1 starts
        cells = []
        for name, lids in sets.items():
            tot = 0
            for s in seeds:
                for l in lids:
                    tot += sum(award(x, c, ml, ratio[c]) for ml, x in units.get((l, s), []))
            cells.append(f"{tot / len(seeds):.0f} ({need / max(tot / len(seeds), 1):.2f} games)")
        print(f"{c}\t{need}\t" + "\t".join(cells))


main()
