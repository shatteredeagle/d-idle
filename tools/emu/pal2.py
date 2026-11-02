#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-3.0-or-later
"""pal2 (card palskill): Smite, Zeal and Sacrifice: game.exe 1.14d's own code vs d2d's real rules.

    python pal2.py gen N SEED > cases.tsv          # the cases (the C++ side only reads them)
    python pal2.py game cases.tsv > game.tsv       # game.exe under unicorn
    pal2_test --blow cases.tsv > d2d_blow.tsv      # d2d's make_fighter + player_blow (tools/emu/pal2_test.cpp)
    pal2_test --calc cases.tsv > d2d_calc.tsv      # d2d's eval_calc / skill_tohit on the loaded patched table
    python pal2.py cmp cases.tsv game.tsv d2d_blow.tsv d2d_calc.tsv

Case lines (one tag each):
- C id skill lvl clvl n (skill level)*n : the calc slots of a skill row (Sacrifice 96, Smite 97, Zeal 106).
    game: `C id c1 c2 c3 c4 tohit` = FUN_00646ca0(unit, calc, skill, lvl) for the four calc slots (row +0x138 ..
    +0x144) and FUN_006449f0 (the to-hit bonus every melee skill passes to FUN_0057ec10).
- T id s0 s1 ar pct alvl dlvl armed def mdef : the roll inside Smite's to-hit. game: FUN_0057d9b0 with bonus 0 (what
    FUN_005ce9f0 passes to FUN_0057ec10 at 0x5ceba8: push 0, push 0). The roll misses often, but 0x5cebb0 ORs the hit
    bit back in on the player path (V read): NOT a deviation (a dead end of this card, kept as the record of it).
- S id d0 d1 bm bM hl hh ed n111 dp str dex sb db off : Smite's damage. game: FUN_0057b420 with edx = 0, the shield
    as the item, min / max the shield's (<< 8) plus Holy Shield's (hl / hh, 256ths), ED = calc1, SrcDam 0x80 (the
    exact call at 0x5ceb31). d2d: player_blow with swing.smite, the shield's item_base row; its Rng is seeded with
    the state game.exe's unit holds (so both draw the damage roll first).
Hooked (not the claim): stat getters, item bonus getters, FUN_00643810 / FUN_006442a0 for the calc machine (as
skill_damage.py), the player-side lookups of sk_melee2.py.
"""
import random
import sys

M32 = 0xFFFFFFFF
SKILLS = (96, 97, 106)
OTHERS = (96, 97, 98, 99, 106, 107, 108, 111, 117, 120, 122, 123, 124, 125, 104, 113, 115)


def s32(v):
    v &= M32
    return v - (1 << 32) if v >> 31 else v


def gen(n, seed):
    rng = random.Random(seed & M32)
    out = []
    i = 0
    for s in SKILLS:                                    # every level 1..40 alone, with synergies at 20, at random
        for lvl in range(1, 41):
            for clvl, pairs in ((30, []), (99, [(t, 20) for t in OTHERS]), (50, [(t, rng.randint(0, 30)) for t in OTHERS])):
                out.append(f"C {i} {s} {lvl} {clvl} {len(pairs)} " + " ".join(f"{a} {b}" for a, b in pairs)); i += 1
    for _ in range(n):
        s = rng.choice(SKILLS)
        pairs = [(rng.choice(OTHERS), rng.randint(0, 40)) for _ in range(rng.randint(0, 8))]
        lvl = rng.choice([rng.randint(1, 20), rng.randint(1, 60)])
        out.append(f"C {i} {s} {lvl} {rng.randint(1, 99)} {len(pairs)} " + " ".join(f"{a} {b}" for a, b in pairs)); i += 1
    for case in range(n):                               # Smite to-hit: sk_melee2's H draws, bonus 0, no mastery
        s0, s1 = rng.getrandbits(32), rng.getrandbits(32)
        ar = rng.randrange(-40, 2500); defense = rng.randrange(-30, 1500)
        pct = rng.choice((0, 0, 0, 10, 25, 50, 100, 200, -10, -25))
        mdef = rng.choice((0, 0, 0, 10))
        out.append(f"T {case} {s0} {s1} {ar} {pct} {rng.randrange(1, 99)} {rng.randrange(1, 99)} {rng.choice((0, 1))} {defense} {mdef}")
    for case in range(n + n // 40):                     # Smite damage; the last n / 40 cases have min = max (Zakarum Shield 46 / 46)
        d0, d1 = rng.getrandbits(32), rng.getrandbits(32)
        bm = rng.randrange(1, 30); bM = bm + rng.randrange(1, 40) if case < n else bm   # game.exe forces max >= min + 1 point (0x57b4ef)
        hl, hh = rng.choice(((0, 0), (0, 0), (0, 0), (256 * rng.randrange(1, 40), 256 * rng.randrange(40, 120))))
        if hh < hl: hh = hl
        ed = rng.choice((15, 30, 45, 90, 150, 285, 300, 0, 0, -20))
        n111 = rng.choice((0, 0, 0, 0, 1, 2)); dp = rng.choice((0, 0, 0, 0, 20, 100, -25, -95))
        st = rng.randrange(10, 200); dx = rng.randrange(10, 200)
        sb, db = rng.choice(((100, 0), (100, 0), (110, 0), (0, 100), (75, 75), (0, 0)))
        off = rng.choice((0, 0, 0, 15, 40, 100))
        out.append(f"S {case} {d0} {d1} {bm} {bM} {hl} {hh} {ed} {n111} {dp} {st} {dx} {sb} {db} {off}")
    return out


def parse(line):
    p = line.split()
    tag, i = p[0], int(p[1])
    v = list(map(int, p[2:]))
    if tag == "C":
        n = v[3]
        return tag, i, dict(skill=v[0], lvl=v[1], clvl=v[2], pairs=[(v[4 + 2 * k], v[5 + 2 * k]) for k in range(n)])
    if tag == "T":
        return tag, i, dict(zip(("s0", "s1", "ar", "pct", "alvl", "dlvl", "armed", "def", "mdef"), v))
    return tag, i, dict(zip(("d0", "d1", "bm", "bM", "hl", "hh", "ed", "n111", "dp", "str", "dex", "sb", "db", "off"), v))


def step(s0, s1):
    x = s0 * 0x6AC690C5 + s1
    return x & M32, x >> 32


def game(path):
    import skill_damage
    import sk_melee2
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
    import emu
    e = emu.Emu()
    stats, st = {}, {}
    e.hook(0x625480, lambda e: stats.get((e.arg(0), e.arg(1)), 0), 3)
    e.hook(0x625500, lambda e: stats.get((e.arg(0), e.arg(1)), 0), 3)
    e.hook(0x6223f0, lambda e: stats.get((e.arg(0), "def"), 0), 1)
    for addr, n in ((0x639df0, 2), (0x554010, 3), (0x5347b0, 0), (0x57d810, 0), (0x451f80, 0)):
        e.hook(addr, lambda e: 0, n)
    e.hook(0x622560, lambda e: st["ar"], 1)
    e.hook(0x57d8a0, lambda e: 0, 2)
    e.hook(0x645830, lambda e: 0, 4)
    e.hook(0x535bc0, lambda e: st["weapon"], 0)
    e.hook(0x63d340, lambda e: 1, 2)
    e.hook(0x629860, lambda e: st["sb"], 1)
    e.hook(0x6298a0, lambda e: st["db"], 1)
    for tag, i, f in cases:
        if tag == "C": continue
        att = e.alloc(0x100)
        e.w32(att, 0)
        stats.clear()
        if tag == "T":
            e.w32(att + 0x20, f["s0"]); e.w32(att + 0x24, f["s1"])
            dfd = e.alloc(0x100); e.w32(dfd, 1)
            st.update(ar=f["ar"], weapon=0x7000000 if f["armed"] else 0)
            stats.update({(att, 0xc): f["alvl"], (att, 0x77): f["pct"], (dfd, 0xc): f["dlvl"], (dfd, "def"): f["def"], (dfd, 0x21): f["mdef"]})
            out[(tag, i)] = f"T {i} {e.call(0x57d9b0, 0, 0, ecx=att, edx=dfd)}"
            continue
        # two seedings: d2d today makes no to-hit draw for Smite, so its damage roll is its FIRST draw (game state d0 / d1);
        # with the fix it draws the to-hit roll first, so the damage roll is the second (game state one step later).
        gots = []
        for g0, g1 in ((f["d0"], f["d1"]), step(f["d0"], f["d1"])):
            e.w32(att + 0x20, g0); e.w32(att + 0x24, g1)
            st.update(weapon=0, sb=f["sb"], db=f["db"])
            stats.update({(att, 0x6f): f["n111"], (att, 0x19): f["dp"], (att, 0): f["str"], (att, 2): f["dex"], (att, 0x11): f["off"], (att, 0x12): f["off"]})
            gots.append(s32(e.call(0x57b420, 0x7100000, (f["bm"] << 8) + f["hl"], (f["bM"] << 8) + f["hh"], f["ed"] & M32, 0, 0x80, ecx=att, edx=0)))
        out[(tag, i)] = f"S {i} {gots[0]} {gots[1]}"
    for tag, i, f in cases:
        print(out[(tag, i)])


def cmp(cpath, gpath, bpath, kpath, after=False):
    cases = [parse(l) for l in open(cpath)]
    g = [l.split() for l in open(gpath)]
    b = {(l.split()[0], int(l.split()[1])): l.split() for l in open(bpath)}
    k = {(l.split()[0], int(l.split()[1])): l.split() for l in open(kpath) if l.startswith("C ")}   # d2d's loader logs to stdout
    assert len(cases) == len(g)
    rep = {}
    for (tag, i, f), gl in zip(cases, g):
        assert gl[0] == tag and int(gl[1]) == i
        r = rep.setdefault(tag, {"n": 0, "diff": 0, "fine": 0, "plain": 0, "cat": {}})
        r["n"] += 1
        if tag == "C":
            dl = k[(tag, i)]
            if gl[2:] != dl[2:]:
                r["diff"] += 1
                r["cat"][f["skill"]] = r["cat"].get(f["skill"], 0) + 1
        elif tag == "T":
            dl = b[(tag, i)]
            if gl[2] != dl[2]: r["diff"] += 1                                 # game vs d2d with swing.smite
            if gl[2] != dl[3]: r["plain"] += 1                                # game vs d2d's plain path (the fix)
            if gl[2] == "0": r["cat"]["game miss"] = r["cat"].get("game miss", 0) + 1
        else:
            dl = b[(tag, i)]
            if after and dl[2] == "0":
                r["miss"] = r.get("miss", 0) + 1                  # the fixed d2d misses (95 % cap): no damage to compare
                continue
            got, phys = int(gl[2]), int(dl[3])      # gl[3]: the game's state one draw later (its forced-hit to-hit roll consumed); unused
            for key in (f"hs{int(f['hl'] > 0)}", f"n111_{int(f['n111'] > 0)}", f"dp{'<0' if f['dp'] < 0 else '>0' if f['dp'] > 0 else '0'}",
                        f"off{int(f['off'] > 0)}", f"ed{'<0' if f['ed'] < 0 else '0' if f['ed'] == 0 else '>0'}"):
                c = r["cat"].setdefault(key, [0, 0]); c[0] += 1
                if max(got >> 8, 1) != phys: c[1] += 1
            grp = "min=max" if f["bm"] == f["bM"] else "n111>0" if f["n111"] > 0 else "dp<-90" if f["dp"] < -90 else "rest"
            gc = r.setdefault("grp", {}).setdefault(grp, [0, 0, 0.0]); gc[0] += 1
            if max(got >> 8, 1) != phys: gc[1] += 1
            gc[2] += got / 256 - phys
            if f["dp"] >= 0 and f["ed"] >= 0 and f["bm"] != f["bM"]:                  # no negative percentage anywhere: no -90 floor, no toward-zero rounding
                key = "clean n111>0" if f["n111"] > 0 else "clean n111=0"
                cc = r["grp"].setdefault(key, [0, 0, 0.0]); cc[0] += 1
                if max(got >> 8, 1) != phys: cc[1] += 1
                cc[2] += got / 256 - phys
            if max(got >> 8, 1) != phys and grp == "rest" and len(r.setdefault("rest", [])) < 8: r["rest"].append((i, got, phys, f))
            if max(got >> 8, 1) != phys: r["diff"] += 1
            if got != phys << 8: r["fine"] += 1
    for tag, r in rep.items():
        r.pop("rest", None)
        if tag == "S": r.pop("cat", None)
        print(tag, r)


if __name__ == "__main__":
    a = sys.argv[1:]
    if a[0] == "gen": print("\n".join(gen(int(a[1]), int(a[2]))))
    elif a[0] == "game": game(a[1])
    elif a[0] == "cmp": cmp(*a[1:5], after=len(a) > 5 and a[5] == "after")
