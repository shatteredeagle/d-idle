# SPDX-License-Identifier: GPL-3.0-or-later
"""skamazon: how many pierces a missile gets, by game.exe's own code (FUN_0059f940).

    python3 ama_pierce.py > pierce.tsv     (worker host, via ama_pierce_run.sh)

FUN_0059f940 (eax = the shooter, [esp+4] = the missile unit) runs when a missile
is made. It gives the missile stat 0x148 (328, pierce_idx) = the number of
pierces it may make, 0..4:
  chance = stat 0xa6 (166, skill_pierce: the Amazon Pierce passive) + stat 0x9c
           (156, item_pierce) of the shooter; 0 -> nothing is set;
  seed   = FUN_00650e40 on the shooter's stat 0x148 (328);
  up to 4 rolls of that seed's FUN_0045c3e0-style step, `roll % 100 < chance`,
  stopping at the first failure; the count of passes is the answer.
Only when the missile's row carries the pierce bit (byte +4 against the mask byte
at 0x6ce270, FUN_0046ace0 hooked to a row with all bits set), shooter type 0 / 1,
missile unit type 3.

Hooked (not the claim): the stat getters FUN_00625480 / 006253b0 / 00625500 and the
setter FUN_00627260 (one dict keyed by (unit, stat)), FUN_0046ace0.
Prints `chance_skill chance_item shooter_stat328 count|none`.
"""
import random

import life_apply

MASK = 0xFFFFFFFF


def main():
    g = life_apply.Game()
    e = g.e
    row = e.alloc(0x100)
    e.mu.mem_write(row + 4, b"\xff")
    owner = g.attacker                       # unit type 0 (a player)
    missile = e.alloc(0x100)
    e.w32(missile, 3)
    e.w32(missile + 4, 5)

    def getter(emu):
        v = g.stats.get((emu.arg(0), emu.arg(1)), 0)
        return v & MASK
    for addr in (0x6253b0, 0x625500):
        e.hook(addr, getter, 3)
    e.hook(0x46ace0, lambda emu: row, 0)
    rng = random.Random(7)
    cases = []
    seeds = [0, 1, 2, 3, 100, 12345, 2 ** 31, 77777777]
    for c in range(0, 111):
        for item in (0, 20):
            for s in seeds:
                cases.append((c, item, s))
    for _ in range(5000):
        cases.append((rng.randint(0, 120), rng.choice((0, 0, rng.randint(0, 60))), rng.getrandbits(32)))
    for c, item, s in cases:
        g.stats.clear()
        g.stats[(owner, 0xa6)] = c
        g.stats[(owner, 0x9c)] = item
        g.stats[(owner, 0x148)] = s
        e.call(0x59f940, missile, regs={"eax": owner})
        got = g.stats.get((missile, 0x148))
        print(c, item, s, "none" if got is None else got)


if __name__ == "__main__":
    main()
