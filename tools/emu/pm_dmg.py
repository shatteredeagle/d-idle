# SPDX-License-Identifier: GPL-3.0-or-later
"""decomp13 PALSKILL matrix: element min / max (256ths -> points) of Holy Bolt 101 and Holy Fire 102 at skill levels 1..8 and no synergy,
game.exe's own FUN_00644d50 / e40 (via skill_damage.Oracle, the same oracle as pal_dmg.py). Prints `id lvl min max len` in points."""
import skill_damage

o = skill_damage.Oracle()
for s, name in ((101, "Holy Bolt"), (102, "Holy Fire")):
    for lvl in range(1, 9):
        r = o.run(s, lvl, 12, [0, 0, 0, 0], [])
        print(name, "lvl", lvl, "raw", r)
