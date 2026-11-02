# SPDX-License-Identifier: GPL-3.0-or-later
"""verifypalaura: own re-check of two PALAURA claims with different seeds and my own model.
  roll: Holy Fire 1 (do 66, FUN_005cf3a0) many pulses, one resist-0 rat: observed min/max vs table min..max-1
  disc: area search FUN_0056b7e0 with a monster filter, random units, own model d2 <= r*r
run: tools/emu/run-on-worker.sh vp_check.py roll 4000 | disc 7"""
import sys

import palaura_area as pa
from palaura_apply import prep

mode = sys.argv[1]
if mode == "roll":
    n = int(sys.argv[2])
    w = pa.World(11)
    w.real_callback()
    e = w.e
    prep(w, {102: 1})
    cx, cy = w.free_spot()
    caster, _, _ = w.make(pa.PLAYER, 4, cx, cy)
    for stat, val in ((0xc, 30), (9, 100 << 8), (8, 100 << 8), (7, 100 << 8), (6, 100 << 8)):
        e.call(0x627150, e.r32(caster + 0x5c), stat, val, 0)
    lo = e.call(0x644d50, caster, 102, 1, 1)
    hi = e.call(0x644e40, caster, 102, 1, 1)
    e.w32(caster + 0x20, 0x13579bdf | 1)
    e.w32(caster + 0x24, 4242)
    rat, _, _ = w.make(pa.MONSTER, 63, cx + 2, cy - 2)
    for st in (7, 6):
        e.call(0x627150, e.r32(rat + 0x5c), st, 100000 << 8, 0)
    rolls = []
    for _ in range(n):
        b = e.call(0x625480, rat, 6, 0)
        e.call(0x5cf3a0, 102, 1, ecx=w.game.g, edx=caster, limit=50000000)
        a = e.call(0x625480, rat, 6, 0)
        rolls.append(b - a)
        e.call(0x627150, e.r32(rat + 0x5c), 6, 100000 << 8, 0)
    print(f"table min {lo} max {hi}; {n} pulses: observed min {min(rolls)} max {max(rolls)} distinct {len(set(rolls))}; "
          f"count == {hi}: {rolls.count(hi)}; count outside [lo,hi): {sum(not lo <= r < hi for r in rolls)}; "
          f"count == lo: {rolls.count(lo)}")
elif mode == "disc":
    r = int(sys.argv[2])
    w = pa.World(9)   # keep the recording hook on the callback (real_callback would remove it)
    e = w.e
    cx, cy = w.free_spot()
    caster, _, _ = w.make(pa.PLAYER, 4, cx, cy)
    units = {}
    seen_pos = set()
    for i in range(400):
        dx, dy = w.rng.randint(-r - 2, r + 2), w.rng.randint(-r - 2, r + 2)
        if (dx, dy) == (0, 0) or (dx, dy) in seen_pos: continue
        seen_pos.add((dx, dy))
        m = w.make(pa.MONSTER, 63, cx + dx, cy + dy)
        if m: units[m[0]] = (dx, dy)
    got = w.search(caster, r, 3)
    bad = []
    for u, (dx, dy) in units.items():
        # real unit position, not the requested one
        p = e.r32(u + 0x2c)
        ux, uy = e.r16(p + 2), e.r16(p + 6)
        d2 = (ux - cx) ** 2 + (uy - cy) ** 2
        if (u in got) != (d2 <= r * r): bad.append((ux - cx, uy - cy, u in got))
    print(f"radius {r}: {len(units)} units, {len(units) - len(bad)} agree with d2 <= r2, disagree {bad}")
    onc = [u for u, _ in units.items() if (lambda p: (e.r16(p + 2) - cx) ** 2 + (e.r16(p + 6) - cy) ** 2)(e.r32(u + 0x2c)) == r * r]
    print("units exactly on the circle:", len(onc), "found:", sum(u in got for u in onc))
