# SPDX-License-Identifier: GPL-3.0-or-later
"""covtreasure2, formulas F17: the player's mana regeneration of one frame, game.exe's FUN_005806f0 run natively as the
oracle for d2d's rules::mana_per_frame plus the cap Fight::apply_regen puts on it.

    covt_regen.py classes                   `C class manaregen` read from game.exe's CharStats table (record +0x3a)
    covt_regen.py CASES SEED OUTFILE        `class maxmana mana bonus flat ::: delta`  (delta = what the frame adds to stat 8)

Hooked (not the claim): FUN_00625480 (stat 8 current mana, 0x1b manarecoverybonus, 0x1a manarecovery fed from the case),
FUN_00625d60 (max mana), FUN_00639df0 / FUN_006256b0 (state 0x55 nomanaregen and the manapot state: off), FUN_006277e0 and
FUN_00626cd0 (the end of the manapot state, never reached), FUN_006272b0 (the stat add: recorded). The division by
ManaRegen x 25 (7500 when 0), the floor of 1, the MulDiv FUN_00483360 and the caps run natively.
"""
import random
import sys

import skill_tips


class Oracle:
    def __init__(self):
        self.e = e = skill_tips.boot()
        self.tables = e.r32(0x744304)
        self.unit = e.alloc(0x200)
        self.mana = self.maxmana = self.bonus = self.flat = 0
        self.added = None
        e.hook(0x625480, self.stat, 3)
        e.hook(0x625d60, lambda e: self.maxmana, 1)
        e.hook(0x639df0, lambda e: 0, 2)
        e.hook(0x6256b0, lambda e: 0, 2)
        e.hook(0x6277e0, lambda e: 0, 2)
        e.hook(0x626cd0, lambda e: 0, 1)
        e.hook(0x6272b0, self.add, 4)

    def stat(self, e):
        s = e.arg(1)
        return {8: self.mana, 0x1b: self.bonus, 0x1a: self.flat}.get(s, 0) & 0xFFFFFFFF

    def add(self, e):
        if e.arg(1) == 8:
            v = e.arg(2)
            self.added = v - (1 << 32) if v & 0x80000000 else v
        return 0

    def regen_seconds(self, cls):
        e = self.tables
        rec = self.e.r32(e + 0xbc4) + cls * 0xc4
        return self.e.read(rec + 0x3a, 1)[0]

    def run(self, cls, maxmana, mana, bonus, flat):
        self.maxmana, self.mana, self.bonus, self.flat, self.added = maxmana, mana, bonus, flat, None
        self.e.w32(self.unit + 4, cls)
        self.e.call(0x5806f0, regs={"edi": self.unit})
        return self.added or 0


def main():
    o = Oracle()
    if sys.argv[1] == "classes":
        for c in range(7):
            print("C", c, o.regen_seconds(c))
        return
    cases, seed, path = int(sys.argv[1]), int(sys.argv[2]), sys.argv[3]
    rng = random.Random(seed)
    with open(path, "w") as f:
        for _ in range(cases):
            cls = rng.randint(0, 6)
            maxm = rng.choice((rng.randint(256, 100 * 256), rng.randint(256, 2000 * 256), rng.randint(0, 600), 0, 3000 * 256 + rng.randint(0, 3)))
            mana = rng.randint(0, maxm)
            if rng.random() < 0.03:
                mana = maxm + rng.randint(1, 5000)
            bonus = rng.choice((0, 0, rng.randint(-90, 500)))
            flat = rng.choice((0, 0, rng.randint(0, 3000), rng.randint(-300, 0)))
            delta = o.run(cls, maxm, mana, bonus, flat)
            f.write(f"{cls} {maxm} {mana} {bonus} {flat} ::: {delta}\n")


if __name__ == "__main__":
    main()
