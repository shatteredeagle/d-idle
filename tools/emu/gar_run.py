#!/usr/bin/env python3
"""Stat 179 (attack_vs_montype) in FUN_0057d9b0 and stat 120 (item_damagetargetac) in FUN_0057b7d0, game.exe 1.14d under unicorn
vs a written reading (card geararmore). Next to upstream's emu.py and tools/emu/sk_melee2.py (helpers hit_chance, muldiv).

    python gar_run.py a 100000 7    # part A: stat 179, chance of FUN_0057d9b0 (captured at FUN_00554010's call)
    python gar_run.py b 100000 7    # part B: stat 120, the defense written back (FUN_00627260's call) by FUN_0057b7d0

Part A runs the real FUN_0057d9b0, FUN_00483360 (MulDiv), FUN_00451f80 (monster record by class id) and FUN_0057a830 (the MonType
membership bit test) on a synthetic game-data struct written at [0x744304]; stubbed: the stat getters, the attacker AR / defense
helpers (FUN_00622560, FUN_006223f0), FUN_0057d8a0 (0, run in sk_gearar), the mastery lookup, FUN_006261d0 (the stat-179 list).
Part B runs FUN_0057b7d0 up to the stat-120 branch with every callee stubbed; the run is cut at FUN_0057b660 (first call after it).
"""
import random
import sys

from sk_melee2 import M32, hit_chance, muldiv, tdiv

NMON, STRIDE = 100, 4          # MonType count, dwords per row of the group table


class Stop(Exception):
    pass


def s32(v):
    v &= M32
    return v - (1 << 32) if v >= 1 << 31 else v


def setup_globals(e, rng):
    """[0x744304] -> struct: +0xc40 NMON, +0xc44 table, +0xc48 stride, +0xa78 monster records (0x1a8 each), +0xa80 count."""
    g = e.alloc(0x1000)
    tab = e.alloc(NMON * STRIDE * 4 + 16)
    nrec = 4
    recs = e.alloc(nrec * 0x1a8 + 16)
    e.w32(0x744304, g)
    e.w32(g + 0xc40, NMON); e.w32(g + 0xc44, tab); e.w32(g + 0xc48, STRIDE)
    e.w32(g + 0xa78, recs); e.w32(g + 0xa80, nrec)
    bits = [[rng.getrandbits(32) if rng.random() < 0.5 else 0 for _ in range(STRIDE)] for _ in range(NMON)]
    for r in range(NMON):
        for w in range(STRIDE):
            e.w32(tab + 4 * (r * STRIDE + w), bits[r][w])
    return recs, bits


def member(bits, t, m):
    """The reading of FUN_0057a830 (t = entry type, m = target MonType)."""
    if not (0 < t < NMON and 0 < m < NMON):
        return 0
    return (bits[m][t >> 5] >> (t & 31)) & 1


def model_a(c, bits):
    atype, ttype, hasrec, mt, ar, df, mdef, alvl, dlvl, s19, s2, s119, bonus, mastery, weapon, ents = c
    if atype == 0:
        pct = (mastery if weapon else 0) + s119 + bonus
        if ttype == 1 and hasrec and 0 < (mt if mt < 0x8000 else mt - 0x10000):
            mm = mt if mt < 0x8000 else mt - 0x10000
            for (t, v) in ents:
                if member(bits, t, mm): pct += v
        a = ar
    else:
        pct = s119
        a = s19 + bonus + 5 * s2
    a = a + muldiv(a, pct, 100)
    return hit_chance(a, df + mdef, alvl, dlvl)


def gen_a(n, rng):
    out = []
    ent_edges = [(0, 40), (1, 40), (NMON - 1, 40), (NMON, 40), (65535, 40), (50, -30)]
    for mt in (0, 1, 2, NMON - 1, NMON, 0x7fff, 0x8000, 0xffff):
        for ttype in (0, 1):
            for atype in (0, 1):
                out.append((atype, ttype, 1, mt, 400, 200, 0, 50, 50, 300, 20, 0, 0, 0, 0, tuple(ent_edges)))
    for _ in range(n):
        ne = rng.choice((0, 1, 2, 3, 8, 128))
        ents = tuple((rng.choice((rng.randrange(0, NMON + 5), rng.randrange(0, 65536))), rng.choice((0, 20, 50, 100, -30, 250, -150))) for _ in range(ne))
        out.append((rng.choice((0, 0, 0, 1)), rng.choice((1, 1, 1, 0)), rng.choice((1, 1, 1, 0)),
                    rng.choice((rng.randrange(0, NMON + 3), rng.randrange(0, 65536))),
                    rng.randrange(-40, 2500), rng.randrange(-30, 1500), rng.choice((0, 0, 10)),
                    rng.randrange(1, 99), rng.randrange(1, 99), rng.randrange(0, 800), rng.randrange(0, 200),
                    rng.choice((0, 0, 0, 10, 50, -20)), rng.choice((0, 0, 30, 100)), rng.choice((0, 28, 61)), rng.choice((0, 1)), ents))
    return out


def run_a(cases, rng):
    import emu
    e = emu.Emu()
    recs, bits = setup_globals(e, rng)
    stats, st, cap = {}, {}, {}
    e.hook(0x625480, lambda e: stats.get((e.arg(0), e.arg(1)), 0), 3)
    e.hook(0x625500, lambda e: stats.get((e.arg(0), e.arg(1)), 0), 3)
    e.hook(0x6223f0, lambda e: stats.get((e.arg(0), "def"), 0), 1)
    e.hook(0x622560, lambda e: st["ar"], 1)
    e.hook(0x57d8a0, lambda e: 0, 2)
    e.hook(0x645830, lambda e: st["mastery"], 4)
    e.hook(0x535bc0, lambda e: st["weapon"], 0)
    for addr, n in ((0x5347b0, 0), (0x57d810, 0)):
        e.hook(addr, lambda e: 0, n)

    def h554010(e):
        cap["chance"] = e.arg(1)
        return 0
    e.hook(0x554010, h554010, 3)

    def h6261d0(e):
        buf, ents = e.arg(2), st["ents"]
        for i, (t, v) in enumerate(ents):
            e.mu.mem_write(buf + 8 * i, (t & 0xffff).to_bytes(2, "little") + b"\0\0" + (v & M32).to_bytes(4, "little"))
        return len(ents)
    e.hook(0x6261d0, h6261d0, 4)
    res = []
    for c in cases:
        atype, ttype, hasrec, mt, ar, df, mdef, alvl, dlvl, s19, s2, s119, bonus, mastery, weapon, ents = c
        att, tgt = e.alloc(0x100), e.alloc(0x100)
        e.w32(att, atype); e.w32(tgt, ttype)
        e.w32(att + 0x20, 12345); e.w32(att + 0x24, 678)
        cls = 1 if hasrec else 9                     # class id 9 is >= the record count (4): FUN_00451f80 gives 0
        e.w32(tgt + 4, cls)
        e.mu.mem_write(recs + 0x1a8 * 1 + 0x1c, (mt & 0xffff).to_bytes(2, "little"))
        stats.clear(); cap.clear()
        stats.update({(att, 0xc): alvl, (att, 0x13): s19, (att, 2): s2, (att, 0x77): s119,
                      (tgt, 0xc): dlvl, (tgt, "def"): df, (tgt, 0x21): mdef})
        st.update(ar=ar, mastery=mastery, weapon=0x7000000 if weapon else 0, ents=ents)
        e.call(0x57d9b0, bonus, 0, ecx=att, edx=tgt)
        res.append(cap.get("chance"))
    return res, bits


def run_b(cases):
    import emu
    e = emu.Emu()
    stats, wr = {}, []
    e.hook(0x625500, lambda e: stats.get((e.arg(0), e.arg(1)), 0), 3)
    e.hook(0x625480, lambda e: stats.get((e.arg(0), e.arg(1)), 0), 3)
    e.hook(0x627260, lambda e: wr.append((e.arg(0), e.arg(1), s32(e.arg(2)))) or 0, 4)
    e.hook(0x535d10, lambda e: 0, 0)
    e.hook(0x6259b0, lambda e: 2 if stats.get("t2") else 0, 1)
    e.hook(0x63e940, lambda e: 0, 1)

    def stop(e):
        raise Stop()
    e.hook(0x57b660, stop, 0)
    res = []
    for (atype, ttype, s120, d31, as2) in cases:
        att, tgt, rec = e.alloc(0x100), e.alloc(0x100), e.alloc(0x100)
        e.w32(att, atype); e.w32(tgt, ttype)
        stats.clear(); wr.clear()
        stats.update({(att, 0x78): s120, (tgt, 0x1f): d31, "t2": as2})
        try:
            e.call(0x57b7d0, tgt, rec, 0, 0x80, ecx=0, edx=att)
        except Stop:
            pass
        except Exception as ex:                      # an emulator fault is reported, not hidden
            res.append(("ERR", repr(ex)[:60])); continue
        res.append(tuple(w for w in wr if w[1] == 0x1f) or None)
    return res, tgt


def model_b(c):
    atype, ttype, s120, d31, as2 = c
    if (atype == 0 or as2) and ttype == 1 and s120 != 0:
        return max(0, s32(d31 + s120))               # 32-bit add: INT_MAX + 5 wraps negative and floors to 0
    return None


def gen_b(n, rng):
    out = []
    for atype in (0, 1):
        for as2 in (0, 1):
            for ttype in (0, 1, 2):
                for s120 in (0, -1, -50, -100, 5, -2**31 + 1):
                    for d31 in (0, 1, 49, 50, 51, 99, 100, 1000, 2**31 - 1):
                        out.append((atype, ttype, s120, d31, as2))
    for _ in range(n):
        out.append((rng.choice((0, 0, 0, 1)), rng.choice((1, 1, 1, 0, 2)), rng.choice((0, -10, -25, -40, -50, -70, -100, rng.randrange(-300, 300))),
                    rng.randrange(0, 3000), rng.choice((0, 1))))
    return out


if __name__ == "__main__":
    part, n, seed = sys.argv[1], int(sys.argv[2]), int(sys.argv[3])
    rng = random.Random(seed & M32)
    if part == "a":
        cases = gen_a(n, rng)
        got, bits = run_a(cases, random.Random((seed + 1) & M32))
        bad = live = 0
        for c, g in zip(cases, got):
            w = model_a(c, bits)
            base = model_a(c[:15] + ((),), bits)
            live += w != base
            if g != w:
                bad += 1
                if bad <= 15: print("DIFF", c[:15], len(c[15]), "game", g, "model", w)
        print(f"A: {len(cases)} cases (seed {seed}, {len(cases) - n} boundary), {bad} differ from the reading; the 179 list changed the chance in {live}")
    else:
        cases = gen_b(n, rng)
        got, _ = run_b(cases)
        bad = errs = chg = 0
        for c, g in zip(cases, got):
            if g and g[0] == "ERR": errs += 1; bad += 1; continue
            w = model_b(c)
            gv = g[-1][2] if g else None
            if gv is not None and gv < 0: gv += 2 ** 32
            chg += w is not None
            if (gv is None) != (w is None) or (w is not None and (gv & M32) != (w & M32)):
                bad += 1
                if bad <= 15: print("DIFF", c, "game", g, "model", w)
        print(f"B: {len(cases)} cases (seed {seed}), {bad} differ from the reading ({errs} emulator errors); the write happened in {chg}")
