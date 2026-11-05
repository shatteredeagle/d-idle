"""covmonskills2: which srvdofunc ids of the Act 2 / Act 3 / boss monster skills (cmk_roles.py's list) d2d's monster executor names.
    python3 -I tools/emu/cmk_dod2d.py <enum.txt> <roles.txt>
enum.txt = lines 262.. of d2d's components/rules/skills.hpp (the ServerDoFunction enum), roles.txt = cmk_roles.py output.
HANDLED = the `ServerDoFunction::k...` names found in components/game/ai.cpp (grep -o over the file, 2026-11-02 read, V read); a do id whose enum
name is absent from HANDLED may still be coded by skill name (first pass covmonskills Q5 found 88, 92, 93, 97 that way): U."""
import re
import sys

HANDLED = """MonFrenzy Charge Jab Smite Hydra BaalTentacle BaalColdMissiles ImpFireMissile ImpTeleport MonTeleport Bestow MonPoisonNova
MissileAtTarget ImpInferno MonsterBreath DiabLight RegurgitatorEat SelfState DiabWall DiabPrison GuidedMissile DoomKnightMissile FingerMageSpider
DiabCold MinionSpawner CorpseExplosion MonCurseCast ScatteredBolts MissileWall Nova Curse SiegeBeastStomp WarCry Impregnate OverseerWhip CryHelp
MaggotDown Hatch MaggotLay DeathMaul LeapAttack TransMissile DiabRun""".split()
num = {}
for line in open(sys.argv[1]):
    m = re.match(r"\s+k(\w+)\s*=\s*(\d+)", line)
    if m:
        num[int(m.group(2))] = m.group(1)
handled_ids = {k for k, v in num.items() if v in HANDLED}
skills = {}
for line in open(sys.argv[2]):
    t = line.split("\t")
    if len(t) > 4 and t[1].startswith("id ") and t[2].startswith("do "):
        do = t[2][3:].strip()
        skills.setdefault(do or "-", []).append((t[0], t[1][3:], t[3], t[4]))
tot = {"named": 0, "unnamed": 0, "blank": 0}
for do, rows in sorted(skills.items(), key=lambda x: (x[0] == "-", int(x[0]) if x[0] != "-" else 0)):
    if do == "-":
        state = "no srvdofunc (start function / missile only)"
        tot["blank"] += len(rows)
    elif int(do) in handled_ids:
        state = "named in ai.cpp (%s)" % num[int(do)]
        tot["named"] += len(rows)
    elif int(do) in num:
        state = "enum name %s, not in ai.cpp's executor" % num[int(do)]
        tot["unnamed"] += len(rows)
    else:
        state = "no enum name"
        tot["unnamed"] += len(rows)
    print("do %-4s %-45s %s" % (do, state, "; ".join("%s (%s, %s)" % (r[0], r[1], r[2]) for r in rows)))
print(tot)
