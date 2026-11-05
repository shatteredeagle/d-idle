# SPDX-License-Identifier: GPL-3.0-or-later
"""Check the monster-level rule against `ls_area.py ... units` output (agent LEVELSPAWN).

  python3 ls_rule.py levels.txt monstats.txt units.txt

Rule under test (read at FUN_00573cb0 0x573e48..0x573e70, docs/research/pr-mods-proof.md): the level handed to the life /
defence / experience rows is the MonStats Level / Level(N) / Level(H) of the class by difficulty in a Normal game, and in an
expansion Nightmare / Hell game the AREA level (Levels.txt MonLvl2Ex / MonLvl3Ex) unless the class is an exception.
For every unit: expected = the MonStats column (diff 0) or the area Ex column (diff 1, 2); a unit whose statlvl differs is an
exception: printed per class with its MonStats boss flag and Level(N) / Level(H).
Also tests the Normal rule against Levels.txt: share of Normal units whose statlvl equals the area's MonLvl1Ex (it is not a rule).
"""
import collections
import csv
import re
import sys


def table(path):
    rows = list(csv.reader(open(path), delimiter="\t"))
    ix = {n: i for i, n in enumerate(rows[0])}
    return rows[1:], ix


def main():
    lrows, lix = table(sys.argv[1])
    mrows, mix = table(sys.argv[2])
    area = {int(r[lix["Id"]]): r for r in lrows if r and r[lix["Id"]].isdigit()}
    mon = {int(r[mix["hcIdx"]]): r for r in mrows if r and r[mix["hcIdx"]].isdigit()}
    cols = ["Level", "Level(N)", "Level(H)"]
    ex = ["MonLvl1Ex", "MonLvl2Ex", "MonLvl3Ex"]
    tot = collections.Counter()
    exc = collections.defaultdict(collections.Counter)
    normal_eq_area = collections.Counter()
    gap = collections.defaultdict(list)
    for line in open(sys.argv[3]):
        m = re.match(r"seed (\d+) lvl (\d+) diff (\d): class (\d+) f(\w+) statlvl (\S+) stat12 (\S+)", line)
        if not m: continue
        lid, d, cls, flags = int(m.group(2)), int(m.group(3)), int(m.group(4)), int(m.group(5), 16)
        if m.group(6) == "None": continue
        sl = int(m.group(6))
        r = mon[cls]
        mcol = int(r[mix[cols[d]]] or 0)
        acol = int(area[lid][lix[ex[d]]] or 0)
        want = mcol if d == 0 else acol
        tot[d] += 1
        if flags & 0x1c == 0:                                  # plain monsters: d2d's level (type_info.level[difficulty]) vs game.exe's
            gap[(lid, d)].append(sl - mcol)
        if d == 0 and sl == acol: normal_eq_area["eq"] += 1
        if sl != want:
            exc[(d, cls)][(lid, sl, want)] += 1
    print("units per difficulty", dict(tot))
    print("Normal units whose level equals the area's MonLvl1Ex:", normal_eq_area["eq"], "of", tot[0])
    for d in (0, 1, 2):
        n = sum(sum(c.values()) for (dd, _), c in exc.items() if dd == d)
        print(f"diff {d}: {n} of {tot[d]} units differ from the rule")
    print("plain monsters: game.exe level minus d2d's (MonStats Level / Level(N) / Level(H)), per level and difficulty")
    print("lid\tname\tdiff\tunits\tdiffer\tmin\tmean\tmax")
    for (lid, d), v in sorted(gap.items()):
        print(f"{lid}\t{area[lid][lix['Name']]}\t{d}\t{len(v)}\t{sum(1 for x in v if x)}\t{min(v)}\t{sum(v) / len(v):.1f}\t{max(v)}")
    for d in (0, 1, 2):
        v = [x for (l, dd), vs in gap.items() if dd == d for x in vs]
        print(f"all plain diff {d}: {len(v)} units, {sum(1 for x in v if x)} differ, mean {sum(v) / len(v):.1f}, min {min(v)}, max {max(v)}")
    print("class\tname\tboss\tprimeevil\tLevel\tLevel(N)\tLevel(H)\tdiff\tunits\t(lid, statlvl, rule) examples")
    for (d, cls), c in sorted(exc.items()):
        r = mon[cls]
        print(f"{cls}\t{r[mix['Id']]}\t{r[mix['boss']]}\t{r[mix['primeevil']]}\t{r[mix['Level']]}\t{r[mix['Level(N)']]}\t{r[mix['Level(H)']]}\t{d}\t{sum(c.values())}\t{sorted(c.items())[:3]}")


main()
