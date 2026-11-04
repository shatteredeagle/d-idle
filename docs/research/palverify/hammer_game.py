# SPDX-License-Identifier: GPL-3.0-or-later
"""Verifier harness (agent palverify): a Blessed Hammer hit by game.exe's own code.

    python hammer_game.py rows                     # missile rows with a pSrvDmgFunc, the table entry, hammer skill Param1
    python hammer_game.py gen SEED N > cases.tsv   # structured edges, then N random cases
    python hammer_game.py game cases.tsv > game.tsv

Run from a d2d tools/emu directory (imports emu, skill_tips: the image and
the data tables loaded for real). A case is
`id lvl clvl aim vigor pct mon res life seedlo seedhi`: Blessed Hammer's
level, the caster's level, Blessed Aim and Vigor levels (the synergies),
the damagepercent stat of the caster's state 0x2a (Concentration; 0 = no
state), the MonStats row of the defender, its magic resistance, its life
in points, the missile's seed.

Per case, in the order game.exe runs it on a hammer hit:
  1. FUN_00644d50 / FUN_00644e40 (mastery flag 1): the magic min / max.
  2. FUN_006461d0(caster, hammer id) natively (state lookups hooked to the
     case's pct) and, when not 0, FUN_00483360 MulDiv(stat, 100 + it, 100)
     on both stats (as FUN_005d0040 does at 0x5d011a..0x5d0161).
  3. The missile hit (FUN_005ae0d3..0x5ae18e): FUN_005a89a0 rolls the whole
     record on the missile's seed, then the row's pSrvDmgFunc through the
     table at 0x73c960 (FUN_005ad2f0 for the hammer), on the same record.
  4. FUN_0057c6c0 applies the record (resistance, life), as missile_hit.py.
Output `id lo hi life` (the min / max after step 2, 256ths; life after, 256ths).
PV_DIRECT=1 skips FUN_005a89a0 (the dmg func's own roll is then the only draw).
To stderr: on how many cases the dmg func changed the field FUN_005a89a0 rolled
(its FUN_005a8c70 re-rolls the element on the same seed).

Hooked (not the claim): the stat getter / setter (FUN_00625480, FUN_00627260,
FUN_006253b0), the unit's skill record and level (FUN_00643810 / FUN_006442a0),
the state plumbing FUN_006461d0 reads (FUN_00639df0 has-state, FUN_006256b0 the
state's stat list, FUN_00625d00 a stat of it), max life and the difficulty row,
and the side effects life_apply.py stubs, EXCEPT FUN_0063e990 (undead) and
FUN_0063e940 (demon), which run natively on the real MonStats rows.
"""
import os
import random
import sys

from unicorn.x86_const import UC_X86_REG_ESI

import skill_tips

MASK = 0xFFFFFFFF
HAMMER, AIM, VIGOR = 112, 108, 115
FAKE = 0x30000000
STATLIST = 0x31000000
DIRECT = os.environ.get("PV_DIRECT") == "1"


def s32(v):
    v &= MASK
    return v - (1 << 32) if v >> 31 else v


class Game:
    def __init__(self):
        e = self.e = skill_tips.boot()
        t = self.tables = e.r32(0x744304)
        self.skills, self.nskills = e.r32(t + 0xb98), e.r32(t + 0xba0)
        self.missiles, self.nmissiles = e.r32(t + 0xb64), e.r32(t + 0xb6c)
        self.nmon = e.r32(t + 0xa80)
        self.stats, self.levels, self.pct = {}, {}, 0
        self.game = e.alloc(0x100)
        g = e.view("game", self.game)
        g.difficulty, g.expansion = 1, 1
        self.caster = e.alloc(0x200)                     # type 0: a player
        self.missile = e.alloc(0x200)
        self.target = e.alloc(0x200)
        self.rec = e.alloc(0x100)
        self.diffrow = e.alloc(0x40)
        self.max_life = 0
        e.w32(self.missile, 3)
        e.w32(self.target, 1)
        e.hook(0x625480, lambda e: self.stats.get((e.arg(0), e.arg(1)), 0) & MASK, 3)
        e.hook(0x6253b0, lambda e: self.stats.get((e.arg(0), e.arg(1)), 0) & MASK, 3)
        e.hook(0x627260, lambda e: self.stats.__setitem__((e.arg(0), e.arg(1)), s32(e.arg(2))) or 0, 4)
        e.hook(0x625d10, lambda e: self.max_life, 1)
        e.hook(0x611d30, lambda e: self.diffrow, 1)
        e.hook(0x643810, lambda e: FAKE + 0x40 * (e.mu.reg_read(UC_X86_REG_ESI) & 0xffff), 0)
        e.hook(0x6442a0, lambda e: self.levels.get((e.arg(1) - FAKE) // 0x40, 0), 3)
        self.state_calls = []
        e.hook(0x639df0, self.has_state, 2)
        e.hook(0x6256b0, lambda e: STATLIST if e.arg(0) == self.caster and e.arg(1) == 0x2a and self.pct else 0, 2)
        e.hook(0x625d00, lambda e: (self.pct & MASK) if e.arg(0) == STATLIST and e.arg(1) == 0x19 else 0, 3)
        # life_apply.py's side-effect stubs, minus 0x63e990 / 0x63e940 (run for real) and 0x639df0 (hooked above)
        e.stub(0, {0x63ee90: 1, 0x63edc0: 1, 0x63e9f0: 2, 0x5c0c30: 3, 0x5347d0: 7,
                   0x5347e0: 3, 0x61ab00: 1, 0x5541b0: 0, 0x621d50: 2, 0x57aa60: 0, 0x57aaa0: 0,
                   0x57aae0: 2, 0x57af80: 2, 0x57b230: 2, 0x57ac50: 3, 0x57add0: 3, 0x5d6410: 0})
        e.stub(1, {0x554200: 1, 0x620bb0: 1, 0x457490: 0})
        rows = [r for r in self.dmg_rows() if r[1] == 5]
        assert len(rows) == 1, rows                      # one row with pSrvDmgFunc 5 (the claim)
        self.hammer_missile = rows[0][0]

    def has_state(self, e):
        self.state_calls.append((e.arg(0), e.arg(1)))
        return 1 if e.arg(0) == self.caster and e.arg(1) == 0x2a and self.pct else 0

    def dmg_rows(self):
        e, out = self.e, []
        for i in range(self.nmissiles):
            r = self.missiles + i * 0x1a4
            f = e.r16(r + 0x10)
            if f:
                out.append((i, f, s32(e.r32(r + 0x78)), s32(e.r32(r + 0x7c))))
        return out

    def flags(self, mon):
        return self.e.call(0x63e990, self.target_for(mon)), self.e.call(0x63e940, self.target_for(mon))

    def target_for(self, mon):
        self.e.w32(self.target + 4, mon)
        return self.target

    def run(self, lvl, clvl, aim, vigor, pct, mon, res, life, seed):
        e = self.e
        self.stats.clear()
        self.levels = {HAMMER: lvl, AIM: aim, VIGOR: vigor}
        self.pct = pct
        self.stats[(self.caster, 0xc)] = clvl
        lo = s32(e.call(0x644d50, self.caster, HAMMER, lvl, 1))
        hi = s32(e.call(0x644e40, self.caster, HAMMER, lvl, 1))
        infl = s32(e.call(0x6461d0, self.caster, HAMMER))
        if infl:
            lo = s32(e.call(0x483360, 100, ecx=lo & MASK, edx=(infl + 100) & MASK))
            hi = s32(e.call(0x483360, 100, ecx=hi & MASK, edx=(infl + 100) & MASK))
        # the hit
        self.stats.clear()
        self.max_life = life << 8
        tgt = self.target_for(mon)
        self.stats[(tgt, 6)] = life << 8
        self.stats[(tgt, 7)] = life << 8
        self.stats[(tgt, 0x25)] = res
        self.stats[(self.missile, 0x34)] = lo
        self.stats[(self.missile, 0x35)] = hi
        e.w32(self.missile + 4, self.hammer_missile)
        e.w32(self.missile + 0x20, seed[0])
        e.w32(self.missile + 0x24, seed[1])
        e.mu.mem_write(self.rec, b"\0" * 0x100)
        if not DIRECT:                                   # PV_DIRECT=1: skip the hit path's first roll (FUN_005a89a0)
            e.call(0x5a89a0, self.missile, tgt, self.rec, ecx=self.game, edx=0)
        first = s32(e.r32(self.rec + 0x20))
        f = e.r16(self.missiles + self.hammer_missile * 0x1a4 + 0x10)
        fn = e.r32(0x73c960 + 4 * f)
        e.call(fn, tgt, self.rec, ecx=self.game, edx=self.missile)
        final = s32(e.r32(self.rec + 0x20))
        e.w32(self.rec, 1)
        e.w32(self.rec + 4, 1)
        e.call(0x57c6c0, tgt, 1, self.rec, ecx=self.game, edx=self.caster)
        after = self.stats.get((tgt, 6), 0)
        return lo, hi, max(after, 0) if after > 0 else 0, first, final, infl


def gen(seed, n, nmon):
    rng = random.Random(seed)
    out = []
    pcts = (0, 60, 75, 105, 165, 225, 285, 345, 405, 555)
    edge_pct = (-100, -60, -8, -1, 1, 7, 8, 9, 1000, 25000, 400000)
    res_set = (-100, -50, -1, 0, 1, 25, 50, 75, 99, 100, 101, 150)

    def seedpair():
        return rng.getrandbits(32), rng.getrandbits(32)

    # every monster row at a few settings
    for mon in range(nmon):
        for pct in (0, 345):
            out.append((rng.randint(1, 30), rng.randint(1, 99), rng.choice((0, 5, 20)), rng.choice((0, 5, 20)), pct, mon, rng.choice((0, 0, 25, 75, 100)), rng.choice((1, 50, 500, 5000)), seedpair()))
    # edges: concentration pct, hammer level, synergies, resistance, life 1
    for pct in edge_pct + pcts:
        for lvl in (1, 2, 20, 40, 60, 99):
            for res in res_set:
                mon = rng.randrange(nmon)
                out.append((lvl, rng.randint(1, 99), rng.choice((0, 20, 99)), rng.choice((0, 20, 99)), pct, mon, res, rng.choice((1, 2, 3000)), seedpair()))
    for _ in range(n):
        out.append((rng.choice((rng.randint(1, 20), rng.randint(1, 45))), rng.randint(1, 99), rng.choice((0, rng.randint(0, 30))), rng.choice((0, rng.randint(0, 30))),
                    rng.choice((0, rng.choice(pcts), rng.randint(-200, 2000))), rng.randrange(nmon), rng.choice((rng.randint(-100, 99), 0, 100, rng.randint(100, 150))),
                    rng.randint(1, 6000), seedpair()))
    for i, (lvl, clvl, aim, vig, pct, mon, res, life, sd) in enumerate(out):
        print(i, lvl, clvl, aim, vig, pct, mon, res, life, sd[0], sd[1])


def main():
    a = sys.argv[1:]
    if not a:
        raise SystemExit(__doc__)
    g = Game()
    if a[0] == "rows":
        print("dmg rows (index, func, dParam1, dParam2):", g.dmg_rows())
        print("table entry 5:", hex(g.e.r32(0x73c960 + 20)))
        print("hammer Param1 (+0x148):", s32(g.e.r32(g.skills + HAMMER * 0x23c + 0x148)))
        print("monstats rows:", g.nmon)
        und = [m for m in range(g.nmon) if g.flags(m)[0]]
        dem = [m for m in range(g.nmon) if g.flags(m)[1]]
        both = sorted(set(und) & set(dem))
        print("undead", len(und), "demon", len(dem), "both", len(both), both[:20])
        for m in range(g.nmon):
            u, d = g.flags(m)
            print("flag", m, u, d)
        return
    if a[0] == "gen":
        gen(int(a[1]), int(a[2]), g.nmon)
        return
    rerolled = 0
    for line in open(a[1]):
        f = [int(x) for x in line.split()]
        lo, hi, life, first, final, infl = g.run(*f[1:9], (f[9] & MASK, f[10] & MASK))
        rerolled += first != final
        print(f[0], lo, hi, life)
    print(f"# field changed by the dmg func on {rerolled} cases", file=sys.stderr)


if __name__ == "__main__":
    main()
