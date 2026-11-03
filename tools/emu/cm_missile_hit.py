# SPDX-License-Identifier: GPL-3.0-or-later
"""covmissiles: the elemental missile hit (roll + apply) for the Sorceress and Paladin skills not yet run, by game.exe's own code.

sk_missile_hit.py (card SKMISS) with a new skill table; same cases format, same `game` output, same d2d side
(sk_missile_hit_test.cpp). Only the element pair of the missile (Skills.txt EMin..EMax with synergies and mastery) goes
through FUN_005a89a0 / FUN_0057c6c0; flight, collision and the hit function (pSrvHitFunc) are not part of this run.

    cm_missile_hit.py rows                    # etype / emin / emax of every Sor + Pal skill row, read by game.exe's own loader
    cm_missile_hit.py gen N SEED > cases.tsv
    cm_missile_hit.py game cases.tsv > game.tsv 2> game.err      (checks each skill's etype against the table below first)

Skills: 41 Inferno, 46 Blaze, 47 Fire Ball, 51 Fire Wall (fire); 55 Glacial Spike, 60 Chilling Armor (cold);
57 Thunder Storm, 121 Fist of the Heavens (lightning).  Synergy partners: every skill within 25 ids of the skill
(Skills.txt calc cells reference some of them; unused partners are harmless to the calc).
"""
import random
import sys

import sk_missile_hit as m

MASK = 0xFFFFFFFF
ETYPE = {"fire": 1, "ltng": 2, "cold": 4}      # game.exe's Skills.txt etype numbering (1 fire, 2 ltng, 3 magic, 4 cold)
m.SKILLS = {41: "fire", 46: "fire", 47: "fire", 51: "fire", 55: "cold", 60: "cold", 57: "ltng", 121: "ltng"}
PARTNERS = {s: tuple(t for t in range(max(s - 25, 0), s + 26) if t != s) for s in m.SKILLS}
m.SYNERGIES = PARTNERS
m.MASTERY = {"fire": 0, "ltng": 1, "cold": 2}


def gen(n, seed):
    rng = random.Random(seed & MASK)
    out = []

    def case(skill, lvl, clvl, mastery, pairs, res, pierce, life):
        ms = [0, 0, 0, 0]
        ms[m.MASTERY[m.SKILLS[skill]]] = mastery
        out.append((skill, lvl, clvl, ms, res, pierce, life, (rng.getrandbits(32), rng.getrandbits(32)), pairs))

    def level_pairs(skill, syn):
        return [(t, syn) for t in PARTNERS[skill]]

    # structured: every skill at levels 1..20, all partners at 0 / 1 / 5 / 10 / 20, masteries, the resistances that matter
    for skill in m.SKILLS:
        for lvl in range(1, 21):
            for syn in (0, 1, 5, 10, 20):
                for mastery in (0, 50, 98, 158, 278):
                    for res in (-100, -50, 0, 20, 25, 33, 50, 75, 99, 100, 150):
                        pierce = rng.choice((0, 0, 15, 60)) if res < 100 else 0
                        case(skill, lvl, rng.randint(1, 99), mastery, level_pairs(skill, syn), res, pierce, rng.choice((1, 2, 5, 20, 60, 300)))
    # random: independent random level per partner (finds which partner a calc cell uses)
    for _ in range(n):
        skill = rng.choice(list(m.SKILLS))
        res = rng.choice((rng.randint(-100, 99), rng.randint(100, 150), 0))
        pierce = 0 if res >= 100 or rng.random() < 0.6 else rng.randint(0, 100)
        pairs = [(t, rng.choice((0, 0, rng.randint(1, 20), rng.randint(21, 40)))) for t in PARTNERS[skill] if rng.random() < 0.5]
        case(skill, rng.randint(1, 20) if rng.random() < 0.8 else rng.randint(21, 40), rng.randint(1, 99),
             rng.choice((0, 0, rng.randint(1, 400))), pairs, res, pierce, rng.randint(1, 400))
    for i, (s, lvl, clvl, ms, res, pierce, life, sd, pairs) in enumerate(out):
        print(i, s, lvl, clvl, *ms, res, pierce, life, *sd, len(pairs), *(v for p in pairs for v in p))


def rows():
    import skill_damage
    o = skill_damage.Oracle()
    for s in list(range(36, 66)) + list(range(96, 126)):
        r = o.row(s)
        print(s, r["etype"], r["emin"], r["emax"], r["eminlev1"], r["emaxlev1"], r["hitshift"], r["elen"])


def check_etypes():
    import skill_damage
    o = skill_damage.Oracle()
    for s, el in m.SKILLS.items():
        et = o.row(s)["etype"]
        if et != ETYPE[el]:
            raise SystemExit(f"skill {s}: game.exe etype {et}, table says {el}")
    print(f"etype check: {len(m.SKILLS)} skills match the table", file=sys.stderr)


if __name__ == "__main__":
    a = sys.argv[1:]
    if a and a[0] == "gen":
        gen(int(a[1]), int(a[2]))
    elif a and a[0] == "rows":
        rows()
    else:
        if a and a[0] == "game":
            check_etypes()
        m.main()
