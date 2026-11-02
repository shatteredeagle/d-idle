# SPDX-License-Identifier: GPL-3.0-or-later
"""skbarb: the Barbarian rings (War Cry, Battle Cry, Howl) flown by game.exe's own code, and the NextHit model of
sk_nova_ring.py part C on them.

    run-on-worker.sh skbarb_ring.py ROW SKILL NEXTDELAY [spots] [seed] [monsters]
      War Cry      235 154 6        Battle Cry  219 146 4        Howl  148 130 0 (no NextHit: only flights)

sk_nova_ring.py (Frost Nova, row 119) with the row, skill and NextDelay as arguments, and without the port comparison
(its flights are checked there). Per spot of a Blood Moor game (missiles.py's Game) and per skill level 1 and 20:
the 64 ring missiles made by the REAL FUN_0059fa30 from params built as FUN_0056d400 builds them (offsets of the tables
0x6e1288 / 0x6e1388, velocity FUN_00663270(row, lvl)), each stepped by its real srvdofunc (FUN_00554200 hooked to
"hostile to nobody"): frames per flight and the end distance. Then part C's MODEL (tag I, not game code) with the game.exe
rule (one unit per missile and frame, the missile's last unit skipped, justhit for NextDelay frames) against d2d's
(`hit_at` per missile): hits per monster per cast by distance. Prints `B ...`, `C ...` lines and `SUMMARY`."""
import math
import random
import sys

import missiles
import sk_nova_ring as ring

ROW, SKILL, DELAY = int(sys.argv[1]), int(sys.argv[2]), int(sys.argv[3])
DEFAULT_VEL = "--default-vel" in sys.argv       # the war cries (do 68): no velocity override
HOWL = "--howl" in sys.argv
rest = [a for a in sys.argv[4:] if not a.startswith("--")]
SPOTS = int(rest[0]) if rest else 4
SEED = int(rest[1]) if len(rest) > 1 else 1
NMON = int(rest[2]) if len(rest) > 2 else 4000


def ring_flights(game, src, sx, sy, lvl, offs, limit=60):
    e = game.e
    vel = e.call(0x663270, ROW, lvl)
    if HOWL: vel += 2 * (lvl - 1)               # do 22: Vel + VelLev x lvl / 8 + calc1, calc1 = Param1 x (lvl - 1) = 2 (lvl - 1) (V run, skbarb_calc.py)
    fn = e.r32(0x73c768 + 4 * game.row(ROW, 0xc, 2))
    out = []
    for k in range(64):
        p = e.alloc(0x60)
        for i in range(0x18): e.w32(p + 4 * i, 0)
        # do 68 (FUN_005d83e0) calls FUN_0056d400 with velocity 0: flags 3, no override (V read, 0x5d843d); do 22 (Howl) passes one
        fields = {0: 3 if DEFAULT_VEL else 3 | 4, 1: src, 4: ROW, 5: sx, 6: sy, 7: offs[k][0] & 0xFFFFFFFF, 8: offs[k][1] & 0xFFFFFFFF,
                  0xa: 0 if DEFAULT_VEL else vel, 0xb: SKILL, 0xc: lvl}
        for i, v in fields.items(): e.w32(p + 4 * i, v)
        m = e.call(0x59fa30, ecx=game.g, edx=p)
        if not m:
            out.append(None)
            continue
        path = e.r32(m + 0x2c)
        rng = e.s32(e.r32(m + 0x14) + 0x10) << 16 >> 16
        track = []
        for _ in range(limit):
            if e.s32(e.r32(m + 0x14) + 0x10) << 16 >> 16 < 1: break
            e.call(fn, ecx=game.g, edx=m)
            track.append((e.r32(path), e.r32(path + 4)))
        out.append((rng, track))
    return out, vel


def main():
    rng = random.Random(SEED)
    game = missiles.Game()
    e = game.e
    offs = [(e.s32(0x6e1288 + 4 * k), e.s32(0x6e1388 + 4 * k)) for k in range(64)]
    found, tries = [], 0
    while len(found) < SPOTS and tries < 20000:
        tries += 1
        room = rng.choice(game.rooms)
        x, y = room[1] + rng.randrange(room[3]), room[2] + rng.randrange(room[4])
        if ring.free_disc(game, x, y, 20): found.append((x, y))
    print(f"# row {ROW} skill {SKILL} NextDelay {DELAY} spots {found}")
    keep = []
    for (x, y) in found:
        u = game.unit(63, x, y)
        if not u: continue
        src, sx, sy, _ = u
        for lvl in (1, 20):
            got, vel = ring_flights(game, src, sx, sy, lvl, offs)
            lens = sorted(set(len(g[1]) for g in got if g))
            reach = [math.hypot((g[1][-1][0] >> 16) - sx, (g[1][-1][1] >> 16) - sy) for g in got if g and g[1]]
            print(f"B spot ({sx},{sy}) lvl {lvl}: vel {vel}, {sum(1 for g in got if g)} of 64 missiles, Range {sorted(set(g[0] for g in got if g))}, "
                  f"frames {lens}, end distance min {min(reach):.2f} max {max(reach):.2f}")
            if lvl == 1:
                keep.append((sx, sy, [ring.crossed_of(g[1], sx, sy) if g else [] for g in got]))
    if not keep or DELAY <= 0:
        print("SUMMARY: flights only")
        return
    sx, sy, cr = keep[0]
    flights = [c for c in cr if c]
    for window in (DELAY, DELAY - 1):
        by = {}
        for _ in range(NMON):
            pack = rng.choice((1, 1, 3, 6))
            foes = []
            while len(foes) < pack:
                d, a = rng.uniform(0.5, 16), rng.uniform(0, 2 * math.pi)
                fx, fy = sx + round(d * math.cos(a)), sy + round(d * math.sin(a))
                if (fx, fy) == (sx, sy) or any((fx, fy) == (f[0], f[1]) for f in foes): continue
                foes.append((fx, fy, rng.choice((1, 2, 3))))
            hg, hd = ring.hits_game(flights, foes, window), ring.hits_d2d(flights, foes, window)
            for j, (fx, fy, size) in enumerate(foes):
                band = min(int(math.hypot(fx - sx, fy - sy)), 16)
                s = by.setdefault((band, size), [0, 0, 0, 0])
                s[0] += 1; s[1] += hg[j]; s[2] += hd[j]; s[3] += hg[j] > 1
        print(f"C window {window}: distance band, size: monsters, hits per cast under the game.exe rule, under d2d's, game.exe 2+ hits")
        for key in sorted(by):
            n, g, d, two = by[key]
            print(f"  d{key[0]:>2} s{key[1]} n={n:>5} game {g / n:.3f} d2d {d / n:.3f} game2+ {two / n:.3f}")
        tot = [sum(v[i] for v in by.values()) for i in range(4)]
        print(f"  all n={tot[0]} game {tot[1] / tot[0]:.3f} d2d {tot[2] / tot[0]:.3f} ratio {tot[2] / max(tot[1], 1):.2f}")
        if window == DELAY:
            print(f"SUMMARY window {window}: all monsters within 16 subtiles: game rule {tot[1] / tot[0]:.2f} hits per cast, d2d rule {tot[2] / tot[0]:.2f}, ratio {tot[2] / max(tot[1], 1):.2f}")


main()
