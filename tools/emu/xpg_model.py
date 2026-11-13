#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-3.0-or-later
"""XPGOLD (decomp14): per-area XP and gold of a kill and a clvl-by-minute model, from game.exe 1.14d numbers.

Modes (host-independent, plain python3; inputs are files produced on the worker host, see docs/research/xp-gold-trace.md):

  xpg_model.py check GRID          GRID = `ms_run.sh xp_award.py grid` output (game.exe's FUN_0057e2f0 under the emulator,
                                   lines `exp clvl mlvl award`). Compares every line with award() below (our port of the
                                   two 256ths tables + the clvl 25 rule) and prints the number of differing lines.
  xpg_model.py area SPAWN CLVLS    SPAWN = `ms_run.sh ms_spawn.py SEEDS LIDS 0` U lines (every monster unit game.exe makes for
                                   a populated Normal level: flags, level stat 0xc, experience stat 0xd). Prints per area:
                                   units by kind, mean base exp and the mean award at each CLVL (integer, as the game).
  xpg_model.py sim SPAWN           clvl-by-minute for named schedules (see SCHEDULES), kill rate given per schedule.

Experience.txt thresholds (patched 1.14d, row n = experience to reach level n+1) are in LEVEL_XP.
"""
import sys

BELOW = [256, 256, 256, 256, 256, 256, 207, 159, 110, 61, 13]   # 0x6e1668, clvl >= mlvl, index min(clvl - mlvl, 10)
ABOVE = [256, 256, 256, 256, 256, 256, 225, 174, 92, 38, 5]     # 0x6e1694, mlvl > clvl (clvl < 25)

# Experience.txt, Paladin column, rows 1..: experience to reach level 2, 3, ... (D, mpq-cat of the patched stack).
LEVEL_XP = [500, 1500, 3750, 7875, 14175, 22680, 32886, 44396, 57715, 72144, 90180, 112725, 140906, 176132, 220165]


def muldiv(a, b, c):
    """FUN_00483360 (V read, xp-award-trace.md): the two shortcuts for large operands, else 64-bit multiply then divide."""
    if a > 0x100000 and c <= a >> 4: return (a // c) * b
    if b > 0x10000 and c <= b >> 4: return (b // c) * a
    return a * b // c


def award(exp, clvl, mlvl):
    if clvl >= mlvl:
        t = BELOW[min(clvl - mlvl, 10)]
    elif clvl >= 25:
        return muldiv(exp, clvl, mlvl)
    else:
        t = ABOVE[min(mlvl - clvl, 10)]
    return exp if t == 256 else muldiv(exp, t, 256)


def clvl_of(xp):
    n = 1
    for need in LEVEL_XP:
        if xp >= need:
            n += 1
    return n


def kind(flags):
    if flags & 8: return "unique"
    if flags & 4: return "champion"
    if flags & 0x10: return "minion"
    if flags & 2: return "super"
    return "plain"


def load(spawn):
    areas = {}
    for line in open(spawn):
        f = line.split()
        if not f or f[0] != "U":
            continue
        lid, flags, statlvl = int(f[1]), int(f[5], 16), f[6]
        if statlvl == "None":
            continue
        st = {}
        for tok in f[8:]:
            k, _, v = tok.partition(":")
            st[int(k, 16)] = int(v)
        if st.get(0xd, 0) <= 0:
            continue
        areas.setdefault(lid, []).append((kind(flags), st[0xd], st.get(0xc, int(statlvl))))
    return areas


def cmd_check(grid):
    n = bad = 0
    for line in open(grid):
        e, c, m, a = map(int, line.split())
        n += 1
        if award(e, c, m) != a:
            bad += 1
    print(f"lines {n}, differ {bad}")


def cmd_area(spawn, clvls):
    areas = load(spawn)
    print("lid kind units mean_exp mean_lvl | mean award at clvl " + " ".join(map(str, clvls)))
    for lid in sorted(areas):
        rows = areas[lid]
        for k in ("plain", "all", "champion", "minion", "unique", "super"):
            sel = [r for r in rows if k == "all" or r[0] == k]
            if not sel:
                continue
            n = len(sel)
            cells = " ".join(f"{sum(award(e, c, l) for _, e, l in sel) / n:.1f}" for c in clvls)
            print(f"{lid} {k} {n} {sum(r[1] for r in sel) / n:.1f} {sum(r[2] for r in sel) / n:.1f} | {cells}")


def area_award(rows, clvl):
    return sum(award(e, clvl, l) for _, e, l in rows) / len(rows)


# (minutes, area lid, kills per minute): the observed hour and two counterfactuals. Rates: observed hour 528 kills / 60 min = 8.8;
# in-field rates from docs/probes/hour1-analysis-2026-11-12.md (Blood Moor 15 early, Cold Plains 14.6, Den 13.5 early).
SCHEDULES = {
    "observed-like (BM 12, Den 14, BM 13, Den 5, CP 10; town 6 min)": [(12, 2, 11), (14, 8, 9.5), (6, 0, 0), (13, 2, 7.4), (5, 8, 8.6), (10, 3, 14.6)],
    "ideal: BM 5, Den 10, CP 45 (14.6 kills/min)": [(5, 2, 15), (10, 8, 13.5), (45, 3, 14.6)],
    "ideal at 25 kills/min: BM 4, Den 6, CP 20, Stony 30": [(4, 2, 25), (6, 8, 25), (20, 3, 25), (30, 4, 25)],
}


def cmd_sim(spawn):
    areas = load(spawn)
    plain = {lid: [r for r in rows if r[0] == "plain"] for lid, rows in areas.items()}
    allk = areas
    for name, sched in SCHEDULES.items():
        for label, pool in (("plain kills only", plain), ("every unit as it is spawned (champions, packs)", allk)):
            xp, t, out = 0.0, 0, []
            for mins, lid, rate in sched:
                for _ in range(mins):
                    if lid:
                        xp += rate * area_award(pool[lid], clvl_of(xp))
                    t += 1
                    if t % 10 == 0:
                        out.append(f"{t}:{clvl_of(xp)}")
            print(f"{name} | {label}: xp {xp:.0f}, clvl {clvl_of(xp)} | " + " ".join(out))


# Per-kill supply by kind (treasure-trace.md section 5, V run exact odds): gold piles, coin multiplier, potions.
SUPPLY = {"plain": (0.131, 1, 0.057), "minion": (0.131, 1, 0.057), "champion": (0.6, 5, 2.0), "unique": (0.0, 1, 4.0), "super": (0.0, 1, 4.0)}


def cmd_gold(spawn, clvl):
    """Per area: mean gold and potions per kill (game.exe pile = ilvl + rand(5 ilvl), mean 3.5 ilvl - 0.5; ilvl = unit level)
    and the XP at CLVL, for plain kills and for the whole population. Casters/archers (0.1125 piles) are counted as 0.131: I."""
    areas = load(spawn)
    print(f"lid | plain: n xp@{clvl} gold potions | all units: n xp@{clvl} gold potions")
    for lid in sorted(areas):
        out = []
        for sel in ([r for r in areas[lid] if r[0] == "plain"], areas[lid]):
            n = len(sel)
            xp = sum(award(e, clvl, l) for _, e, l in sel) / n
            gold = sum(SUPPLY[k][0] * SUPPLY[k][1] * (3.5 * max(l, 1) - 0.5) for k, _, l in sel) / n
            pot = sum(SUPPLY[k][2] for k, _, _ in sel) / n
            out.append(f"{n} {xp:.1f} {gold:.2f} {pot:.3f}")
        print(f"{lid} | " + " | ".join(out))


if __name__ == "__main__":
    m = sys.argv[1]
    if m == "check": cmd_check(sys.argv[2])
    elif m == "area": cmd_area(sys.argv[2], [int(x) for x in sys.argv[3].split(",")])
    elif m == "gold": cmd_gold(sys.argv[2], int(sys.argv[3]))
    elif m == "sim": cmd_sim(sys.argv[2])
    else: sys.exit(__doc__)
