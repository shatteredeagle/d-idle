# SPDX-License-Identifier: GPL-3.0-or-later
"""verifycovmissiles: own case generator for the element hit of the Sorceress / Paladin skills (harness: sk_missile_hit.py,
card SKMISS, independently verified before; only the skill table and the cases are mine).

    vcm_hit.py gen N SEED > cases.tsv            # structured grid + N random cases
    vcm_hit.py game cases.tsv > game.tsv 2> game.err

Skills: 41 Inferno, 46 Blaze, 47 Fire Ball, 51 Fire Wall (fire); 55 Glacial Spike, 60 Chilling Armor (cold); 57 Thunder Storm,
121 Fist of the Heavens (lightning). Partners: EVERY skill id 36..130 (94 of them) at random levels, not a window around the skill,
so a calc cell that reads any Sorceress or Paladin skill is exercised; Skills.txt calcs of these skills reference at most a few.
"""
import random
import sys

import sk_missile_hit as m

MASK = 0xFFFFFFFF
ETYPE = {"fire": 1, "ltng": 2, "cold": 4}      # game.exe's etype numbering
m.SKILLS = {41: "fire", 46: "fire", 47: "fire", 51: "fire", 55: "cold", 60: "cold", 57: "ltng", 121: "ltng"}
m.MASTERY = {"fire": 0, "ltng": 1, "cold": 2}
ALL = list(range(36, 131))


def gen(n, seed):
    rng = random.Random(seed * 7919 + 13)
    out = []

    def case(skill, lvl, clvl, mastery, partners, res, pierce, life):
        ms = [0, 0, 0, 0]
        ms[m.MASTERY[m.SKILLS[skill]]] = mastery
        pairs = [(t, partners(t)) for t in ALL if t != skill]
        out.append((skill, lvl, clvl, ms, res, pierce, life, (rng.getrandbits(32), rng.getrandbits(32)), pairs))

    for skill in m.SKILLS:
        for lvl in (1, 2, 3, 5, 8, 10, 13, 17, 20, 24, 30, 40):
            for psyn in (0, 1, 7, 20, 60):
                for mastery in (0, 31, 98, 233):
                    for res in (-100, -1, 0, 1, 40, 75, 99, 100, 120):
                        pierce = rng.choice((0, 0, 20, 100)) if res < 100 else 0
                        case(skill, lvl, rng.randint(1, 99), mastery, lambda t, p=psyn: p, res, pierce, rng.choice((1, 3, 10, 77, 500)))
    for _ in range(n):
        skill = rng.choice(list(m.SKILLS))
        res = rng.choice((rng.randint(-100, 99), rng.randint(100, 150), 0))
        pierce = 0 if res >= 100 or rng.random() < 0.5 else rng.randint(0, 100)
        levels = {t: rng.choice((0, 0, rng.randint(1, 20), rng.randint(21, 60))) for t in ALL}
        case(skill, rng.randint(1, 60), rng.randint(1, 99), rng.choice((0, 0, rng.randint(1, 500))), lambda t, L=levels: L[t], res, pierce,
             rng.randint(1, 600))
    for i, (s, lvl, clvl, ms, res, pierce, life, sd, pairs) in enumerate(out):
        print(i, s, lvl, clvl, *ms, res, pierce, life, *sd, len(pairs), *(v for p in pairs for v in p))


def check_etypes():
    import skill_damage
    o = skill_damage.Oracle()
    for s, el in m.SKILLS.items():
        et = o.row(s)["etype"]
        if et != ETYPE[el]:
            raise SystemExit(f"skill {s}: game.exe etype {et}, expected {el}")
    print("etype check: ok", file=sys.stderr)


if __name__ == "__main__":
    a = sys.argv[1:]
    if a and a[0] == "gen":
        gen(int(a[1]), int(a[2]))
    else:
        if a and a[0] == "game":
            check_etypes()
        m.main()
