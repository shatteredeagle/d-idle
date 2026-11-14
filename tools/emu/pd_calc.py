#!/usr/bin/env python3
"""Card playerdef: game.exe 1.14d's player block chance, defence and attack rating, run natively, against d2d's formulas.

Runs under tools/emu/run-on-worker.sh (upstream's emu.py harness, unicorn): the three game.exe functions are executed from
their first instruction, only their leaf helpers are hooked (the stat getters FUN_00625480 / FUN_00625500 return the case's value by
stat id; the shield test FUN_0063c8f0 and the state test FUN_00639df0 return the case's flag; FUN_00483360 and the signed
division run natively):

  FUN_00622720(unit, scaled)  block chance: a player needs a shield; b = stat 20 + CharStats BlockFactor (record +0x49);
                              scaled != 0: (dex - 15) x b / (max(clvl,1) x 2) (idiv), at most 0x4b; no lower clamp.
  FUN_006223f0(unit)          defence: (stat 31 + dex(stat 2) / 4 (cdq/and 3/sar: toward zero)) then x (stat 171 + stat 16) %, + stat 182.
  FUN_00622560(unit)          attack rating of a player: stat 19 + 5 x (dex - 7) + CharStats record +0x3c.

The d2d side is a line-for-line port of components/rules/combat.hpp (make_fighter: block, ar_base) and components/game/character.hpp
(panel_stats: defense) with C++ integer semantics (truncation toward zero); it is NOT the compiled d2d code.

  tools/emu/run-on-worker.sh pd_calc.py 200000 7 > $D2_SCRATCH/pd_calc.txt
"""
import random
import sys


def tdiv(a, b):
    q = abs(a) // abs(b)
    return q if (a >= 0) == (b > 0) else -q


def d2d_block(shield_block, bf, tob, dex, clvl):
    if shield_block <= 0:
        return 0
    clvl = max(clvl, 1)
    return min(max(tdiv((shield_block + bf + tob) * (dex - 15), clvl * 2), 0), 75)


def d2d_ar(dex, to_hit_class, tohit):
    return dex * 5 - 35 + to_hit_class + tohit


def d2d_def(base, dex, pct):
    d = base + tdiv(dex, 4)
    return d + tdiv(d * pct, 100)


def main():
    n = int(sys.argv[1]) if len(sys.argv) > 1 else 20000
    seed = int(sys.argv[2]) if len(sys.argv) > 2 else 7
    import emu
    e = emu.Emu(__import__("os").path.join(__import__("os").environ.get("D2_EMU_DIR", __import__("os").path.expanduser("~/d2d-lab/emu")), "casttrace", "game.exe"))
    stats = {}
    flags = {"shield": 1}
    e.hook(0x625480, lambda e_: stats.get(e_.arg(1), 0) & 0xFFFFFFFF, 3)
    e.hook(0x625500, lambda e_: stats.get(e_.arg(1), 0) & 0xFFFFFFFF, 3)
    e.hook(0x63c8f0, lambda e_: flags["shield"], 2)
    e.hook(0x639df0, lambda e_: 0, 2)
    unit = e.alloc(0x200)
    gp = e.alloc(0x2000)
    tbl = e.alloc(0xc4 * 8)
    e.w32(0x744304, gp)
    e.w32(gp + 0xbc4, tbl)
    e.w32(gp + 0xbc8, 8)
    rng = random.Random(seed)
    cases = []
    for dex in range(-5, 301, 5):
        for clvl in (0, 1, 2, 9, 30, 60, 99):
            for b in (0, 20, 30, 61, 75, 100):
                cases.append((dex, clvl, b))
    for _ in range(n):
        cases.append((rng.randrange(-20, 400), rng.randrange(0, 100), rng.randrange(0, 130)))
    bad = {"block": 0, "block_eff": 0, "def": 0, "ar": 0}
    shown = 0
    for i, (dex, clvl, b) in enumerate(cases):
        cls = i % 7
        bf = rng.choice((0, 5, 12, 15, 20, 25, 30)) if i % 3 else 0
        e.w32(tbl + 0xc4 * cls + 0x49, bf)
        tohit_cls = rng.choice((0, 5, 10, 15, 20))
        e.w32(tbl + 0xc4 * cls + 0x3c, tohit_cls)
        for k in range(0, 0x200, 4):
            e.w32(unit + k, 0)
        e.w32(unit, 0)
        e.w32(unit + 4, cls)
        sh = 1 if i % 5 else 0
        flags["shield"] = sh
        s20 = b
        stats.clear()
        stats.update({2: dex, 12: clvl, 20: s20, 19: rng.randrange(0, 400), 31: rng.randrange(-30, 3000),
                      171: rng.choice((0, 0, 15, 40, 100, -20)), 16: rng.choice((0, 0, 30, 120, -10)), 182: 0})
        # block (scaled)
        g = e.call(0x622720, unit, 1)
        g = g - (1 << 32) if g >= 1 << 31 else g
        # the stat 20 inside game.exe is (shield base + gear); d2d adds the shield base + gains + gear: same sum
        dp = d2d_block(1, bf, s20 - 1, dex, clvl) if sh else 0
        ge = max(g, 0) if sh else g
        if sh and g != dp:
            bad["block"] += 1
            if max(g, 0) != dp:
                bad["block_eff"] += 1
            if shown < 15:
                print(f"block differ id {i}: dex {dex} clvl {clvl} stat20 {s20} bf {bf}: game {g} d2d {dp}")
                shown += 1
        if not sh and g != 0 and shown < 15:
            print(f"noshield id {i}: game {g}")
        # defence
        g = e.call(0x6223f0, unit)
        g = g - (1 << 32) if g >= 1 << 31 else g
        pct = stats[171] + stats[16]
        dp = d2d_def(stats[31], dex, pct)
        if stats[31] + tdiv(dex, 4) > 0:
            bad["def_pos_cases"] = bad.get("def_pos_cases", 0) + 1
        if g != dp:
            bad["def"] += 1
            if stats[31] + tdiv(dex, 4) > 0:
                bad["def_pos"] = bad.get("def_pos", 0) + 1
            if shown < 30:
                print(f"def differ id {i}: base {stats[31]} dex {dex} pct {pct}: game {g} d2d {dp}")
                shown += 1
        # attack rating
        g = e.call(0x622560, unit)
        g = g - (1 << 32) if g >= 1 << 31 else g
        dp = d2d_ar(dex, tohit_cls, stats[19])
        if g != dp:
            bad["ar"] += 1
            if shown < 40:
                print(f"ar differ id {i}: dex {dex} stat19 {stats[19]} class {tohit_cls}: game {g} d2d {dp}")
                shown += 1
    print(f"cases {len(cases)} block mismatches {bad['block']} (of which differ after floor at 0: {bad['block_eff']}) "
          f"defence mismatches {bad['def']} (of {bad.get('def_pos_cases', 0)} cases with positive base: {bad.get('def_pos', 0)}) attack-rating mismatches {bad['ar']}")


if __name__ == "__main__":
    main()
