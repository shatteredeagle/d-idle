# SPDX-License-Identifier: GPL-3.0-or-later
"""game.exe's real populations against d2d's monster level (card nmlevel).

  python nm_pop.py cases units.txt > cases.tsv    # plain units of ls_area.py's `units` output, as boss_level.py cases (kind p + level id)
  build/tests/test_boss_level --boss-level cases.tsv > d2d.tsv
  python nm_pop.py compare units.txt d2d.tsv      # per difficulty: units, level and experience differences, by class

ls_area.py (the card levelspawn, upstream's room population under game.exe's code) prints a line per monster unit:
`seed S lvl L diff D: class C fF statlvl SL stat12 S12 exp X`; a plain unit has no flag in 0x1e. The unit's level is stat 12,
its experience stat 13. The rule's subject is the level handed to the life / defence / experience rows (statlvl, FUN_00573cb0 0x573f7e);
a unit of a unique's party also gets leveladd +3 and experience x5 from the mods pass (stat 12 above statlvl: counted apart, the
minion rule is another claim), and a class with level 0 is clamped to 1 by d2d (counted apart). Run from tools/emu with D2_MPQ_DIR set and boss_level.py on PYTHONPATH (it reads the MonStats ids).
"""
import re
import sys
from collections import Counter, defaultdict

UNIT = re.compile(r"seed (\d+) lvl (\d+) diff (\d+): class (\d+) f([0-9a-f]+) statlvl (-?\d+|None) stat12 (-?\d+|None) exp (-?\d+)")


def units(path):
    for line in open(path):
        m = UNIT.match(line)
        if m and not int(m.group(5), 16) & 0x1e and m.group(7) != "None":
            yield tuple(int(m.group(i), 16 if i == 5 else 10) for i in (1, 2, 3, 4, 5, 6, 7, 8))


def cases(path):
    import boss_level
    ids = boss_level.monstats_ids()
    for i, (seed, lid, diff, cls, _, _, _, _) in enumerate(units(path)):
        print(i, ids[cls], cls, diff, "p", "-", seed, lid)


def compare(units_path, d2d_path):
    rows = list(units(units_path))
    d2d = {}
    for line in open(d2d_path):
        f = line.split()
        if len(f) >= 10 and f[0].isdigit() and f[1] == "|" and f[2] != "no-type": d2d[int(f[0])] = f[2:]   # log lines share the stream
    total, level, exp, skipped, raised, zero = Counter(), Counter(), Counter(), Counter(), Counter(), Counter()
    gap, by_class = defaultdict(list), defaultdict(Counter)
    sums = defaultdict(lambda: [0, 0])
    for i, (seed, lid, diff, cls, _, statlvl, level12, xp) in enumerate(rows):
        if i not in d2d:
            skipped[diff] += 1
            continue
        if statlvl == 0:                        # d2d clamps a level-0 class to 1
            zero[diff] += 1
            continue
        if level12 != statlvl:                  # a unique's party: leveladd +3, experience x5
            raised[diff] += 1
            continue
        total[diff] += 1
        got_level, got_exp = int(d2d[i][0]), int(d2d[i][3])
        sums[diff][0] += xp
        sums[diff][1] += got_exp
        if got_level != statlvl:
            level[diff] += 1
            gap[diff].append(statlvl - got_level)
            by_class[diff][cls] += 1
        if got_exp != xp: exp[diff] += 1
    for diff in sorted(total):
        g = gap[diff]
        mean = sum(g) / len(g) if g else 0
        print(f"difficulty {diff}: {total[diff]} plain units ({raised[diff]} of a unique's party and {zero[diff]} of a level-0 class apart, {skipped[diff]} of a class d2d has no row for), "
              f"level differs {level[diff]} ({100 * level[diff] / total[diff]:.1f} %), game minus d2d level mean over those {mean:+.1f}, experience differs {exp[diff]}, "
              f"mean experience game {sums[diff][0] / total[diff]:.1f} d2d {sums[diff][1] / total[diff]:.1f}")
        print("  classes with the most units at another level:", ", ".join(f"class {c} x{n}" for c, n in by_class[diff].most_common(6)))


if __name__ == "__main__":
    {"cases": lambda: cases(sys.argv[2]), "compare": lambda: compare(sys.argv[2], sys.argv[3])}[sys.argv[1]]()
