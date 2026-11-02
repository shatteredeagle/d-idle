# SPDX-License-Identifier: GPL-3.0-or-later
"""VERIFY-SKRESIST: my own harness for the resistance rules. Models are written from my own reading of
FUN_0057be00 / FUN_0057bf80 / FUN_005c3540, then compared with game.exe's real code under the emulator.
    python vr_resist.py N SEED
Scaffolding (stat hooks, unit stubs) comes from life_apply.py of the skmana read checkout; no case, model or
claim comes from sk_resist.py."""
import os
import random
import sys

sys.path.insert(0, os.path.expanduser("~/d2d-lab/pr/skmana/tools/emu"))
import life_apply

N, SEED = int(sys.argv[1]), int(sys.argv[2])
rng = random.Random(SEED)
M = 0xFFFFFFFF


def s32(v):
    v &= M
    return v - (1 << 32) if v >> 31 else v


def tdiv(a, b):               # C truncating division
    q = abs(a) // abs(b)
    return q if (a < 0) == (b < 0) else -q


g = life_apply.Game()
e = g.e
base = {}
hire = [0]
e.hook(0x6253b0, lambda e: base.get((e.arg(0), e.arg(1)), 0) & M, 3)
e.hook(0x63ee90, lambda e: hire[0], 1)
player = g.attacker                       # type 0
RES = [0x24, 0x25, 0x27, 0x29, 0x2b, 0x2d]
PIERCE = {0x24: -1, 0x25: -1, 0x27: 0x14d, 0x29: 0x14e, 0x2b: 0x14f, 0x2d: 0x150}
bad = {"A": 0, "B": 0, "C": 0}
cnt = {"A": 0, "B": 0, "C": 0}
imm_hit = 0


def boundary(lo, hi):
    return rng.choice([lo, hi, 0, 99, 100, 101, -1, -100, rng.randint(lo, hi), rng.randint(lo, hi)])


# A: FUN_005c3540(unit, stat, v) for every stat 0x20..0x30, monster / player / hireling
for i in range(N):
    stat = rng.choice(list(range(0x20, 0x31)) + RES * 3)
    v = boundary(-200, 60)
    b = boundary(-100, 200)
    kind = rng.choice(["mon", "mon", "player", "hire"])
    tgt = g.target if kind != "player" else player
    hire[0] = 1 if kind == "hire" else 0
    base.clear(); base[(tgt, stat)] = b
    got = s32(e.call(0x5c3540, v & M, ecx=tgt, edx=stat))
    exp = v
    if stat in RES and v <= 0 and kind == "mon" and b >= 100:
        exp = tdiv(v, 5)
    cnt["A"] += 1
    if got != exp:
        bad["A"] += 1
        if bad["A"] <= 5:
            print("A MISMATCH stat %x v %d base %d kind %s game %d model %d" % (stat, v, b, kind, got, exp))
hire[0] = 0

# B: FUN_0057be00 final resistance; plain monster defender and player defender, all difficulties, both rule sets
rows = {}
for k in range(12):
    rows.setdefault(e.r32(0x732980 + 0x2c * k + 4), 0x732980 + 0x2c * k)
ctx = e.alloc(0x40)
g.stats.clear()
for i in range(N):
    stat = rng.choice(RES)
    row = rows[stat]
    plain = rng.random() < 0.6
    diff = rng.randint(0, 2)
    expansion = rng.randint(0, 1)
    pen = -rng.randint(0, 100)
    tot = boundary(-150, 200)
    pierce = rng.choice([0, 0, rng.randint(0, 100), 50])
    maxr = rng.choice([0, 0, rng.randint(0, 40)])
    e.mu.mem_write(g.game + 0x6c, bytes([0, diff]))
    e.w32(g.game + 0x70, expansion)
    e.w32(g.diffrow, pen & M)
    tgt = g.target if plain else player
    g.stats.clear()
    g.stats[(tgt, stat)] = tot & M
    rmax = e.r32(row + 8)
    if rmax != 0xFFFFFFFF:
        g.stats[(tgt, rmax)] = maxr
    if PIERCE[stat] >= 0:
        g.stats[(g.attacker, PIERCE[stat])] = pierce
    e.w32(ctx, g.game); e.w32(ctx + 4, g.diffrow); e.w32(ctx + 8, g.attacker); e.w32(ctx + 0xc, tgt)
    e.w32(ctx + 0x10, 0); e.w32(ctx + 0x14, 1 if plain else 0)
    got = s32(e.call(0x57be00, regs={"ebx": row, "edi": ctx}))
    # model (my reading)
    r = tot
    if PIERCE[stat] >= 0 and (r < 100 or not plain):
        r -= pierce
    if not plain and stat not in (0x24, 0x25):
        r += pen if expansion else (0, -20, -50)[diff]
    if r <= 0:
        exp = max(r, -100)
    else:
        exp = r
        if not plain:
            cap = (75 + maxr) if rmax != 0xFFFFFFFF else (50 if stat == 0x24 else 75)
            cap = min(cap, 95)
            if r >= cap:
                exp = cap
    cnt["B"] += 1
    if got != exp:
        bad["B"] += 1
        if bad["B"] <= 5:
            print("B MISMATCH stat %x plain %d diff %d exp %d pen %d tot %d pierce %d max %d game %d model %d" %
                  (stat, plain, diff, expansion, pen, tot, pierce, maxr, got, exp))

# C: damage through FUN_0057c6c0 (lightning field) against a plain monster, classic rules, Normal
e.mu.mem_write(g.game + 0x6c, bytes([0, 1]))
e.w32(g.game + 0x70, 1)
for i in range(N):
    res = boundary(-150, 200)
    pierce = rng.choice([0, 0, rng.randint(0, 100)])
    roll = rng.randint(256, 100 * 256)
    g.reset(30000, res, pierce)
    e.mu.mem_write(g.rec, b"\0" * 0x80)
    e.w32(g.rec, 1); e.w32(g.rec + 4, 1); e.w32(g.rec + 0x1c, roll)
    before = g.life()
    e.call(0x57c6c0, g.target, 1, g.rec, ecx=g.game, edx=g.attacker)
    dealt = before - g.life()
    r = res - pierce if res < 100 else res
    r = max(r, -100)
    if r >= 100:
        exp = 0
    else:
        exp = (roll * (100 - r)) // 100             # truncation (first run with round-to-nearest failed 3307/20000)
    cnt["C"] += 1
    if dealt != exp:
        bad["C"] += 1
        if bad["C"] <= 5:
            print("C MISMATCH res %d pierce %d roll %d game %d model %d" % (res, pierce, roll, dealt, exp))
    if r >= 100 and dealt == 0:
        imm_hit += 1
print("cases", cnt, "mismatches", bad, "immune-zero hits", imm_hit, "seed", SEED)
