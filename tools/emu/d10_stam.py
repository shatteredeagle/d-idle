# SPDX-License-Identifier: GPL-3.0-or-later
"""decomp10 (TASK card ST08): the player's stamina drain per run frame (FUN_0057f240) and stamina regeneration per frame
(FUN_00580500), game.exe's own code run natively as the oracle for d2d's rules::stamina_drain / stamina_regen
(plus the clamp and the 'stop running at 0' step of World::stamina_frame).

    d10_stam.py classes                      `C class rundrain` (CharStats +0x42) from game.exe's loaded table
    d10_stam.py drain CASES SEED OUTFILE     `class town armor_speed has_armor pct cur ::: ret after`
    d10_stam.py regen CASES SEED OUTFILE     `mode max cur bonus ::: after`
Hooked (not the claim): the stat getters FUN_00625480 / FUN_00625500 (fed from the case), the max stamina FUN_00625db0, the
stat add FUN_006272b0 and set FUN_00627260 (recorded), the town test FUN_00620bb0 + FUN_0061ab00, the body armor lookup
FUN_0063bde0 + the item record FUN_006335f0 (a fake record whose +0xd8 is the armor.txt speed). The CharStats RunDrain read,
the doubling, the speed / 10 + 1, the percent step, the floor of 1 and the stop at 0 run natively.
"""
import random
import sys

import skill_tips


class Oracle:
    def __init__(self):
        self.e = e = skill_tips.boot()
        self.tables = e.r32(0x744304)
        self.unit = e.alloc(0x200)
        self.item = e.alloc(0x40)
        self.rec = e.alloc(0x200)
        self.cur = self.max = self.bonus = self.pct = 0
        self.town = 0
        self.armor = None
        self.e.w32(self.unit + 0x60, self.unit + 0x100)   # the inventory pointer: only tested for non-null
        e.hook(0x625480, self.stat, 3)
        e.hook(0x625500, self.stat, 3)
        e.hook(0x625db0, lambda e: self.max, 1)
        e.hook(0x6272b0, self.add, 4)
        e.hook(0x627260, self.set, 4)
        e.hook(0x620bb0, lambda e: 1, 1)
        e.hook(0x61ab00, lambda e: self.town, 1)
        e.hook(0x63bde0, lambda e: self.item if self.armor is not None else 0, 2)
        e.hook(0x6335f0, lambda e: self.rec, 1)

    def stat(self, e):
        s = e.arg(1)
        return {0xa: self.cur, 0x1c: self.bonus, 0x9a: self.pct}.get(s, 0) & 0xFFFFFFFF

    def add(self, e):
        if e.arg(1) == 0xa:
            v = e.arg(2)
            self.cur += v - (1 << 32) if v & 0x80000000 else v
        return 0

    def set(self, e):
        if e.arg(1) == 0xa:
            v = e.arg(2)
            self.cur = v - (1 << 32) if v & 0x80000000 else v
        return 0

    def rundrain(self, cls):
        rec = self.e.r32(self.tables + 0xbc4) + cls * 0xc4
        return self.e.read(rec + 0x42, 1)[0]

    def drain(self, cls, town, speed, has_armor, pct, cur):
        self.town, self.armor, self.pct, self.cur = town, (speed if has_armor else None), pct, cur
        self.e.w32(self.unit + 4, cls)
        self.e.w32(self.item + 4, 1)
        self.e.w32(self.rec + 0xd8, speed)
        ret = self.e.call(0x57f240, regs={"edi": self.unit}) & 0xFFFFFFFF
        return ret & 0xff if ret < 256 else ret, self.cur

    def regen(self, mode, mx, cur, bonus):
        self.max, self.cur, self.bonus = mx, cur, bonus
        self.e.w32(self.unit + 0x10, mode)
        self.e.call(0x580500, regs={"edi": self.unit})
        return self.cur


def main():
    o = Oracle()
    if sys.argv[1] == "classes":
        for c in range(7):
            print("C", c, o.rundrain(c))
        return
    kind, cases, seed, path = sys.argv[1], int(sys.argv[2]), int(sys.argv[3]), sys.argv[4]
    rng = random.Random(seed)
    with open(path, "w") as f:
        for _ in range(cases):
            if kind == "drain":
                cls = rng.randint(0, 6)
                town = rng.choice((0, 0, 0, 1))
                speed = rng.choice((0, 5, 10, 15, 20, 25, 30, rng.randint(0, 60)))
                has = rng.choice((0, 1, 1))
                pct = rng.choice((0, 0, rng.randint(0, 100), rng.randint(-50, 150)))
                cur = rng.choice((rng.randint(0, 200 * 256), rng.randint(0, 40), rng.randint(0, 4000), 0))
                ret, after = o.drain(cls, town, speed, has, pct, cur)
                f.write(f"{cls} {town} {speed} {has} {pct} {cur} ::: {ret} {after}\n")
            else:
                mode = rng.randint(0, 16) if rng.random() < 0.3 else rng.choice((1, 2, 3, 5, 6))
                mx = rng.choice((rng.randint(256, 400 * 256), rng.randint(0, 600), 0))
                cur = rng.choice((rng.randint(0, mx), rng.randint(0, 600), 0, mx, mx + rng.randint(1, 3000)))
                bonus = rng.choice((0, 0, rng.randint(-90, 500), 1000, rng.randint(900, 1200)))
                after = o.regen(mode, mx, cur, bonus)
                f.write(f"{mode} {mx} {cur} {bonus} ::: {after}\n")


if __name__ == "__main__":
    main()
