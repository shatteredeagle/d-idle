# SPDX-License-Identifier: GPL-3.0-or-later
"""verifyskamazon: the missile hit's element roll, game.exe vs d2d, own harness.

game.exe: FUN_005a8910(unit, minstat, pctstat) with EAX = maxstat reads the two stats (FUN_00625480, hooked), swaps them
if min > max, adds the percent when pctstat != -1 and the pct stat is non-zero (not exercised here: pct stat reads 0),
then FUN_0045c3e0(ecx = unit+0x20 seed {lo, hi}, edx = max - min) + min. Read at 0x5a8910..0x5a8999 and
0x45c3e0..0x45c429. d2d: Rng::range (rules.hpp:169) = min + rng(max - min + 1), same generator.

    tools/emu/run-on-worker.sh vam_roll.py gen N SEED > cases    (lo hi min max; done by vam_roll.sh)
    tools/emu/run-on-worker.sh vam_roll.py game CASES            -> `result lo' hi'`
"""
import random
import sys

import skill_tips

M = 0xFFFFFFFF


def gen(n, seed):
    rng = random.Random(seed)
    for _ in range(n):
        mn = rng.choice([1, 1, 2, 6, 12, 40, rng.randint(1, 400), rng.randint(1, 3000)])
        mx = mn + rng.choice([0, 1, 1, 2, 3, 4, 8, 10, 11, 16, 25, 36, rng.randint(1, 500)])
        print(rng.randrange(1 << 32), rng.choice([666, rng.randrange(1 << 32)]), mn, mx)


def main():
    a = sys.argv[1:]
    if a[0] == "gen":
        gen(int(a[1]), int(a[2]))
        return
    e = skill_tips.boot()
    unit = e.alloc(0x200)
    cur = {}
    e.hook(0x625480, lambda e: cur.get(e.arg(1), 0), 3)
    for line in open(a[1]):
        lo, hi, mn, mx = map(int, line.split())
        cur.clear()
        cur[0x30], cur[0x31] = mn, mx
        e.w32(unit + 0x20, lo)
        e.w32(unit + 0x24, hi)
        r = e.call(0x5A8910, unit, 0x30, 0xFFFFFFFF, regs={"eax": 0x31})
        print(r - (1 << 32) if r >> 31 else r, e.r32(unit + 0x20), e.r32(unit + 0x24))


if __name__ == "__main__":
    main()
