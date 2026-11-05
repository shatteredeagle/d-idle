# SPDX-License-Identifier: GPL-3.0-or-later
"""COVMONSKILLS: the srvdofunc / srvstfunc values the 147 monster and common rows use, against the names d2d's
skills.hpp gives (hand list below, copied from the enum lines read on the worker host), grouped by value.
    python3 cms_dofuncs.py rows.txt   (rows.txt = cms_rows.py output)"""
import re
import sys

# ServerDoFunction names in d2d's rules/skills.hpp (value: name)
DO = {1: "kAttack", 2: "kResolveHit", 3: "kThrow", 6: "kStateAroundCaster", 7: "kJab", 8: "kMissileFan", 9: "kFrenzy",
      10: "kGuidedMissile", 11: "kChargedStrike", 12: "kStrafe", 13: "kRepeatedHit", 14: "kLightningStrike", 15: "kDecoy",
      16: "kValkyrie", 17: "kScatteredBolts", 18: "kSelfState", 19: "kBreath", 20: "kStaticField", 21: "kTelekinesis",
      22: "kNova", 23: "kSelfStateWithMissile", 24: "kMissileWall", 25: "kEnchant", 26: "kChainLightning", 27: "kTeleport",
      28: "kMissileAtTarget", 29: "kThunderStorm", 30: "kCurse", 31: "kRaiseSkeleton", 32: "kPoisonDagger",
      55: "kCorpseExplosion", 65: "kFriendlyAura", 66: "kEnemyAura", 67: "kCharge", 83: "(none: kFireHit is a START func 42)",
      84: "kHatch", 85: "kTransMissile", 86: "kMaggotDown", 87: "kMaggotLay", 91: "kNest", 95: "kMonsterBreath", 96: "kBestow",
      98: "kMonTeleport", 99: "kMonPoisonNova", 100: "kDiabCold", 101: "kFingerMageSpider", 102: "kDiabWall",
      103: "kDiabRun", 104: "kDiabPrison", 105: "kDesertTurret", 106: "kArcaneTower", 108: "kRegurgitatorEat",
      109: "kMonFrenzy", 112: "kMonCurseCast", 113: "kBookOrScroll", 124: "kStormAroundCaster", 125: "kWakeOfDestruction",
      126: "kImpInferno", 128: "kCryHelp", 129: "kImpTeleport", 131: "kOverseerWhip", 132: "kImpFireMissile",
      133: "kImpregnate", 134: "kSiegeBeastStomp", 135: "kMinionSpawner", 136: "kDeathMaul", 139: "kBaalColdMissiles",
      140: "kBaalTentacle", 148: "kDoomKnightMissile", 149: "kVariantMissile", 152: "kDiabLight"}

use = {}
for line in open(sys.argv[1]):
    m = re.match(r"^(\d+) (.*?) do (\d*) st", line)
    if m and m.group(3):
        use.setdefault(int(m.group(3)), []).append((int(m.group(1)), m.group(2)))
unnamed = {k: v for k, v in use.items() if k not in DO}
print("distinct do values used:", len(use), "named in d2d skills.hpp:", len([k for k in use if k in DO]), "unnamed:", len(unnamed))
for k in sorted(unnamed):
    print(" unnamed do", k, "rows", ", ".join(f"{i} {n}" for i, n in unnamed[k]))
