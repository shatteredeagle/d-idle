#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-3.0-or-later
"""Player defence differentials D4 (magic resist), D5 (resist clamp order), and
D6 (DR order of stats 34/35/36): game.exe 1.14d vs d2d (PLAYERDIFF, #24).

Runs game.exe 1.14d's own FUN_0057c1e0 (calling FUN_0057bf80 and FUN_0057be00
natively on a type-0 player defender) under Unicorn using upstream d2decomped's
`tools/emu/emu.py` helper (commit `5a81bcc`).

Compares native game.exe outputs (record fields + total at rec+0x4c, in 256ths)
against a line-for-line Python port of upstream d2decomped (`5a81bcc`):
  - D4 (magic resist): `combat/combat.cpp:1003` hardcodes `res = 0, resist = false`
    for `type == 4` (magic), ignoring player stats 37/38.
  - D5 (resist clamp order): `character/character.cpp:218-219` (`panel_stats`)
    clamps `gear + penalty` to `[-100, min(75 + gear_max, 95)]` BEFORE
    `fight/strike.cpp:288-290` (`update_fighters`) adds aura/curse/potion states
    with only an upper cap (no `-100` floor in `strike.cpp:290` or `resisted_fine`).
  - D6 (DR order of stats 34/35/36): `combat/combat.cpp:139, 269-274, 314-319`
    (`make_fighter`, `reduce_fine_by`, `elemental_fine`).

Usage (with PYTHONPATH=<d2decomped>/tools/emu and D2_MPQ_DIR=<mpq dir>):
    python tools/emu/playerdef_resist_dr.py [N_TOTAL] [SEED] [--exe PATH]
    python tools/emu/playerdef_resist_dr.py 100000 20261104
"""
import random
import sys
import time

M32 = 0xFFFFFFFF

# Patched 1.14d LoD DifficultyLevels.txt ResistPenalty (Normal, Nightmare, Hell; D)
LOD_RESIST_PENALTIES = (0, -40, -100)

# Element type -> (record offset, resist stat, max-resist stat)
ELEM_ROWS = {
    0: (0x10, 0x27, 0x28),  # fire: stats 39, 40
    1: (0x1C, 0x29, 0x2A),  # lightning: stats 41, 42
    2: (0x24, 0x2B, 0x2C),  # cold: stats 43, 44
    4: (0x20, 0x25, 0x26),  # magic: stats 37, 38
}


def s32(v):
    v &= M32
    return v - (1 << 32) if v & 0x80000000 else v


def tdiv(a, b):
    q = abs(a) // abs(b)
    return q if (a >= 0) == (b > 0) else -q


def mul_div(a, b, c):
    """Exact port of d2d::rules::mul_div / FUN_00483360."""
    if c == 0:
        return 0
    return tdiv(a * b, c)


def d2d_resisted_fine(fine, res):
    """Line-for-line port of d2d::rules::resisted_fine (combat.cpp:263-267)."""
    if fine <= 0 or res >= 100:
        return 0
    if res == 0:
        return fine
    if fine > 0x100000:
        val = s32(((fine // 100) & M32) * ((100 - res) & M32))
    else:
        val = tdiv(fine * (100 - res), 100)
    return max(val, 0)


def d2d_reduce_fine_signed(fine, pct, flat):
    """Line-for-line port of d2d::rules::reduce_fine_signed (combat.cpp:269-274)."""
    if fine <= 0:
        return 0
    fine -= flat * 256
    if fine > 0 and pct != 0:
        clamped = max(-100, min(100, pct))
        fine = mul_div(fine, 100 - clamped, 100)
    return fine


def d2d_elemental_fine(field, mdr, res, apply_resist):
    """Line-for-line port of d2d::rules::elemental_fine (combat.cpp:314-319, no gear absorb)."""
    if field <= 0:
        return 0
    field -= mdr * 256
    if field > 0 and apply_resist:
        field = d2d_resisted_fine(field, res)
    return field


def d2d_d4(field_pts, mdr, stat37, stat38, fix=False):
    """D4: player magic resist (combat.cpp:1003)."""
    if not fix:
        # Current 5a81bcc: hardcodes res=0, resist=false for type == 4
        fine = d2d_elemental_fine(field_pts << 8, mdr, 0, False)
    else:
        cap = min(75 + stat38, 95)
        eff = max(-100, min(cap, stat37))
        fine = d2d_elemental_fine(field_pts << 8, mdr, eff, True)
    return fine, fine


def d2d_d5(field_pts, mdr, gear_res, state_res, gear_max, state_max, penalty, fix=False):
    """D5: player elemental resist clamp order (character.cpp:218-219 + strike.cpp:288-290)."""
    if not fix:
        # Current 5a81bcc:
        # 1) panel_stats (character.cpp:218-219) clamps gear + penalty first
        panel_cap = min(75 + gear_max, 95)
        panel_res = max(-100, min(panel_cap, gear_res + penalty))
        # 2) update_fighters (strike.cpp:288-290) adds state_res with upper cap only (no -100 floor!)
        if state_res != 0 or state_max != 0:
            eff = min(panel_res + state_res, min(panel_cap + state_max, 95))
        else:
            eff = panel_res
    else:
        # Proposed single clamp after summing gear + state + penalty
        cap = min(75 + gear_max + state_max, 95)
        eff = max(-100, min(cap, gear_res + state_res + penalty))
    fine = d2d_elemental_fine(field_pts << 8, mdr, eff, True)
    return fine, fine


def d2d_d6(phys_fine, fire_pts, dr_flat, mdr, dr_pct_raw, fire_res_raw, penalty, fix=False):
    """D6: DR order of stats 34/35/36 (combat.cpp:139, 269-274, 314-319)."""
    # make_fighter (combat.cpp:139): dr_pct = min(stat36, 50)
    dr_pct = min(dr_pct_raw, 50)
    if fix:
        dr_pct = max(-100, dr_pct)
    p_fine = d2d_reduce_fine_signed(phys_fine, dr_pct, dr_flat)

    # Fire field with MDR (stat 35) then resist (stat 39 + penalty, clamped [-100, 75])
    f_eff = max(-100, min(75, fire_res_raw + penalty))
    f_fine = d2d_elemental_fine(fire_pts << 8, mdr, f_eff, True)
    return p_fine, f_fine, p_fine + f_fine


def gen_cases(n_total, seed):
    """Generate n_total cases across D4 (30%), D5 (40%), and D6 (30%)."""
    r = random.Random(seed)
    n_d4 = (n_total * 3) // 10
    n_d5 = (n_total * 4) // 10
    n_d6 = n_total - n_d4 - n_d5

    d4_rows = []
    for _ in range(n_d4):
        diff = r.randrange(3)
        field = r.choice((1, 5, 20, r.randint(1, 200), r.randint(1, 500)))
        mdr = r.choice((0, 0, r.randint(1, 20), r.randint(1, field + 5)))
        stat37 = r.choice((0, 10, 25, 50, 75, 85, 100, -50, -100, r.randint(-120, 120)))
        stat38 = r.choice((0, 0, 5, 10, 20, 25))
        d4_rows.append((diff, field, mdr, stat37, stat38))

    d5_rows = []
    for _ in range(n_d5):
        diff = r.randrange(3)
        etype = r.choice((0, 1, 2))
        field = r.choice((5, 20, r.randint(1, 200), r.randint(1, 500)))
        mdr = r.choice((0, 0, r.randint(1, 15)))
        gear_res = r.choice((0, 50, 75, 100, 120, 150, -20, r.randint(-30, 160)))
        state_res = r.choice((0, -50, -80, -100, 30, 60, r.randint(-120, 100)))
        gear_max = r.choice((0, 0, 5, 10, 20))
        state_max = r.choice((0, 0, 5, 10))
        d5_rows.append((diff, etype, field, mdr, gear_res, state_res, gear_max, state_max))

    d6_rows = []
    for _ in range(n_d6):
        diff = r.randrange(3)
        phys_fine = r.choice((0, r.randint(1, 40 * 256), r.randint(1, 200 * 256)))
        fire_pts = r.choice((0, 0, r.randint(1, 150)))
        if phys_fine == 0 and fire_pts == 0:
            phys_fine = 10 * 256
        dr_flat = r.choice((0, 1, 5, 15, r.randint(0, 60)))
        mdr = r.choice((0, 1, 5, 15, r.randint(0, 60)))
        dr_pct = r.choice((0, 10, 25, 50, 60, 75, -50, -100, -120, r.randint(-120, 100)))
        fire_res = r.choice((0, 25, 75, -50, r.randint(-50, 120)))
        d6_rows.append((diff, phys_fine, fire_pts, dr_flat, mdr, dr_pct, fire_res))

    return d4_rows, d5_rows, d6_rows


def main():
    args = sys.argv[1:]
    exe_path = None
    if "--exe" in args:
        idx = args.index("--exe")
        exe_path = args[idx + 1]
        args = args[:idx] + args[idx + 2:]

    n_total = int(args[0]) if len(args) > 0 else 100000
    seed = int(args[1]) if len(args) > 1 else 20261104

    t0 = time.time()
    import emu
    e = emu.Emu(exe_path) if exe_path else emu.Emu()
    penalties = LOD_RESIST_PENALTIES

    stats = {}
    game_ptr = e.alloc(0x1000)
    e.w32(game_ptr + 0x70, 1)  # expansion = 1

    # DifficultyLevels rows (ResistPenalty at offset +0x00, V read FUN_0057be00)
    diffrows = []
    for d in range(3):
        dr = e.alloc(0x80)
        e.w32(dr + 0x00, penalties[d] & M32)
        diffrows.append(dr)

    attacker = e.alloc(0x200)
    e.w32(attacker + 0x00, 1)  # monster
    e.w32(attacker + 0x04, 0x10)

    defender = e.alloc(0x200)
    e.w32(defender + 0x00, 0)  # player
    e.w32(defender + 0x04, 1)  # Sorceress class
    e.w32(defender + 0x10, 1)  # mode

    rec = e.alloc(0x100)
    zero_rec = b"\x00" * 0x100

    # Minimal hooks matching upstream monster_damage.py / gear_absorb.py
    e.hook(0x625480, lambda e_: stats.get((e_.arg(0), e_.arg(1)), 0) & M32, 3)
    e.hook(0x611D30, lambda e_: diffrows[min(e_.arg(0) & 0xFF, 2)], 1)
    e.stub(100, {0x57C060: 0})  # pair_pct = 100
    e.stub(0, {
        0x625500: 3, 0x535BC0: 0, 0x620BB0: 1, 0x639DF0: 2, 0x639E30: 3,
        0x6277E0: 2, 0x626CD0: 1, 0x40B480: 3, 0x63EE90: 1, 0x63E990: 1,
        0x63E940: 1, 0x5C0C30: 3, 0x6256B0: 2,
    })

    d4_rows, d5_rows, d6_rows = gen_cases(n_total, seed)

    # --- D4: player magic resist (stat 37 / 38) ---
    d4_b = d4_a = 0
    d4_first = None
    for i, (diff, field, mdr, s37, s38) in enumerate(d4_rows):
        e.w32(game_ptr + 0x6C, diff << 8)
        stats.clear()
        stats[(defender, 0x23)] = mdr
        stats[(defender, 0x25)] = s37
        stats[(defender, 0x26)] = s38
        e.mu.mem_write(rec, zero_rec)
        e.w32(rec + 0x04, 1 | 4)
        e.w32(rec + 0x20, field << 8)
        e.call(0x57C1E0, defender, rec, ecx=game_ptr, edx=attacker)
        gf, gt = s32(e.r32(rec + 0x20)), s32(e.r32(rec + 0x4C))
        bf, bt = d2d_d4(field, mdr, s37, s38, fix=False)
        af, at = d2d_d4(field, mdr, s37, s38, fix=True)
        if (bf, bt) != (gf, gt):
            d4_b += 1
            if d4_first is None:
                d4_first = (i, (diff, field, mdr, s37, s38), (gf, gt), (bf, bt), (af, at))
        if (af, at) != (gf, gt):
            d4_a += 1

    # --- D5: player elemental resist clamp order ---
    d5_b = d5_a = d5_overcap = d5_underflow = d5_early_floor = 0
    d5_first = None
    for i, (diff, etype, field, mdr, gres, sres, gmax, smax) in enumerate(d5_rows):
        off, rstat, mstat = ELEM_ROWS[etype]
        pen = penalties[diff]
        e.w32(game_ptr + 0x6C, diff << 8)
        stats.clear()
        stats[(defender, 0x23)] = mdr
        stats[(defender, rstat)] = gres + sres
        stats[(defender, mstat)] = gmax + smax
        e.mu.mem_write(rec, zero_rec)
        e.w32(rec + 0x04, 1 | 4)
        e.w32(rec + off, field << 8)
        e.call(0x57C1E0, defender, rec, ecx=game_ptr, edx=attacker)
        gf, gt = s32(e.r32(rec + off)), s32(e.r32(rec + 0x4C))
        bf, bt = d2d_d5(field, mdr, gres, sres, gmax, smax, pen, fix=False)
        af, at = d2d_d5(field, mdr, gres, sres, gmax, smax, pen, fix=True)
        if (bf, bt) != (gf, gt):
            d5_b += 1
            pcap = min(75 + gmax, 95)
            pres = max(-100, min(pcap, gres + pen))
            if gres + pen > pcap and sres < 0:
                d5_overcap += 1
            elif pres + sres < -100:
                d5_underflow += 1
            elif gres + pen < -100 and sres > 0:
                d5_early_floor += 1
            if d5_first is None:
                d5_first = (i, (diff, etype, field, mdr, gres, sres, gmax, smax, pen), (gf, gt), (bf, bt), (af, at))
        if (af, at) != (gf, gt):
            d5_a += 1

    # --- D6: DR order of stats 34 / 35 / 36 ---
    d6_b = d6_a = 0
    d6_first = None
    d6_first_a = None
    for i, (diff, pfine, fpts, dflat, mdr, dpct, fres) in enumerate(d6_rows):
        pen = penalties[diff]
        e.w32(game_ptr + 0x6C, diff << 8)
        stats.clear()
        stats[(defender, 0x22)] = dflat  # stat 34
        stats[(defender, 0x23)] = mdr    # stat 35
        stats[(defender, 0x24)] = dpct   # stat 36
        stats[(defender, 0x27)] = fres   # stat 39
        e.mu.mem_write(rec, zero_rec)
        e.w32(rec + 0x04, 1 | 4)
        e.w32(rec + 0x08, pfine & M32)
        e.w32(rec + 0x10, fpts << 8)
        e.call(0x57C1E0, defender, rec, ecx=game_ptr, edx=attacker)
        gp, gf, gt = s32(e.r32(rec + 0x08)), s32(e.r32(rec + 0x10)), s32(e.r32(rec + 0x4C))
        bp, bf, bt = d2d_d6(pfine, fpts, dflat, mdr, dpct, fres, pen, fix=False)
        ap, af, at = d2d_d6(pfine, fpts, dflat, mdr, dpct, fres, pen, fix=True)
        if (bp, bf, bt) != (gp, gf, gt):
            d6_b += 1
            if d6_first is None:
                d6_first = (i, (diff, pfine, fpts, dflat, mdr, dpct, fres, pen), (gp, gf, gt), (bp, bf, bt))
        if (ap, af, at) != (gp, gf, gt):
            d6_a += 1
            if d6_first_a is None:
                d6_first_a = (i, (diff, pfine, fpts, dflat, mdr, dpct, fres, pen), (gp, gf, gt), (ap, af, at))

    elapsed = time.time() - t0
    print(f"# playerdef_resist_dr.py N={n_total} seed={seed} penalties={penalties} elapsed={elapsed:.2f}s")
    print(f"D4 magic_resist | cases={len(d4_rows)} | before_diff={d4_b} | after_diff={d4_a}")
    print(f"D5 resist_clamp | cases={len(d5_rows)} | before_diff={d5_b} (overcap={d5_overcap} underflow={d5_underflow} early_floor={d5_early_floor}) | after_diff={d5_a}")
    print(f"D6 dr_order     | cases={len(d6_rows)} | before_diff={d6_b} | after_diff={d6_a}")
    print(f"TOTAL           | cases={n_total} | before_diff={d4_b + d5_b + d6_b} | after_diff={d4_a + d5_a + d6_a}")
    if d4_first:
        print(f"# first_diff D4: case={d4_first[0]} in={d4_first[1]} game={d4_first[2]} before={d4_first[3]} after={d4_first[4]}")
    if d5_first:
        print(f"# first_diff D5: case={d5_first[0]} in={d5_first[1]} game={d5_first[2]} before={d5_first[3]} after={d5_first[4]}")
    if d6_first:
        print(f"# first_diff D6: case={d6_first[0]} in={d6_first[1]} game={d6_first[2]} before={d6_first[3]}")
    if d6_first_a:
        print(f"# first_diff D6_after: case={d6_first_a[0]} in={d6_first_a[1]} game={d6_first_a[2]} after={d6_first_a[3]}")

    return 1 if (d4_a > 0 or d5_a > 0 or d6_a > 0) else 0


if __name__ == "__main__":
    sys.exit(main())
