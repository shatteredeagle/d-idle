# SPDX-License-Identifier: GPL-3.0-or-later
"""verifycovmissiles: own harness. game.exe's FUN_0059fa30 makes a missile; its range, velocity, MaxVel and Accel are read
back and checked against a model built from the RECORD AS LOADED IN GAME MEMORY (not the table text), for

  grid:   every row 0..683 x 18 levels (0, 1, 2, 6, 11, 13, 18, 24, 31, 37, 45, 63, 80, 99, 127, 128, 200, 255)
  random: N cases, row random, record perturbed half the time (Range, LevRange short; Vel, VelLev, MaxVel byte; Accel short;
          VelLev above 127 and a negative LevRange among them), level 0..255

    vcm_flight.py OUTDIR N SEED

Writes OUTDIR/game.tsv (`id level vel vlev rng lrng | made gvel grange gmax gacc | model_bad`) and OUTDIR/d2din.txt (`vel vlev
level rng lrng`, the input of vcm_test, the real d2d rules::missile_velocity / missile_range). Prints counts to stderr.
"""
import os
import random
import sys

import missiles

LEVELS = (0, 1, 2, 6, 11, 13, 18, 24, 31, 37, 45, 63, 80, 99, 127, 128, 200, 255)


def tdiv(a, b):
    q = abs(a) // abs(b)
    return q if (a < 0) == (b < 0) else -q


def s32(v):
    v &= 0xFFFFFFFF
    return v - (1 << 32) if v & 0x80000000 else v


def main():
    out, n, seed = sys.argv[1], int(sys.argv[2]), int(sys.argv[3])
    os.makedirs(out, exist_ok=True)
    rng = random.Random(seed)
    game = missiles.Game()
    e = game.e
    sh = None
    while not sh:
        case = missiles.make_case(game, rng)
        sh = game.unit(63, *case["from"])
    src, sx, sy, _ = sh
    jobs = [(m, l, False) for m in range(684) for l in LEVELS]
    for _ in range(n):
        jobs.append((rng.randrange(684), rng.choice((rng.randrange(0, 256), rng.randrange(0, 41), 0, 1)), rng.random() < 0.5))
    stats = {"cases": 0, "made": 0, "none": 0, "bad_vel": 0, "bad_rng": 0, "bad_max": 0, "bad_acc": 0, "perturbed": 0}
    with open(f"{out}/game.tsv", "w") as g, open(f"{out}/d2din.txt", "w") as d:
        for mid, lvl, pert in jobs:
            base = game.rows + mid * 0x1a4
            saved = e.read(base + 0x96, 10)
            if pert:
                stats["perturbed"] += 1
                rg, lr = rng.choice((rng.randint(-50, 400), 0, rng.randint(0, 60))), rng.choice((0, rng.randint(-60, 60), rng.randint(-4000, 4000)))
                vel, vl, mv = rng.randrange(0, 256), rng.choice((0, rng.randrange(0, 12), rng.randrange(0, 256))), rng.randrange(0, 256)
                ac = rng.choice((0, rng.randint(-300, 300), rng.randint(-32768, 32767)))
                e.mu.mem_write(base + 0x96, (rg & 0xFFFF).to_bytes(2, "little") + (lr & 0xFFFF).to_bytes(2, "little")
                               + bytes([vel, vl, mv, 0]) + (ac & 0xFFFF).to_bytes(2, "little"))
            # the record as game memory holds it
            rg, lr = game.row(mid, 0x96, 2), game.row(mid, 0x98, 2)
            vel, vl, mv, ac = game.row(mid, 0x9a, 1), game.row(mid, 0x9b, 1), game.row(mid, 0x9c, 1), game.row(mid, 0x9e, 2)
            p = e.alloc(0x60)
            tx, ty = sx + rng.choice((-1, 1)) * rng.randint(1, 40), sy + rng.choice((-1, 1)) * rng.randint(1, 40)
            for i, v in {0: 0x21, 1: src, 4: mid, 5: sx, 6: sy, 7: tx, 8: ty, 0xc: lvl}.items():
                e.w32(p + 4 * i, v)
            m = e.call(0x59fa30, ecx=game.g, edx=p)
            e.mu.mem_write(base + 0x96, saved)
            stats["cases"] += 1
            if not m:
                stats["none"] += 1
                g.write(f"{mid}\t{lvl}\t{vel}\t{vl}\t{rg}\t{lr}\t0\t0\t0\t0\t0\t0\n")
                continue
            stats["made"] += 1
            path = e.r32(m + 0x2c)
            gr = e.r16(e.r32(m + 0x14) + 0x10)
            gr = gr - 0x10000 if gr & 0x8000 else gr
            gv, gm, ga = e.s32(path + 0x7c), e.s32(path + 0x84), e.s32(path + 0x88)
            want_v = tdiv(s32((vel + tdiv(vl * lvl, 8)) << 8) * 75, 100)
            want_r = max(-0x8000, min(0x7fff, rg + lr * lvl))
            bad = (gv != want_v) | (gr != want_r) << 1 | (gm != mv << 8) << 2 | (ga != ac) << 3
            stats["bad_vel"] += bool(bad & 1)
            stats["bad_rng"] += bool(bad & 2)
            stats["bad_max"] += bool(bad & 4)
            stats["bad_acc"] += bool(bad & 8)
            g.write(f"{mid}\t{lvl}\t{vel}\t{vl}\t{rg}\t{lr}\t1\t{gv}\t{gr}\t{gm}\t{ga}\t{bad}\n")
            d.write(f"{vel} {vl} {lvl} {rg} {lr}\n")
    print(f"vcm_flight seed {seed}: {stats}", file=sys.stderr)


if __name__ == "__main__":
    main()
