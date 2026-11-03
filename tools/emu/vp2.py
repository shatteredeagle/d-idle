#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-3.0-or-later
"""vp2 (card verifypalskill2): an independent check of the PALSKILL trace.

    python vp2.py gen N SEED > cases.tsv        # own generator, lines in pal2_test's format (C and S only)
    python vp2.py game cases.tsv > game.tsv     # game.exe 1.14d under unicorn: FUN_00646ca0, FUN_006449f0, FUN_0057b420
    python vp2.py cmp cases.tsv game.tsv d2d_blow.tsv d2d_calc.tsv

My own model, written from the skill table text and my reading of the disassembly (not from d2d):
  C: calc slots by the Skills.txt formulas (lnXY = ParamX + (lvl-1) * ParamY), to-hit bonus = ToHit + LevToHit * (lvl-1).
  S: FUN_0057b420 with edx = 0 and the min / max passed in: both ends << 8, plus stat 111 << 8, min under 0x100 -> 0x100,
     max at or under min -> min + 0x100, p = ED + stat 25 + Strength (stat 0), floored at -90 once, min' = min + trunc(min *
     (stat 18 + p) / 100), max' likewise with stat 17; the result lies in [min', max'] (the draw is the unit's seed; I
     check the bounds, not the draw). The d2d side is the REAL player_blow in pal2_test (not my code).
The game side hooks only the stat / item getters (the shared inputs), as in pal2.py (same Oracle).
"""
import random
import sys

M32 = 0xFFFFFFFF
SAC, SMITE, ZEAL, RED, FAN = 96, 97, 106, 124, 122
OTHERS = (96, 97, 106, 122, 124, 98, 99, 113, 117, 120)


def s32(v):
    v &= M32
    return v - (1 << 32) if v >> 31 else v


def tdiv(a, b):
    q = abs(a) // abs(b)
    return q if (a >= 0) == (b > 0) else -q


def gen(n, seed):
    rng = random.Random(seed * 7919 + 13)
    out, i = [], 0
    for _ in range(n):                                   # C: random skill, level, synergies
        s = rng.choice((SAC, SMITE, ZEAL))
        lvl = rng.choice([rng.randint(1, 30), rng.randint(1, 80)])
        pairs = [(rng.choice(OTHERS), rng.randint(0, 35)) for _ in range(rng.randint(0, 6))]
        out.append(f"C {i} {s} {lvl} {rng.randint(1, 99)} {len(pairs)} " + " ".join(f"{a} {b}" for a, b in pairs)); i += 1
    for case in range(n):                                # S: Smite damage on a shield row; Holy Shield 0 here (own scope)
        bm = rng.randrange(1, 50)
        bM = bm + rng.randrange(1, 60) if case % 10 else bm
        ed = rng.choice((15, 30, 45, 60, 150, 300, 0))
        n111 = rng.choice((0, 0, 0, 1, 3, 6)); dp = rng.choice((0, 0, 0, 10, 50, 120))   # no negative percentages (D3 / D4 off)
        st = rng.randrange(0, 250); off = rng.choice((0, 0, 20, 60))
        out.append(f"S {case} {rng.getrandbits(32)} {rng.getrandbits(32)} {bm} {bM} 0 0 {ed} {n111} {dp} {st} {rng.randrange(5, 150)} 0 0 {off}")
    return out


def model_c(f):
    lvl, lv = f["lvl"], dict(f["pairs"])
    lv[f["skill"]] = lvl
    s = f["skill"]
    if s == SAC:
        return (180 + 15 * (lvl - 1) + lv.get(RED, 0) * 15 + lv.get(FAN, 0) * 5, 8, 0, 0, 20 + 7 * (lvl - 1))
    if s == SMITE:
        return (15 + 15 * (lvl - 1), min(250, 15 + 5 * (lvl - 1)), 0, 0, 0)
    return (min(2 + lvl - 1, 5), (0 if lvl < 5 else (lvl - 4) * 6) + lv.get(SAC, 0) * 12, 0, 0, 10 + 10 * (lvl - 1))


def model_s(f):
    mn = (f["bm"] + f["n111"]) << 8
    mx = (f["bM"] + f["n111"]) << 8
    mn = max(mn, 0x100)
    if mx <= mn: mx = mn + 0x100
    # the shield is passed as the item, so the Strength / Dex term is the item's StrBonus / DexBonus percent of them (0 here);
    # my first model added Strength itself (the barehanded path): wrong, 4,463 of 8,000 outside the bounds (see the doc)
    p = max(f["ed"] + f["dp"] + (f["str"] * f["sb"] + f["dex"] * f["db"]) // 100, -90)
    return mn + tdiv(mn * (f["off"] + p), 100), mx + tdiv(mx * (f["off"] + p), 100)


def parse(line):
    p = line.split()
    tag, i = p[0], int(p[1])
    v = list(map(int, p[2:]))
    if tag == "C":
        return tag, i, dict(skill=v[0], lvl=v[1], clvl=v[2], pairs=[(v[4 + 2 * k], v[5 + 2 * k]) for k in range(v[3])])
    return tag, i, dict(zip(("d0", "d1", "bm", "bM", "hl", "hh", "ed", "n111", "dp", "str", "dex", "sb", "db", "off"), v))


def game(path):
    import skill_damage
    import emu
    o = skill_damage.Oracle()
    cases = [parse(l) for l in open(path)]
    out = {}
    e = o.e
    for tag, i, f in cases:
        if tag != "C": continue
        o.levels = dict(f["pairs"]); o.levels[f["skill"]] = f["lvl"]
        o.stats = {0xc: f["clvl"]}
        r = o.skills + f["skill"] * skill_damage.RECORD
        vals = []
        for off in (0x138, 0x13c, 0x140, 0x144):
            ptr = e.r32(r + off)
            vals.append(s32(e.call(0x646ca0, o.unit, ptr, f["skill"], f["lvl"])) if ptr else 0)
        vals.append(s32(e.call(0x6449f0, o.unit, f["skill"], f["lvl"])))
        out[(tag, i)] = f"C {i} " + " ".join(map(str, vals))
    e = emu.Emu()
    stats, st = {}, {}
    e.hook(0x625480, lambda e: stats.get((e.arg(0), e.arg(1)), 0), 3)
    e.hook(0x625500, lambda e: stats.get((e.arg(0), e.arg(1)), 0), 3)
    for addr, n in ((0x639df0, 2), (0x554010, 3), (0x5347b0, 0), (0x57d810, 0), (0x451f80, 0)):
        e.hook(addr, lambda e: 0, n)
    e.hook(0x645830, lambda e: 0, 4)
    e.hook(0x63d340, lambda e: 1, 2)
    e.hook(0x629860, lambda e: 0, 1)
    e.hook(0x6298a0, lambda e: 0, 1)
    for tag, i, f in cases:
        if tag != "S": continue
        att = e.alloc(0x100); e.w32(att, 0)
        stats.clear()
        e.w32(att + 0x20, f["d0"]); e.w32(att + 0x24, f["d1"])
        stats.update({(att, 0x6f): f["n111"], (att, 0x19): f["dp"], (att, 0): f["str"], (att, 2): f["dex"], (att, 0x11): f["off"], (att, 0x12): f["off"]})
        got = s32(e.call(0x57b420, 0x7100000, f["bm"] << 8, f["bM"] << 8, f["ed"] & M32, 0, 0x80, ecx=att, edx=0))
        out[(tag, i)] = f"S {i} {got}"
    for tag, i, f in cases:
        print(out[(tag, i)])


def cmp(cpath, gpath, bpath, kpath):
    cases = [parse(l) for l in open(cpath)]
    g = [l.split() for l in open(gpath)]
    b = {(l.split()[0], int(l.split()[1])): l.split() for l in open(bpath)}
    k = {(l.split()[0], int(l.split()[1])): l.split() for l in open(kpath) if l.startswith("C ")}
    r = dict(C=[0, 0, 0, {}], S=[0, 0, 0, 0, 0.0])   # C: n, game!=model, game!=d2d ; S: n, game outside model, d2d whole differs, game==d2d exactly, sum diff
    for (tag, i, f), gl in zip(cases, g):
        if tag == "C":
            c = r["C"]; c[0] += 1
            gv = tuple(map(int, gl[2:]))
            if gv != model_c(f): c[1] += 1
            if gl[2:] != k[(tag, i)][2:]:
                c[2] += 1; c[3][f["skill"]] = c[3].get(f["skill"], 0) + 1
        else:
            c = r["S"]; c[0] += 1
            got, phys = int(gl[2]), int(b[(tag, i)][3])
            lo, hi = model_s(f)
            if not lo <= got <= hi:
                c[1] += 1
                if c[1] <= 40 and (c[1] % 5 == 0 or f["n111"] == 0): print("outside", got, (lo, hi), f)
            if (got >> 8) != phys: c[2] += 1
            if got == phys << 8: c[3] += 1
            c[4] += got / 256 - phys
    print("C n, game!=my model, game!=d2d, by skill:", r["C"])
    s = r["S"]
    print(f"S n={s[0]} game outside my bounds={s[1]} d2d whole points differ={s[2]} exact-256ths equal={s[3]} mean(game-d2d)={s[4] / max(s[0], 1):.3f}")
    for name, pred in (("n111>0", lambda f: f["n111"] > 0), ("n111=0 min<max", lambda f: f["n111"] == 0 and f["bm"] < f["bM"]),
                       ("n111=0 min=max", lambda f: f["n111"] == 0 and f["bm"] == f["bM"])):
        n = d = 0
        for (tag, i, f), gl in zip(cases, g):
            if tag == "S" and pred(f):
                n += 1; d += (int(gl[2]) >> 8) != int(b[(tag, i)][3])
        print(f"S {name}: {d} of {n} differ in whole points")


if __name__ == "__main__":
    a = sys.argv[1:]
    if a[0] == "gen": print("\n".join(gen(int(a[1]), int(a[2]))))
    elif a[0] == "game": game(a[1])
    elif a[0] == "cmp": cmp(*a[1:5])
