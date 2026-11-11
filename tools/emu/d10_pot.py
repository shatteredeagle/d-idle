# SPDX-License-Identifier: GPL-3.0-or-later
"""decomp10 (card ST05, potions): the arithmetic pieces of FUN_005be3f0 (the healing / mana potion effect) as game.exe's own
code, run natively, against d2d's rules::potion_amount:
  - the class bonus on the amount: FUN_0062a5d0 (life, stat hpregen) and FUN_0062a620 (mana, stat manarecovery), called with a
    fake unit {type 0, class c} (or type != 0: a monster, a pet),
  - the double-fill roll: FUN_0045c3e0 (rand n on the unit seed at unit+0x20) for the vitality / energy total, shifted right 1,
    then FUN_0045c390 for rand(100); the amount doubles when rand(100) < half, in that order (read at 0x5be525..0x5be542).
    The composition around the two primitives is transcribed from the disassembly (V read); the primitives are V run.

    d10_pot.py CASES SEED OUTFILE      `class life calc stat low high ::: amount low' high'`  (life 1 = healing, 0 = mana)
"""
import random
import sys

import skill_tips


class Oracle:
    def __init__(self):
        self.e = e = skill_tips.boot()
        self.unit = e.alloc(0x100)

    def bonus(self, fn, cls, typ, v):
        e = self.e
        e.w32(self.unit, typ)
        e.w32(self.unit + 4, cls)
        return e.call(fn, self.unit, v & 0xFFFFFFFF) & 0xFFFFFFFF

    def amount(self, cls, life, calc, stat, low, high):
        e = self.e
        v = self.bonus(0x62a5d0 if life else 0x62a620, cls, 0, calc << 8)
        if v & 0x80000000:
            v -= 1 << 32
        e.w32(self.unit + 0x20, low)
        e.w32(self.unit + 0x24, high)
        if stat > 0:
            half = (e.call(0x45c3e0, stat, ecx=self.unit + 0x20) & 0xFFFFFFFF) >> 1
            if (e.call(0x45c390, 100, ecx=self.unit + 0x20) & 0xFFFFFFFF) < half:
                v *= 2
        return v, e.r32(self.unit + 0x20), e.r32(self.unit + 0x24)


def main():
    o = Oracle()
    if sys.argv[1] == "classes":
        for c in range(7):
            print("C", c, [o.bonus(0x62a5d0, c, 0, 256) for _ in (0,)][0] / 256, o.bonus(0x62a620, c, 0, 256) / 256)
        print("C monster", o.bonus(0x62a5d0, 0, 1, 256) / 256, o.bonus(0x62a620, 0, 1, 256) / 256)
        return
    cases, seed, path = int(sys.argv[1]), int(sys.argv[2]), sys.argv[3]
    rng = random.Random(seed)
    with open(path, "w") as f:
        for _ in range(cases):
            cls = rng.randint(0, 6)
            life = rng.randint(0, 1)
            calc = rng.choice((30, 60, 100, 180, 320, 20, 40, 80, 150, 250, rng.randint(1, 400)))
            stat = rng.choice((0, rng.randint(1, 60), rng.randint(1, 400), 128, 150, 256, 100))
            low, high = rng.getrandbits(32), rng.choice((666, rng.getrandbits(32)))
            amt, lo2, hi2 = o.amount(cls, life, calc, stat, low, high)
            f.write(f"{cls} {life} {calc} {stat} {low} {high} ::: {amt} {lo2} {hi2}\n")


if __name__ == "__main__":
    main()
