#!/usr/bin/env python3
"""The player's plain Attack: game.exe 1.14d vs the REAL d2d build (d2lod-bot, card skmelee2).

Runs next to upstream's harness (psi29a/d2decomped tools/emu: emu.py) with sk_melee.py
(card skmelee) beside it; the d2d side is tools/emu/sk_melee2_test.cpp built on the upstream
tree as tests/game/test_melee.cpp.

    python sk_melee2.py gen 100000 7 [UREPS] > c7.tsv   # the cases (the C++ side only reads them); UREPS cases per undead cell (default 25)
    python sk_melee2.py game c7.tsv > g7.tsv           # game.exe's own code under unicorn
    <build>/tests/test_melee --melee c7.tsv > d7.tsv   # d2d's make_fighter + player_blow
    python sk_melee2.py cmp c7.tsv g7.tsv d7.tsv

Parts (tags):
- H: to-hit, the case grid of sk_melee.py's `hit` part (240 boundary + N random, the same
  draws from random.Random(seed & 0xFFFFFFFF)): FUN_0057d9b0 vs player_blow's hit.
- M: damage, sk_melee.py's `dmg` part (N cases, the same draws, continuing the same rng):
  FUN_0057b420 (ED = the skill ED, flat 0, SrcDam 0x80) vs player_blow's Blow::phys.
- U: the undead grid (undead x {barehanded, non-blunt, blunt} x stat 122 x record ED, 25 random
  builds each): FUN_0057b660 then FUN_0057b420 on the record's ED vs player_blow (no term).
- F: the -90 floor grid (stat 25 under -90 with a skill ED).
- O: off-weapon enhanced damage (stats 17 / 18 at the player from other gear, model A': the
  weapon's own ED excluded there; trace Q3) vs make_fighter's `sum` 17 / 18.
Seeds: the case's s0 / s1 (sk_melee.py's draws). To-hit: s0 / s1 on both sides. Damage: d2d's
Rng starts at s0 / s1 and draws its to-hit roll first; game.exe's unit + 0x20 gets the state one
step later (g0 / g1), so both damage rolls are the same draw. A d2d miss (that to-hit roll
% 100 >= 95, the 95 % cap) carries no damage and is excluded.
"""
import random
import sys

A = 0x6AC690C5
M32 = 0xFFFFFFFF


def step(s0, s1):
    """One LCG step (FUN_0045c3e0's draw, d2d Rng::next): the state after d2d's to-hit roll."""
    x = s0 * A + s1
    return x & M32, x >> 32


def tdiv(a, b):
    q = abs(a) // abs(b)
    return q if (a >= 0) == (b > 0) else -q


def muldiv(value, times, over):
    return tdiv(value * times, over) if over else 0


def dmg_fields(rng, weapon=None):
    """sk_melee.py dmg_part's draws, in its order."""
    seed = [rng.getrandbits(32), rng.getrandbits(32)]
    if weapon is None: weapon = 1 if rng.random() < 0.85 else 0
    bm = rng.randrange(1, 40); bM = bm + rng.randrange(1, 60)
    ed = rng.choice((0, 0, 0, 0, 30, 50, 100, 200, 300))
    fmin = rng.choice((0, 0, 0, 1, 2, 5)); fmax = rng.choice((0, 0, 0, 1, 3, 10))
    n111 = rng.choice((0, 0, 0, 0, 1, 2))
    dp = rng.choice((0, 0, 0, 0, 20, 100, -25))
    st = rng.randrange(10, 200); dx = rng.randrange(10, 200)
    sb, db = rng.choice(((100, 0), (100, 0), (110, 0), (0, 100), (75, 75), (0, 0)))
    mastery = rng.choice((0, 0, 0, 0, 28, 100)) if weapon else 0
    eskill = rng.choice((0, 0, 0, 0, 0, 50, 150))
    return dict(s0=seed[0], s1=seed[1], weapon=weapon, bm=bm, bM=bM, ed=ed, fmin=fmin, fmax=fmax, n111=n111,
                dp=dp, str=st, dex=dx, sb=sb, db=db, mastery=mastery, eskill=eskill, off=0)


DKEYS = ("weapon", "bm", "bM", "ed", "fmin", "fmax", "n111", "dp", "str", "dex", "sb", "db", "mastery", "eskill", "off")


def dline(tag, i, f, extra=()):
    g0, g1 = step(f["s0"], f["s1"])
    return " ".join(str(v) for v in (tag, i, f["s0"], f["s1"], *(f[k] for k in DKEYS), g0, g1, *extra))


def gen(cases, seed, ureps=25):
    rng = random.Random(seed & M32)
    out = []
    edges = [(ar, pct, d) for ar in (-40, -1, 0, 1, 7, 99, 100, 101) for pct in (-90, -33, -1, 0, 1, 33) for d in (-10, 0, 1, 50, 100)]
    for case in range(len(edges) + cases):              # sk_melee.py hit_part, draw for draw
        s0, s1 = rng.getrandbits(32), rng.getrandbits(32)
        if case < len(edges):
            ar, pct, defense = edges[case]; mastery, bonus, mdef = 0, 0, 0
        else:
            ar = rng.randrange(-40, 2500); defense = rng.randrange(-30, 1500)
            pct = rng.choice((0, 0, 0, 10, 25, 50, 100, 200, -10, -25))
            mastery = rng.choice((0, 0, 0, 28, 61)); bonus = rng.choice((0, 0, 0, 0, 30, 100)); mdef = rng.choice((0, 0, 0, 10))
        alvl, dlvl = rng.randrange(1, 99), rng.randrange(1, 99)
        weapon = rng.choice((0, 1))
        out.append(f"H {case} {s0} {s1} {ar} {pct} {mastery} {bonus} {defense} {mdef} {alvl} {dlvl} {weapon}")
    for case in range(cases):                           # sk_melee.py dmg_part, draw for draw
        out.append(dline("M", case, dmg_fields(rng)))
    i = 0
    for undead in (0, 1):
        for weapon, blunt in ((0, 0), (1, 0), (1, 1)):
            for s122 in (-60, -1, 0, 1, 50):
                for ed0 in (0, 35):
                    for _ in range(ureps):
                        f = dmg_fields(rng, weapon); f["eskill"] = ed0
                        out.append(dline("U", i, f, (undead, blunt, s122))); i += 1
    i = 0
    for dp in (-150, -120, -91, -90, -50):
        for eskill in (0, 50, 150, 300):
            for _ in range(20):
                f = dmg_fields(rng, 1); f["dp"] = dp; f["eskill"] = eskill
                out.append(dline("F", i, f)); i += 1
    i = 0
    for off in (0, 15, 40, 100, 300):
        for _ in range(200):
            f = dmg_fields(rng); f["off"] = off
            out.append(dline("O", i, f)); i += 1
    return out


def parse(line):
    p = line.split()
    tag, i = p[0], int(p[1])
    if tag == "H":
        return tag, i, dict(zip(("s0", "s1", "ar", "pct", "mastery", "bonus", "def", "mdef", "alvl", "dlvl", "weapon"), map(int, p[2:13])))
    f = dict(zip(("d0", "d1") + DKEYS + ("g0", "g1"), map(int, p[2:21])))
    if tag == "U": f.update(zip(("undead", "blunt", "s122"), map(int, p[21:24])))
    return tag, i, f


def hit_chance(attack, defense, alvl, dlvl):
    if defense < 0: attack, defense = attack - defense, 0
    if attack < 0: defense, attack = defense - attack, 0
    chance = tdiv(attack * 100, attack + defense) if attack + defense else 100
    chance = tdiv(chance * 2 * alvl, max(alvl + dlvl, 1))
    return max(5, min(95, chance))


def game(path, exe=None):
    import emu
    e = emu.Emu(exe) if exe else emu.Emu()
    stats, st = {}, {}
    e.hook(0x625480, lambda e: stats.get((e.arg(0), e.arg(1)), 0), 3)
    e.hook(0x625500, lambda e: stats.get((e.arg(0), e.arg(1)), 0), 3)
    e.hook(0x6223f0, lambda e: stats.get((e.arg(0), "def"), 0), 1)
    for addr, n in ((0x639df0, 2), (0x554010, 3), (0x5347b0, 0), (0x57d810, 0), (0x451f80, 0)):
        e.hook(addr, lambda e: 0, n)
    e.hook(0x622560, lambda e: st["ar"], 1)
    e.hook(0x57d8a0, lambda e: 0, 2)
    e.hook(0x645830, lambda e: st["mastery"], 4)
    e.hook(0x535bc0, lambda e: st["weapon"], 0)
    e.hook(0x63d340, lambda e: 1, 2)
    e.hook(0x629860, lambda e: st["sb"], 1)
    e.hook(0x6298a0, lambda e: st["db"], 1)
    e.hook(0x63e990, lambda e: st["undead"], 1)
    e.hook(0x629bb0, lambda e: 1 if (st["blunt"] and e.arg(1) == 57) else 0, 2)
    for line in open(path):
        tag, i, f = parse(line)
        att = e.alloc(0x100)
        e.w32(att, 0)
        e.w32(att + 0x20, f["s0" if tag == "H" else "g0"]); e.w32(att + 0x24, f["s1" if tag == "H" else "g1"])
        stats.clear()
        if tag == "H":
            dfd = e.alloc(0x100); e.w32(dfd, 1)
            st.update(ar=f["ar"], mastery=f["mastery"], weapon=0x7000000 if f["weapon"] else 0)
            stats.update({(att, 0xc): f["alvl"], (att, 0x77): f["pct"], (dfd, 0xc): f["dlvl"], (dfd, "def"): f["def"], (dfd, 0x21): f["mdef"]})
            print(f"H {i} {e.call(0x57d9b0, f['bonus'], 0, ecx=att, edx=dfd)}")
            continue
        w = f["weapon"]
        s21 = (tdiv(f["bm"] * (100 + f["ed"]), 100) if w else 0) + f["fmin"]
        s22 = (tdiv(f["bM"] * (100 + f["ed"]), 100) if w else 0) + f["fmax"]
        st.update(weapon=0x7000000 if w else 0, sb=f["sb"], db=f["db"], mastery=f["mastery"],
                  undead=f.get("undead", 0), blunt=f.get("blunt", 0))
        stats.update({(att, 0x15): s21, (att, 0x16): s22, (att, 0x6f): f["n111"], (att, 0x19): f["dp"],
                      (att, 0): f["str"], (att, 2): f["dex"], (att, 0x11): f["off"], (att, 0x12): f["off"],
                      (att, 0x7a): f.get("s122", 0)})
        ed = f["eskill"]
        if tag == "U":
            tgt, rec = e.alloc(0x100), e.alloc(0x100)
            e.w32(rec + 0xc, ed)
            e.call(0x57b660, regs={"eax": tgt, "edi": att, "ebx": rec})
            ed = e.r32(rec + 0xc)
            ed = ed - (1 << 32) if ed >= 1 << 31 else ed
        got = e.call(0x57b420, 0, 0, 0, ed & M32, 0, 0x80, ecx=att, edx=1) & M32
        got = got - (1 << 32) if got >= 1 << 31 else got
        print(f"{tag} {i} {got} {ed}")


def cmp(cpath, gpath, dpath):
    cases = [parse(l) for l in open(cpath)]
    g = [l.split() for l in open(gpath)]
    d = [l.split() for l in open(dpath)]
    assert len(cases) == len(g) == len(d), (len(cases), len(g), len(d))
    rep = {}
    for (tag, i, f), gl, dl in zip(cases, g, d):
        assert gl[0] == dl[0] == tag and int(gl[1]) == int(dl[1]) == i
        r = rep.setdefault(tag, {"n": 0, "miss": 0, "whole": 0, "fine": 0, "sum": 0, "cat": {}})
        r["n"] += 1
        if tag == "H":
            if int(gl[2]) != int(dl[2]):
                r["whole"] += 1
                t = f["pct"] + (f["mastery"] if f["weapon"] else 0) + f["bonus"]
                k = "ar'<=0" if f["ar"] + muldiv(f["ar"], t, 100) <= 0 else "pct<0" if t < 0 else "other"
                r["cat"][k] = r["cat"].get(k, 0) + 1
            continue
        if int(dl[2]) == 0:
            r["miss"] += 1
            continue
        got = int(gl[2])
        mine = int(dl[4]) * 256 + int(dl[5]) if len(dl) > 5 else int(dl[3]) * 256     # d2d's physical in 256ths: damage x 256 + frac
        if tag == "M": k = "weapon" if f["weapon"] else "barehanded"
        elif tag == "U": k = f"undead{f['undead']} {'blunt' if f['blunt'] else 'weapon' if f['weapon'] else 'bare'} 122{'+' if f['s122'] > 0 else '0' if f['s122'] == 0 else '-'}"
        elif tag == "F": k = f"dp{f['dp']} e{f['eskill']}"
        else: k = f"off{f['off']} {'weapon' if f['weapon'] else 'bare'}"
        c = r["cat"].setdefault(k, [0, 0, 0, 0])
        c[0] += 1; r["sum"] += got - mine; c[3] += got - mine
        if got >> 8 != mine >> 8: r["whole"] += 1; c[1] += 1
        if got != mine: r["fine"] += 1; c[2] += 1
    for tag, r in rep.items():
        if tag == "H":
            print(f"H to-hit: {r['n']} cases, {r['whole']} differ {r['cat']}")
            continue
        n = r["n"] - r["miss"]
        print(f"{tag}: {r['n']} cases, {r['miss']} d2d misses excluded, {n} compared: {r['whole']} differ in whole points, "
              f"{r['fine']} in 256ths; mean game - d2d {r['sum'] / max(n, 1) / 256:.4f} points")
        for k, c in sorted(r["cat"].items()):
            print(f"  {k}: {c[0]} compared, whole {c[1]}, 256ths {c[2]}, mean {c[3] / max(c[0], 1) / 256:+.4f}")


if __name__ == "__main__":
    argv, exe = sys.argv[1:], None
    if "--exe" in argv:
        k = argv.index("--exe"); exe = argv[k + 1]; del argv[k:k + 2]
    what = argv[0]
    if what == "gen": print("\n".join(gen(int(argv[1]), int(argv[2]), int(argv[3]) if len(argv) > 3 else 25)))
    elif what == "game": game(argv[1], exe)
    elif what == "cmp": cmp(argv[1], argv[2], argv[3])
