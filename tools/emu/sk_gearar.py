#!/usr/bin/env python3
"""Gear attack-rating terms, stats 115 / 116 / 123 / 124: FUN_0057d8a0 of game.exe 1.14d under unicorn vs a written reading (card skgearar).

Runs next to upstream's harness (emu.py) and sk_melee2.py (helpers tdiv, muldiv).

    python sk_gearar.py run 100000 7      # boundary grid + 100000 random cases, seed 7; prints the report

The real FUN_0057d8a0 runs; stubbed: the stat getter FUN_00625500, the demon / undead tests FUN_0063e940 / FUN_0063e990,
FUN_0063e9f0. FUN_005a0180 (flag word on the target's data), FUN_0063ee90 (class id) and the MulDiv FUN_00483360 run for real.
Input: AR, defense, stats 115 / 116 / 123 / 124, target type 0 / 1 / 2, boss-ish kind, demon / undead.
Stat 179 and stat 120 are not run here (see the trace, Q4 / Q5).
"""
import random
import sys

from sk_melee2 import M32, muldiv, tdiv

NOBOSS, BOSS_MASK2, BOSS_MASK10, BOSS_FLAG9F0, BOSS_CLASS = range(5)


def s32(v):
    v &= M32
    return v - (1 << 32) if v >= 1 << 31 else v


def model_g(ar, df, s115, s116, s123, s124, ttype, kind, demon, undead):
    """The reading of FUN_0057d8a0 (trace Q1)."""
    if s115 != 0 and ttype == 1 and kind == NOBOSS:      # mask-10 flags clear, FUN_0063e9f0 and FUN_0063ee90 clear
        df = 0
    if s116 > 0:
        s = s116
        if ttype == 0 or (ttype == 1 and kind != NOBOSS and kind != BOSS_MASK10):   # mask-2 flag, 9f0, boss class
            s = tdiv(s, 2)
        s = min(max(s, 0), 100)
        df = df - muldiv(df, s, 100)
    if s123 != 0 and demon: ar += s123
    if s124 != 0 and undead: ar += s124
    return ar, df


def gen_g(cases, rng):
    out = []
    for s116 in (1, 2, 3, 99, 100, 101, 200, -5):
        for kind in range(5):
            for df in (-7, -1, 0, 1, 99, 100, 101, 1000):
                for ttype in (0, 1, 2):
                    out.append((200, df, 5, s116, 0, 0, ttype, kind, 0, 0))
    for _ in range(cases):
        out.append((rng.randrange(-100, 3000), rng.randrange(-50, 3000),
                    rng.choice((0, 0, 1, 1, 5)), rng.choice((0, 0, 0, 10, 25, 50, 75, 100, 150, -10)),
                    rng.choice((0, 0, 0, 20, 50, 100, -30)), rng.choice((0, 0, 0, 20, 50, 100, -30)),
                    rng.choice((1, 1, 1, 0, 2)), rng.choice((0, 0, 0, 1, 2, 3, 4)),
                    rng.choice((0, 1)), rng.choice((0, 1))))
    return out


def make_target(e, ttype, kind):
    t = e.alloc(0x100)
    e.w32(t, ttype)
    e.w32(t + 4, 0x167 if kind == BOSS_CLASS else 1)          # FUN_0063ee90 keys on the class id
    data = e.alloc(0x100)
    e.w32(t + 0x14, data)
    mask = {BOSS_MASK2: 2, BOSS_MASK10: 8}.get(kind, 0)       # FUN_005a0180 tests the word at data + 0x16
    e.mu.mem_write(data + 0x16, mask.to_bytes(2, "little"))
    return t


def run_g(cases):
    import emu
    e = emu.Emu()
    stats, st = {}, {}
    e.hook(0x625500, lambda e: stats.get((e.arg(0), e.arg(1)), 0), 3)
    e.hook(0x63e940, lambda e: st["demon"], 1)
    e.hook(0x63e990, lambda e: st["undead"], 1)
    e.hook(0x63e9f0, lambda e: st["f9f0"], 2)
    att = e.alloc(0x100)
    res = []
    for (ar, df, s115, s116, s123, s124, ttype, kind, demon, undead) in cases:
        stats.clear()
        stats.update({(att, 0x73): s115, (att, 0x74): s116, (att, 0x7b): s123, (att, 0x7c): s124})
        st.update(demon=demon, undead=undead, f9f0=1 if kind == BOSS_FLAG9F0 else 0)
        t = make_target(e, ttype, kind)
        pa, pd = e.alloc(8), e.alloc(8)
        e.w32(pa, ar); e.w32(pd, df)
        e.call(0x57d8a0, pa, pd, regs={"ebx": att, "edi": t})
        res.append((s32(e.r32(pa)), s32(e.r32(pd))))
    return res


if __name__ == "__main__":
    n, seed = int(sys.argv[2]), int(sys.argv[3])
    rng = random.Random(seed & M32)
    cases = gen_g(n, rng)
    got = run_g(cases)
    bad = 0
    for c, g in zip(cases, got):
        w = model_g(*c)
        if w != g:
            bad += 1
            if bad <= 15: print("DIFF", c, "game", g, "model", w)
    changed = sum(1 for c, g in zip(cases, got) if g != (c[0], c[1]))
    print(f"G: {len(cases)} cases (seed {seed}, {len(cases) - n} boundary), {bad} differ from the reading; game changed AR or defense in {changed}")
    for name, f in (("115", lambda c: c[2] != 0 and c[6] == 1), ("116", lambda c: c[3] > 0), ("123", lambda c: c[4] != 0 and c[8]), ("124", lambda c: c[5] != 0 and c[9])):
        sel = [(c, g) for c, g in zip(cases, got) if f(c)]
        ch = sum(1 for c, g in sel if g != (c[0], c[1]))
        print(f"  stat {name}: {len(sel)} cases with the term live, {ch} changed AR / defense")
    for k, nm in enumerate(("plain", "mask2", "mask8", "fn9f0", "class167")):
        sel = [(c, g) for c, g in zip(cases, got) if c[7] == k and c[6] == 1 and c[2] != 0]
        z = sum(1 for c, g in sel if g[1] == 0 and c[1] != 0)
        print(f"  115 on a monster, kind {nm}: {len(sel)} cases, defense zeroed in {z}")
