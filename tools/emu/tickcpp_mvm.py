#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-3.0-or-later
"""Monster-vs-monster 256ths life differential: game.exe 1.14d vs d2d (TICKMVM, #26).

Runs game.exe 1.14d's own FUN_0057b420 (damage roll, monster attacker EDX=1) and
FUN_0057c6c0 (apply damage with flag=1, calling FUN_0057c1e0 reduction and stat 6
update on a type-1 monster defender) natively under Unicorn using upstream
d2decomped's `emu.py` helper.

Compares the native game.exe defender life (stat 6, 256ths) and attacker RNG seed
against a line-for-line Python port of upstream d2decomped (`2b7ec10`):
  - before: `combat.cpp:718-799` (`monster_blow`) -> `fight/strike.cpp:614-621`
    (`Fight::take_blows`, passing `blow.damage << 8`, dropping `blow.frac`) ->
    `ai.cpp:3032-3056` (`hurt`).
  - after:  same path with `blow.frac` carried into `take_blows`:
    `soak(struck, physical_now(blow), (blow.damage << 8) + blow.frac)`.

Usage (with PYTHONPATH=<d2decomped>/tools/emu and D2_MPQ_DIR=<mpq dir>):
    python tools/emu/tickcpp_mvm.py [N_TOTAL] [SEED] [--exe PATH]
    python tools/emu/tickcpp_mvm.py 100000 20261104
"""
import random
import sys
import time

M32 = 0xFFFFFFFF

CLASSES = (
    ("fallen1", 1, 2),
    ("zombie1", 1, 3),
    ("skeleton1", 1, 3),
    ("quillrat1", 1, 2),
    ("corruptrogue1", 1, 3),
    ("brute1", 2, 3),
    ("bloodraven", 4, 6),
    ("andariel", 6, 19),
)

SETS = ("plain", "pct", "flat", "both", "exact", "exactred")


def s32(v):
    v &= M32
    return v - (1 << 32) if v & 0x80000000 else v


def tdiv(a, b):
    q = abs(a) // abs(b)
    return q if (a >= 0) == (b > 0) else -q


def mul_div(a, b, c):
    """Exact port of d2d::rules::mul_div (FUN_00483360)."""
    if c == 0:
        return 0
    return tdiv(a * b, c)


def rng_step(sl, sh):
    """Exact 64-bit LCG step of d2d::rules::Rng / FUN_0045c3e0."""
    prod = sl * 0x6AC690C5 + sh
    return prod & M32, (prod >> 32) & M32


def rng_mod(sl, sh, n):
    sl, sh = rng_step(sl, sh)
    if n <= 0:
        return 0, sl, sh
    return sl % n, sl, sh


def d2d_reduce_fine_signed(fine, percent, flat):
    """Line-for-line port of d2d::rules::reduce_fine_signed (combat.cpp:197-202)."""
    if fine <= 0:
        return 0
    fine -= flat * 256
    if fine > 0 and percent != 0:
        clamped = max(-100, min(100, percent))
        fine = mul_div(fine, 100 - clamped, 100)
    return fine


def d2d_mvm(mn, mx, sl, sh, pct, flat, exact, init_life, carry_frac):
    """Line-for-line port of monster_blow -> Foe::take -> Fight::take_blows -> hurt (2b7ec10).

    Monster life representation in d2d (ai.hpp:523-527):
        hit_points = (life + 255) >> 8
        life_frac  = hit_points * 256 - life   (deficit in 0..255)
    """
    if exact:
        raw = mn * 256
        asl, ash = sl, sh
    else:
        low = max(mn, 1)
        high = max(mx, low + 1)
        r, asl, ash = rng_mod(sl, sh, (high - low) * 256)
        raw = low * 256 + r

    physical_fine = d2d_reduce_fine_signed(raw, pct, flat)
    total = max(physical_fine, 0)
    taken_damage = total >> 8
    taken_frac = total & 255

    # Fight::take_blows (strike.cpp:616):
    #   before (2b7ec10): soak(struck, blow.physical << 8, blow.damage << 8)  -> frac dropped
    #   after  (fix):     soak(struck, physical_now(blow), (blow.damage << 8) + blow.frac)
    applied_fine = (taken_damage << 8) + (taken_frac if carry_frac else 0)
    dmg = applied_fine >> 8
    frac = applied_fine & 255

    # Monster::set_life_fine (ai.hpp:524-527)
    hp = (init_life + 255) >> 8
    lfrac = hp * 256 - init_life

    # hurt() (ai.cpp:3032-3044)
    if dmg > 0 or frac > 0:
        lfrac += max(frac, 0)
        dmg += lfrac >> 8
        lfrac &= 255
        hp -= dmg
        if hp < 1 or (hp == 1 and lfrac > 0):
            hp = 0
            lfrac = 0

    life_after = hp * 256 - lfrac
    return life_after, asl, ash, total


def gen_cases(n_total, seed):
    """Generate n_total deterministic test cases across all 6 classes."""
    r = random.Random(seed)
    rows = []
    # Split: 20% plain, 20% pct, 20% flat, 20% both, 10% exact, 10% exactred
    counts = {
        "plain": n_total // 5,
        "pct": n_total // 5,
        "flat": n_total // 5,
        "both": n_total // 5,
        "exact": n_total // 10,
        "exactred": n_total - 4 * (n_total // 5) - (n_total // 10),
    }

    def pick_life(mx):
        roll = r.random()
        if roll < 0.25:
            # Near-death life (exercises < 0x100 death threshold)
            return r.randint(0x100, max(0x200, (mx + 2) * 256))
        if roll < 0.65:
            # Fractional starting life (subsequent blows after a fractional hit)
            return r.randint(0x100, 200 * 256)
        # Integer starting life
        return r.randint(1, 200) * 256

    for cat in SETS:
        cnt = counts[cat]
        for i in range(cnt):
            if cat == "exact":
                x = i % 6 if i < 60 else r.randint(0, 40)
                mn = mx = x
                pct = flat = 0
                exact = 1
                name = f"x{x}" if x < 6 else "x_rand"
            elif cat == "exactred":
                x = r.randint(0, 40)
                mn = mx = x
                pct = r.choice([0, -100, -50, 25, 50, 75, 100, r.randint(-100, 100)])
                flat = r.randint(0, max(x, 1) + 2)
                if pct == 0 and flat == 0:
                    flat = 1
                exact = 1
                name = "x0_40"
            else:
                name, mn, mx = CLASSES[i % len(CLASSES)]
                exact = 0
                pct = flat = 0
                if cat in ("pct", "both"):
                    pct = r.choice([-100, -50, 10, 25, 50, 75, 100, r.randint(-100, 100)])
                    if pct == 0:
                        pct = 25
                if cat in ("flat", "both"):
                    flat = r.randint(1, mx + 2)
            init_life = pick_life(max(mx, 1))
            sl = r.getrandbits(32)
            sh = r.getrandbits(32)
            rows.append((cat, name, mn, mx, pct, flat, exact, init_life, sl, sh))
    return rows


def run_game(rows, exe_path=None):
    """Run FUN_0057b420 + FUN_0057c6c0 natively on game.exe 1.14d under Unicorn."""
    import emu
    e = emu.Emu(exe_path) if exe_path else emu.Emu()

    st = {}
    game_ptr = e.alloc(0x1000)
    e.w32(game_ptr + 0x70, 1)  # expansion
    e.w32(game_ptr + 0x6C, 0)  # Normal difficulty

    attacker = e.alloc(0x200)
    e.w32(attacker + 0x00, 1)  # type 1 = monster
    e.w32(attacker + 0x04, 1)  # class id
    att_data = e.alloc(0x40)
    att_row = e.alloc(0x400)
    e.w32(attacker + 0x14, att_data)
    e.w32(att_data, att_row)

    defender = e.alloc(0x200)
    e.w32(defender + 0x00, 1)  # type 1 = monster
    e.w32(defender + 0x04, 60)  # class id (monster)
    def_data = e.alloc(0x40)
    def_row = e.alloc(0x400)
    e.w32(defender + 0x14, def_data)
    e.w32(def_data, def_row)

    diffrow = e.alloc(0x80)
    e.w32(diffrow + 0x28, 1)
    e.w32(diffrow + 0x2C, 1)

    rec = e.alloc(0x100)

    # Core stat / unit hooks (same minimal surface as d35_leech.py / w3_run.py)
    e.hook(0x535BC0, lambda e_: 0, 0)  # active weapon (none for monster)
    e.hook(0x625480, lambda e_: st.get((e_.arg(0), e_.arg(1)), 0) & M32, 3)  # GetStat
    e.hook(0x625500, lambda e_: 0, 3)  # GetBaseStat
    e.hook(0x627260, lambda e_: st.__setitem__((e_.arg(0), e_.arg(1)), s32(e_.arg(2))) or 0, 4)  # SetStat
    e.hook(0x625D10, lambda e_: st.get((e_.arg(0), 7), 1 << 30) & M32, 1)  # GetMaxLife
    e.hook(0x611D30, lambda e_: diffrow, 1)  # DifficultyLevels row
    e.hook(0x554010, lambda e_: game_ptr, 0)
    e.hook(0x620BB0, lambda e_: 1, 1)

    # Side-effect / unrelated subsystem stubs for FUN_0057c6c0 / FUN_0057c1e0
    for addr, nargs in (
        (0x63EE90, 1), (0x63E990, 1), (0x63E940, 1), (0x63EDC0, 1), (0x63E9F0, 2),
        (0x5C0C30, 3), (0x5347D0, 7), (0x5347E0, 3), (0x61AB00, 1), (0x5541B0, 0),
        (0x621D50, 2), (0x639DF0, 2), (0x57AA60, 0), (0x57AAA0, 0), (0x57AAE0, 2),
        (0x57AF80, 2), (0x57B230, 2), (0x57AC50, 3), (0x57ADD0, 3), (0x5D6410, 0),
        (0x6256B0, 2), (0x5A54F0, 2), (0x5A5650, 0), (0x5734C0, 0), (0x5A43A0, 0),
        (0x5A7C20, 3), (0x621E40, 3), (0x57A980, 1), (0x57AA00, 1),
        (0x5A7E60, 3), (0x5809D0, 5), (0x64C040, 1),
    ):
        e.hook(addr, lambda e_: 0, nargs)
    for addr, nargs in ((0x554200, 1), (0x457490, 0)):
        e.hook(addr, lambda e_: 1, nargs)
    e.hook(0x46C140, lambda e_: 1 if e_.mu.reg_read(emu.UC_X86_REG_EDX) == 3 else 0, 0)

    out = []
    zero_rec = b"\x00" * 0x100
    for _, _, mn, mx, pct, flat, exact, init_life, sl, sh in rows:
        st.clear()
        st[(attacker, 0x15)] = mn
        st[(attacker, 0x16)] = mx
        st[(defender, 6)] = init_life
        st[(defender, 7)] = max(init_life, 200 * 256)
        st[(defender, 0x22)] = flat   # stat 34: normal_damage_reduction
        st[(defender, 0x24)] = pct    # stat 36: damageresist

        e.w32(attacker + 0x20, sl)
        e.w32(attacker + 0x24, sh)
        if exact:
            v = mn << 8
            asl, ash = sl, sh
        else:
            v = s32(e.call(0x57B420, 0, 0, 0, 0, 0, 0x80, ecx=attacker, edx=1))
            asl = e.r32(attacker + 0x20)
            ash = e.r32(attacker + 0x24)

        e.mu.mem_write(rec, zero_rec)
        e.w32(rec + 0x00, 1)
        e.w32(rec + 0x04, 1)   # result_flags bit 0 = landed hit
        e.w32(rec + 0x08, v & M32)
        e.call(0x57C6C0, defender, 1, rec, ecx=game_ptr, edx=attacker)

        life_after = st[(defender, 6)]
        rec_total = s32(e.r32(rec + 0x4C))
        out.append((life_after, asl, ash, rec_total, v))
    return out


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
    rows = gen_cases(n_total, seed)
    game_out = run_game(rows, exe_path=exe_path)
    elapsed = time.time() - t0

    print(f"# tickcpp_mvm.py N={len(rows)} seed={seed} elapsed={elapsed:.2f}s")
    print("# class | cases | before_life_diff before_dead_diff before_seed_diff before_mean_err_hp | after_life_diff after_dead_diff after_seed_diff")

    tot = {k: 0 for k in ("n", "b_life", "b_dead", "b_seed", "b_err_sum", "a_life", "a_dead", "a_seed", "frac_kept_game")}
    first_diff = {}

    for cat in SETS:
        c = {k: 0 for k in ("n", "b_life", "b_dead", "b_seed", "b_err_sum", "a_life", "a_dead", "a_seed")}
        for i, row in enumerate(rows):
            if row[0] != cat:
                continue
            _, name, mn, mx, pct, flat, exact, init_life, sl, sh = row
            g_life, g_sl, g_sh, g_tot, g_v = game_out[i]
            if g_life > 0 and (g_life & 255) != 0 and (init_life & 255) == 0:
                tot["frac_kept_game"] += 1

            b_life, b_sl, b_sh, _ = d2d_mvm(mn, mx, sl, sh, pct, flat, exact, init_life, carry_frac=False)
            a_life, a_sl, a_sh, _ = d2d_mvm(mn, mx, sl, sh, pct, flat, exact, init_life, carry_frac=True)

            c["n"] += 1
            if b_life != g_life:
                c["b_life"] += 1
                c["b_err_sum"] += abs(b_life - g_life)
                if cat not in first_diff:
                    first_diff[cat] = (i, row, game_out[i], (b_life, b_sl, b_sh), (a_life, a_sl, a_sh))
            if (b_life == 0) != (g_life == 0):
                c["b_dead"] += 1
            if (b_sl, b_sh) != (g_sl, g_sh):
                c["b_seed"] += 1

            if a_life != g_life:
                c["a_life"] += 1
            if (a_life == 0) != (g_life == 0):
                c["a_dead"] += 1
            if (a_sl, a_sh) != (g_sl, g_sh):
                c["a_seed"] += 1

        mean_hp = (c["b_err_sum"] / 256.0 / c["n"]) if c["n"] else 0.0
        print(f"{cat:8s} | {c['n']:6d} | {c['b_life']:6d} {c['b_dead']:5d} {c['b_seed']:4d} {mean_hp:.4f} | {c['a_life']:6d} {c['a_dead']:5d} {c['a_seed']:4d}")
        for k in ("n", "b_life", "b_dead", "b_seed", "b_err_sum", "a_life", "a_dead", "a_seed"):
            tot[k] += c[k]

    tot_mean_hp = (tot["b_err_sum"] / 256.0 / tot["n"]) if tot["n"] else 0.0
    print(f"TOTAL    | {tot['n']:6d} | {tot['b_life']:6d} {tot['b_dead']:5d} {tot['b_seed']:4d} {tot_mean_hp:.4f} | {tot['a_life']:6d} {tot['a_dead']:5d} {tot['a_seed']:4d}")
    print(f"# game.exe integer-start cases ending with fractional stat 6 (non-zero 256ths): {tot['frac_kept_game']}")
    for cat in SETS:
        if cat in first_diff:
            i, row, go, bo, ao = first_diff[cat]
            print(f"# first_diff {cat}: case={i} name={row[1]} mn={row[2]} mx={row[3]} pct={row[4]} flat={row[5]} exact={row[6]} init_life={row[7]} | game_life={go[0]} rec_tot={go[3]} | d2d_before={bo[0]} d2d_after={ao[0]}")

    return 1 if (tot["a_life"] > 0 or tot["a_seed"] > 0) else 0


if __name__ == "__main__":
    sys.exit(main())
