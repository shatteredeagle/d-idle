# SPDX-License-Identifier: GPL-3.0-or-later
"""skamazon: the Amazon skill rows' calcs by game.exe's own calc machine.

    python3 ama_calc.py gen N SEED > cases.tsv
    python3 ama_calc.py game cases.tsv > game.tsv
    ama_test --calc cases.tsv > d2d.tsv          # tools/emu/nec_test.cpp (generic by skill id), the real d2d rules
    diff game.tsv d2d.tsv

Same as nec_calc.py (card necskill) with the Amazon rows, Skills 6..35.
Case format = pal_aura.py's (`id skill lvl clvl m_fire m_ltng m_cold m_pois n
(skill level)*n`). `game` prints per case `id` then calc1..4 (+0x138..0x144),
petmax (+0xc0), sumsk1..5 calc (+0xd0, x when no sumskill id +0xc4), delay
(+0x190), aurarange (+0x64), auralen (+0x60), perdelay (+0x128), aurastatcalc1..6
(+0x68, x when no aurastat id +0x54), passivecalc1..5 (+0xa4, x when no
passivestat +0x98). Each by FUN_00646ca0(unit, calc, skill, lvl). Hooks as
skill_damage.py (stat getter: stat 0xc = clvl, 0x149..0x14c = masteries, else 0;
skill record by id; skill level)."""
import random
import sys

import skill_damage

AMA = list(range(6, 36))


def s32(v):
    return v - (1 << 32) if v >> 31 else v


def run(o, skill, lvl, clvl, m, pairs):
    e = o.e
    o.levels = dict(pairs)
    o.levels[skill] = lvl
    o.stats = {0xc: clvl, 0x149: m[0], 0x14a: m[1], 0x14b: m[2], 0x14c: m[3]}
    r = o.skills + skill * skill_damage.RECORD
    out = []

    def calc(off):
        ptr = e.r32(r + off)
        if ptr in (0, 0xffffffff):
            return 0
        return s32(e.call(0x646ca0, o.unit, ptr, skill, lvl))

    def sid(off):
        return int.from_bytes(e.read(r + off, 2), "little", signed=True)
    out += [calc(0x138 + 4 * k) for k in range(4)]
    out.append(calc(0xc0))
    for k in range(5):
        out.append(calc(0xd0 + 4 * k) if sid(0xc4 + 2 * k) >= 0 else "x")
    out += [calc(0x190), calc(0x64), calc(0x60), calc(0x128)]
    for k in range(6):
        out.append(calc(0x68 + 4 * k) if sid(0x54 + 2 * k) >= 0 else "x")
    for k in range(5):
        out.append(calc(0xa4 + 4 * k) if sid(0x98 + 2 * k) >= 0 else "x")
    return out


def gen(n, seed):
    rng = random.Random(seed)
    out = []
    for s in AMA:
        for lvl in range(1, 41):
            out.append((s, lvl, 30, [0, 0, 0, 0], []))
            out.append((s, lvl, 99, [50, 50, 50, 50], [(t, 20) for t in AMA]))
    for _ in range(n):
        s = rng.choice(AMA)
        pairs = [(rng.choice(AMA), rng.randint(0, 30)) for _ in range(rng.randint(0, 8))]
        out.append((s, rng.choice([rng.randint(1, 20), rng.randint(1, 60)]), rng.randint(1, 99),
                    [rng.choice([0, rng.randint(0, 300)]) for _ in range(4)], pairs))
    for i, (s, lvl, clvl, m, pairs) in enumerate(out):
        print(i, s, lvl, clvl, *m, len(pairs), *(v for p in pairs for v in p))


def main():
    a = sys.argv[1:]
    if a[0] == "gen":
        gen(int(a[1]), int(a[2]))
    elif a[0] == "game":
        o = skill_damage.Oracle()
        with open(a[1]) as f:
            for line in f:
                cid, s, lvl, clvl, m, pairs = skill_damage.parse(line)
                print(cid, *run(o, s, lvl, clvl, m, pairs))


if __name__ == "__main__":
    main()
