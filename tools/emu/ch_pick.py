#!/usr/bin/env python3
"""decomp11 (fid-chain-lightning, Q2/D4): game.exe's hop pick, run natively, for the d2d side to be compared byte for byte.

FUN_0056bd10 (Chain Lightning's next-target search, flags 0x8a783) gathers candidates through FUN_0056b7e0 and hands each unit to its
callback 0x56bcd0 (ecx = a context whose word +0xc points at the result block, edx = unit or 0). The result block FUN_0056bd10 builds
(0x56bd49..0x56bd5e): +0 best-above unit (0), +4 its id (0xffffffff), +8 lowest-at-or-below unit (0), +0xc its id (= the struck id),
+0x10 the struck id, +0x14 count (0). Result = +0 if set, else +8 (0x56bd6a..0x56bd71).
This script runs the callback 0x56bcd0 ITSELF (V run) on random unit-id sets in a random order, then reads the result as 0x56bd6a does,
and prints  `case_no picked_id|-`  (the id at unit+0xc). Not run: the room walk FUN_0056b7e0 and the filter FUN_0056b3e0 (which units
reach the callback): see the trace, V read only.

    ch_pick.py gen N SEED > cases        case: `no last n id1..idn`
    ch_pick.py game cases > game.out
The d2d side is ch_pick_test.cpp (rules::ring_pick in components/rules/rules.hpp, compiled as it is).
"""
import random
import sys

M32 = 0xFFFFFFFF


def gen(n, seed):
    rng = random.Random(seed)
    for i in range(n):
        cnt = rng.choice([0, 1, 1, 2, 3, 4, 6, 10, 25])
        pool = rng.choice([40, 300, 70000, 1 << 31])     # unit ids are small sequential numbers; d2d's int ids < 2^31 (ring_pick tests `above < 0`)
        ids = rng.sample(range(0, pool), cnt) if cnt else []
        mode = rng.random()
        if ids and mode < 0.55:
            last = rng.choice(ids)                    # the struck unit is among the found ones (the usual case)
        elif ids and mode < 0.8:
            last = rng.choice(ids) + rng.choice([-1, 1])
        else:
            last = rng.randrange(0, min(pool, 1 << 32))
        last &= M32
        ids = [x & M32 for x in ids]
        if rng.random() < 0.3:
            ids.sort()
        elif rng.random() < 0.3:
            ids.sort(reverse=True)
        print(i, last, len(ids), *ids)


def game(path):
    import emu
    e = emu.Emu()
    cb = 0x56bcd0
    blk = e.alloc(0x40)
    ctx = e.alloc(0x40)
    e.w32(ctx + 0xc, blk)
    units = {}

    def unit(uid):
        if uid not in units:
            u = e.alloc(0x40)
            e.w32(u + 0xc, uid)
            units[uid] = u
        return units[uid]

    byptr = {}
    for line in open(path):
        f = line.split()
        if not f:
            continue
        no, last, n = int(f[0]), int(f[1]), int(f[2])
        ids = [int(x) for x in f[3:3 + n]]
        e.w32(blk + 0, 0); e.w32(blk + 4, M32); e.w32(blk + 8, 0)
        e.w32(blk + 0xc, last); e.w32(blk + 0x10, last); e.w32(blk + 0x14, 0)
        for uid in ids:
            u = unit(uid)
            byptr[u] = uid
            e.call(cb, ecx=ctx, edx=u)
        if f and (no % 3 == 0):
            e.call(cb, ecx=ctx, edx=0)                # the null call (end of the walk?): must change nothing
        p = e.r32(blk + 0) or e.r32(blk + 8)
        cnt = e.r32(blk + 0x14)
        print(no, byptr[p] if p else '-', cnt)


if __name__ == '__main__':
    if sys.argv[1] == 'gen':
        gen(int(sys.argv[2]), int(sys.argv[3]))
    else:
        game(sys.argv[2])
