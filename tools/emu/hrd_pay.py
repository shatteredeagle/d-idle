# SPDX-License-Identifier: GPL-3.0-or-later
"""hirereader: game.exe 1.14d's merc payment as the oracle (the worker host, upstream's emulator modules).

    hrd_pay.py hire N SEED      FUN_00576d90 (the purse: carried gold stat 0xe, then the stash stat 0xf) for N random
                                (gold, stash, price) cases: `g b p: ok g2 b2`
    hrd_pay.py revive N SEED    FUN_006637b0 (price from merc level stat 0xc) then FUN_00576d90, as FUN_00579c00 chains
                                them at 0x579c9e..0x579ca9, for N random (gold, stash, level): `g b L: ok g2 b2`

Hooked (stat storage only, V read of each in the disassembly): FUN_00625480 stat get, FUN_006272b0 stat add,
FUN_00627260 stat set, FUN_00622e70 gold limit. Everything else, including FUN_00576d90's and FUN_0053ff00's own
decisions, is game.exe's real code. tools/emu/hrd_d2d_test.cpp prints d2d's rules::hire / resurrect_merc on the same
cases (same xorshift32 stream); diff the two with tools/emu/mst_cmp.py or plain diff.
"""
import sys

import drlg

GOLD_LIMIT = 1_000_000
MASK = 0xffffffff


def stream(seed):
    x = seed & MASK or 1
    while True:
        x ^= (x << 13) & MASK
        x ^= x >> 17
        x ^= (x << 5) & MASK
        yield x


def pick(r, top):
    return next(r) % (top + 1)


def case(r, revive):
    scale = (3, 120, 1000, 20000, 100000, 2_500_000)
    g = min(pick(r, scale[next(r) % 6]), GOLD_LIMIT)
    b = pick(r, scale[next(r) % 6])
    if revive:
        return g, b, pick(r, 255)
    kind = next(r) % 4
    if kind == 0:
        p = pick(r, 120000)
    elif kind == 1:
        p = max(0, g + pick(r, 4) - 2)
    elif kind == 2:
        p = max(0, g + b + pick(r, 4) - 2)
    else:
        p = pick(r, 40)
    return g, b, p


def main():
    mode, n, seed = sys.argv[1], int(sys.argv[2]), int(sys.argv[3])
    revive = mode == "revive"
    e = drlg.boot()
    stats = {14: 0, 15: 0, 12: 0}

    def get(_e):
        return stats.get(_e.arg(1), 0) & MASK
    e.hook(0x625480, get, 3)

    def add(_e):
        d = _e.arg(2)
        d -= 1 << 32 if d & 0x80000000 else 0
        stats[_e.arg(1)] = stats.get(_e.arg(1), 0) + d
        return 0
    e.hook(0x6272b0, add, 4)

    def put(_e):
        v = _e.arg(2)
        stats[_e.arg(1)] = v - (1 << 32 if v & 0x80000000 else 0)
        return 0
    e.hook(0x627260, put, 4)
    e.hook(0x622e70, lambda _e: GOLD_LIMIT, 1)
    unit = e.alloc(0x100)                    # unit type 0 (a player): [unit] stays 0
    r = stream(seed)
    for _ in range(n):
        g, b, x = case(r, revive)
        stats[14], stats[15], stats[12] = g, b, x
        price = e.call(0x6637b0, unit) & MASK if revive else x
        ok = e.call(0x576d90, price, ecx=0, edx=unit) & 0xff
        print(f"{g} {b} {x}: {ok} {stats[14]} {stats[15]}")


if __name__ == "__main__":
    main()
