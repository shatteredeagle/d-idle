# SPDX-License-Identifier: GPL-3.0-or-later
"""verifyskbarb: independent run of the Barbarian cries' rings: the params the REAL game.exe builds and the flights of the
64 missiles by the real FUN_0059fa30 / srvdofunc.

    run-on-worker.sh vbb_ring.py params       # phase 1: Howl through FUN_005c9b50 (the real do 22), the war cries through
                                              # FUN_0056d400 (the ring maker of do 68, velocity 0), FUN_0059fa30 hooked to RECORD
                                              # the 64 param blocks of each (level 1..40); writes $D2_SCRATCH/vbb_ring_params.json
    run-on-worker.sh vbb_ring.py fly [SEED]   # phase 2 (fresh process, the real FUN_0059fa30): the 64 missiles made from those
                                              # blocks, stepped frame by frame by their srvdofunc, end distance in subtiles

Own design: the velocity argument of FUN_0056d400 and its flags come from the recorded block (not from my reading of do 22);
the flights are not compared with a port, only summarised (frames flown, min / max end distance per level).
"""
import json
import math
import os
import random
import sys

import missiles
from unicorn.x86_const import UC_X86_REG_EDX

OUT = os.path.join(os.environ.get("D2_SCRATCH", "/tmp"), "vbb_ring_params.json")
RINGS = {"howl": (148, 130, 5), "warcry": (235, 154, 1), "battlecry": (219, 146, 1), "shout": (149, 138, 1), "battleorders": (237, 149, 1)}
LEVELS = (1, 2, 5, 10, 15, 20, 30, 40)


def free_disc(game, x, y, r):
    for dx in range(-r, r + 1):
        for dy in range(-r, r + 1):
            if dx * dx + dy * dy <= r * r:
                c = game.collision(x + dx, y + dy)
                if c == 0xFFFF or c & 0x1f0f:
                    return False
    return True


def site(game, rng, r=24):
    for room in game.rooms:
        _, rx, ry, rw, rh = room[:5]
        for _ in range(200):
            x, y = rx + r + rng.randrange(max(rw - 2 * r, 1)), ry + r + rng.randrange(max(rh - 2 * r, 1))
            if game.room_at(x, y) and free_disc(game, x, y, r):
                return x, y
    return None


def phase_params():
    game = missiles.Game()
    e = game.e
    rng = random.Random(5)
    s = site(game, rng, 8) or site(game, rng, 4)
    cas = game.unit(63, *s)
    caster = cas[0]
    seen = []

    def rec(e):
        p = e.mu.reg_read(UC_X86_REG_EDX)
        seen.append([e.r32(p + 4 * i) for i in range(0x18)])
        return 0
    e.hook(0x59fa30, rec, 0)
    out = {}
    for name, (row, skill, _) in RINGS.items():
        out[name] = {}
        for lvl in LEVELS:
            seen.clear()
            if name == "howl":
                ok = e.call(0x5c9b50, skill, lvl, ecx=game.g, edx=caster)       # the real do 22 (ret 8)
            else:
                ok = e.call(0x56d400, caster, row, skill, lvl, 0, ecx=game.g, edx=caster)   # the real ring maker, velocity 0
            out[name][lvl] = {"ret": ok, "blocks": [list(b) for b in seen]}
            print(f"S {name} L{lvl}: ret {ok}, {len(seen)} blocks, flags {sorted({b[0] for b in seen})}, vel word {sorted({b[10] for b in seen})}, "
                  f"row {sorted({b[4] for b in seen})}, skill {sorted({b[11] for b in seen})}, lvl {sorted({b[12] for b in seen})}")
    with open(OUT, "w") as f:
        json.dump(out, f)
    print("SUMMARY params written", OUT)


def phase_fly(seed):
    data = json.load(open(OUT))
    game = missiles.Game()
    e = game.e
    rng = random.Random(seed)
    sites = []
    while len(sites) < 3:
        s = site(game, rng, 24)
        if s is None:
            break
        sites.append(s)
    print("# free sites", len(sites))
    for name, (row, skill, _) in RINGS.items():
        fn = e.r32(0x73c768 + 4 * game.row(row, 0xc, 2))
        for lvl in LEVELS:
            blocks = data[name][str(lvl)]["blocks"]
            lo, hi, frames, made = 1e9, 0, set(), 0
            for (sx, sy) in sites:
                cas = game.unit(63, sx, sy)
                for b in blocks:
                    p = e.alloc(0x60)
                    for i, v in enumerate(b):
                        e.w32(p + 4 * i, v & 0xFFFFFFFF)
                    e.w32(p + 4, cas[0])
                    e.w32(p + 4 * 5, sx)
                    e.w32(p + 4 * 6, sy)
                    m = e.call(0x59fa30, ecx=game.g, edx=p)
                    if not m:
                        continue
                    made += 1
                    path = e.r32(m + 0x2c)
                    x0, y0 = (sx << 16) + 0x8000, (sy << 16) + 0x8000
                    n, pos = 0, (x0, y0)
                    for _ in range(80):
                        if e.s32(e.r32(m + 0x14) + 0x10) << 16 >> 16 < 1:
                            break
                        e.call(fn, ecx=game.g, edx=m)
                        pos = (e.s32(path), e.s32(path + 4))
                        n += 1
                    d = math.hypot(pos[0] - x0, pos[1] - y0) / 65536
                    lo, hi = min(lo, d), max(hi, d)
                    if name in ("warcry", "howl") and lvl in (1, 20) and (sx, sy) == sites[0] and blocks.index(b) % 8 == 0:
                        print(f"  dbg {name} L{lvl} block dx,dy {b[7]},{b[8]} (x0 {x0 >> 16}.{(x0 & 0xffff) / 65536:.3f}) end {pos[0] / 65536:.3f},{pos[1] / 65536:.3f} "
                              f"d {d:.3f} frames {n}")
                    frames.add(n)
            print(f"S {name} L{lvl}: {made} of {64 * len(sites)} made, frames {sorted(frames)}, end distance {lo:.2f} - {hi:.2f} subtiles")
    print("SUMMARY fly done")


if __name__ == "__main__":
    if sys.argv[1] == "params":
        phase_params()
    else:
        phase_fly(int(sys.argv[2]) if len(sys.argv) > 2 else 1)
