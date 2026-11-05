# SPDX-License-Identifier: GPL-3.0-or-later
"""COVMONSKILLS: table columns of the 147 monster / common skill rows as game.exe's loader holds them (the
Skills record in emulated memory, offsets from docs/research/sk-mana-trace.md and the d2d skills.hpp comments) vs
d2d's parsed Skill.  Output per row: id st do p1 p2 p3 mana lvlmana manashift minmana m0 m1 m2 m3
(m* = 1 when srvmissile / a / b / c is set).

    python3 cms_cols.py > game_cols.tsv
    cms_cols_test > d2d_cols.tsv
"""
import skill_damage

IDS = list(range(0, 6)) + list(range(156, 221)) + list(range(281, 357))


def main():
    o = skill_damage.Oracle()
    e = o.e

    def w(r, off):
        return int.from_bytes(e.read(r + off, 2), "little", signed=True)
    for s in IDS:
        r = o.skills + s * skill_damage.RECORD
        print(s, w(r, 0x2c), w(r, 0x2e), w(r, 0x30), w(r, 0x32), w(r, 0x34),
              w(r, 0x18a), w(r, 0x18c), w(r, 0x188), w(r, 0x186),
              *(1 if w(r, off) >= 0 else 0 for off in (0x46, 0x48, 0x4a, 0x4c)))


if __name__ == "__main__":
    main()
