#!/usr/bin/env python3
"""Writes builds/*.json (same schema as internal/plandata/build-paladin-normal30.json plus main_attack).
Run from this directory; validates prerequisites and reqlevels against the d2tables gear.json (V from the MPQ tables; run cmd/d2tables first)."""
import json, os

TABLES = os.environ.get("D2_TABLES_DIR") or os.path.expanduser("~/.cache/d2lod/tables")
D = json.load(open(os.path.join(TABLES, "gear.json")))["skills"]

def stats(rows):
    return rows

def build(name, cls, schema_tags, seq, stat_rows, hotkeys, main, respec=None):
    rows = []
    for clvl in range(2, 31):
        s, d, v, e = stat_rows[clvl]
        assert s + d + v + e == 5, (name, clvl)
        sk = {}
        for k in seq.get(clvl, []):
            sk[k] = sk.get(k, 0) + 1
        rows.append({"clvl": clvl, "strength": s, "dexterity": d, "vitality": v, "energy": e,
                     "skills": [{"skill": k, "points": p} for k, p in sk.items()]})
    # validate: reqlevel at the row, prerequisites owned earlier or same row earlier, max 20
    lv = {}
    for r in rows:
        for sp in r["skills"]:
            for _ in range(sp["points"]):
                k = sp["skill"]
                info = D[k]
                assert r["clvl"] >= info["req_level"] or False, (name, k, r["clvl"], info["req_level"])
                for q in info["req"]:
                    assert lv.get(q, 0) > 0, (name, k, "needs", q, r["clvl"])
                lv[k] = lv.get(k, 0) + 1
                assert lv[k] <= 20
    out = {"build": name, "class": cls,
           "schema": "per-clvl allocation, same shape as build-paladin-normal30.json (leveler.ParseBuild). main_attack: extra, for combat: skill to use as main attack per clvl range, weapon says which equipped weapon type it needs ('any' = none). Quest points (Den, Radament, Izual) are folded into the row where the guide spends them; the planner pulls rows forward when more points are unspent than the level grants.",
           "tags": schema_tags,
           "respec": respec or {"enabled": False},
           "hotkeys": hotkeys, "main_attack": main, "rows": rows}
    json.dump(out, open(f"builds/{name}.json", "w"), indent=1)

def stat_table(spec):
    t = {}
    for clvl in range(2, 31):
        t[clvl] = spec(clvl)
    return t

# ---- Necromancer, Maxroll leveling (V guide https://maxroll.gg/d2/guides/necromancer-leveling, fetched 2026-10-29)
def nec_stats(c):
    if c in (2, 3): return (0, 0, 3, 2)      # V Maxroll: levels 2-3 Energy (guide says all; U split: 3 Vit 2 Ene, see tags)
    if c <= 11: return (0, 0, 5, 0)
    if c in (12, 13): return (5, 0, 0, 0)
    if c <= 16: return (0, 0, 5, 0)
    if c <= 20: return (5, 0, 0, 0)
    if c == 21: return (2, 0, 3, 0)
    return (0, 0, 5, 0)

def nec_stats_v(c):
    # exact Maxroll: 2-3 all Energy
    s = nec_stats(c)
    return (0, 0, 0, 5) if c in (2, 3) else s

leveling = {2: ["Teeth"], 3: ["Teeth"], 4: ["Teeth"], 5: ["Teeth", "Teeth"],  # 2-5 + Den of Evil point
            6: ["Clay Golem"], 7: ["Teeth"], 8: ["Teeth"], 9: ["Teeth"], 10: ["Teeth"],
            11: ["Amplify Damage"], 12: ["Iron Maiden"], 13: ["Teeth"], 14: ["Bone Armor"],
            15: ["Bone Wall", "Bone Wall"],  # Radament point + 15
            16: ["Bone Wall"], 17: ["Corpse Explosion"], 18: ["Bone Spear"], 19: ["Bone Spear"], 20: ["Bone Spear"],
            21: ["Corpse Explosion", "Corpse Explosion"],  # Izual point + 21
            22: ["Corpse Explosion"], 23: ["Corpse Explosion"], 24: ["Corpse Explosion"], 25: ["Corpse Explosion"],
            26: ["Corpse Explosion"], 27: ["Corpse Explosion"], 28: ["Corpse Explosion"], 29: ["Corpse Explosion"], 30: ["Corpse Explosion"]}
build("necromancer_leveling_normal30", "necromancer",
      "Skills 2-26: V (Maxroll Necromancer leveling https://maxroll.gg/d2/guides/necromancer-leveling, fetched 2026-10-29; guide says Amplify Damage/Iron Maiden points go into Teeth if you hold a Wand, kept as the guide's default). Quest points folded into rows 5, 15, 21 (Den of Evil, Radament, Izual). Skills 27-30: U (own choice: Corpse Explosion, the guide's main damage after its clvl 27 respec; the respec itself is NOT used). Stats 2-26: V Maxroll (2-3 Energy, 4-11 Vit, 12-13 Str, 14-16 Vit, 17-21 22 Str + 3 Vit, 22-26 Vit); 27-30: U (all Vit, Maxroll says Str only for gear). Prerequisites and reqlevels checked by gen_builds.py against skills.txt (V Patch_D2.mpq). Start values 15 str 25 dex 15 vit 25 ene.",
      leveling, stat_table(nec_stats_v),
      [{"key": "F1", "skill": "Teeth", "hand": "right", "from_clvl": 2},
       {"key": "F2", "skill": "Clay Golem", "hand": "right", "from_clvl": 6},
       {"key": "F3", "skill": "Amplify Damage", "hand": "right", "from_clvl": 11},
       {"key": "F4", "skill": "Bone Armor", "hand": "right", "from_clvl": 14},
       {"key": "F5", "skill": "Corpse Explosion", "hand": "right", "from_clvl": 17},
       {"key": "F6", "skill": "Bone Spear", "hand": "right", "from_clvl": 18}],
      [{"from_clvl": 1, "until_clvl": 17, "skill": "Teeth", "weapon": "any"},
       {"from_clvl": 18, "until_clvl": 30, "skill": "Bone Spear", "weapon": "any", "note": "Teeth stays the fallback when mana is low; Corpse Explosion needs a corpse target"}],
      {"at_clvl": 27, "enabled": False, "require": "Maxroll respecs at 27 to Corpse Explosion 10, Bone Spear 9, Bone Prison 3; NOT implemented, rows do not depend on it", "refund_all": True,
       "respend_pool": [{"skill": "Corpse Explosion", "points": 10}, {"skill": "Bone Spear", "points": 9}, {"skill": "Bone Prison", "points": 3}]})

# ---- Necromancer summoner (U: own order, end state follows Maxroll summoner
# https://maxroll.gg/d2/guides/summoner-necromancer-guide V: Raise Skeleton, Skeleton Mastery, Corpse Explosion maxed; Teeth is only the prerequisite)
summ = {2: ["Raise Skeleton"], 3: ["Skeleton Mastery"], 4: ["Raise Skeleton"], 5: ["Teeth", "Skeleton Mastery"],
        6: ["Clay Golem"], 7: ["Raise Skeleton"], 8: ["Skeleton Mastery"], 9: ["Raise Skeleton"], 10: ["Skeleton Mastery"],
        11: ["Amplify Damage"], 12: ["Raise Skeletal Mage"], 13: ["Raise Skeletal Mage"], 14: ["Bone Armor"],
        15: ["Corpse Explosion", "Raise Skeletal Mage"], 16: ["Raise Skeletal Mage"], 17: ["Raise Skeletal Mage"],
        18: ["Raise Skeleton"], 19: ["Skeleton Mastery"], 20: ["Raise Skeletal Mage"],
        21: ["Corpse Explosion", "Raise Skeleton"], 22: ["Skeleton Mastery"], 23: ["Raise Skeletal Mage"],
        24: ["Corpse Explosion"], 25: ["Raise Skeleton"], 26: ["Skeleton Mastery"], 27: ["Corpse Explosion"],
        28: ["Raise Skeletal Mage"], 29: ["Corpse Explosion"], 30: ["Raise Skeleton"]}
build("necromancer_summoner_normal30", "necromancer",
      "Skills: U (own order for a bot with no weapon swap and no positioning micro: warriors first, Clay Golem at 6, mages from 12, Bone Armor 14 as Maxroll leveling, Corpse Explosion from 15; end state direction V from Maxroll Summoner Necromancer https://maxroll.gg/d2/guides/summoner-necromancer-guide fetched 2026-10-29: maxed Raise Skeleton, Skeleton Mastery, Corpse Explosion; 1-point Iron Golem, Summon Resist, Decrepify, Dim Vision, Lower Resist, Revive come after clvl 30 and are not in these rows). Quest points folded into rows 5, 15, 21. Stats: same table as necromancer_leveling_normal30 (V Maxroll leveling; guide says summoner Str 25 for gear, rest Vit). Prerequisites and reqlevels checked against skills.txt (V Patch_D2.mpq). Teeth gets one point only (Corpse Explosion prerequisite and the wand attack).",
      summ, stat_table(nec_stats_v),
      [{"key": "F1", "skill": "Teeth", "hand": "right", "from_clvl": 5},
       {"key": "F2", "skill": "Raise Skeleton", "hand": "right", "from_clvl": 2},
       {"key": "F3", "skill": "Clay Golem", "hand": "right", "from_clvl": 6},
       {"key": "F4", "skill": "Amplify Damage", "hand": "right", "from_clvl": 11},
       {"key": "F5", "skill": "Raise Skeletal Mage", "hand": "right", "from_clvl": 12},
       {"key": "F6", "skill": "Bone Armor", "hand": "right", "from_clvl": 14},
       {"key": "F7", "skill": "Corpse Explosion", "hand": "right", "from_clvl": 15}],
      [{"from_clvl": 1, "until_clvl": 14, "skill": "Raise Skeleton", "weapon": "any", "note": "attack skill is Teeth (1 point) or the wand/melee; the army does the killing: recast Raise Skeleton to the cap"},
       {"from_clvl": 15, "until_clvl": 30, "skill": "Corpse Explosion", "weapon": "any", "note": "needs a corpse; Teeth as the corpse-free fallback"}])

# ---- Amazon bow: V Maxroll leveling https://maxroll.gg/d2/guides/amazon-leveling-guide (fetched 2026-10-29), adapted: no weapon swap in the bot
def ama_stats(c):
    if c <= 11: return (0, 0, 5, 0)
    if c == 12: return (2, 3, 0, 0)
    if c == 13: return (5, 0, 0, 0)
    if c <= 16: return (0, 0, 0, 5)
    return (0, 0, 5, 0)

bow = {2: ["Magic Arrow"]}
for c in range(3, 11): bow[c] = ["Fire Arrow"]
bow[11] = ["Multiple Shot"]
for c in range(12, 31): bow[c] = ["Exploding Arrow"]
build("amazon_bow_normal30", "amazon",
      "Skills: V Maxroll for Fire Arrow (2-5 and Den/Izual points), Multiple Shot 11 and Exploding Arrow 12+ (https://maxroll.gg/d2/guides/amazon-leveling-guide, fetched 2026-10-29). U adaptation: Maxroll spends 6-10 on Poison Javelin and swaps weapons; the bot has no weapon swap, so those points go to Fire Arrow (a bow skill). Magic Arrow 1 point at clvl 2 (Maxroll takes it from the Den of Evil quest point; it is the Multiple Shot prerequisite, V skills.txt). Quest points are not separate rows: the planner pulls rows forward. Stats V Maxroll: 2-11 Vit; 12: 3 Dex 2 Str; 13 Str; 14-16 Energy; 17+ Vit (30: U continues Vit). Hunter's Bow needs 28 Dex (V weapons.txt), start Dex 25: the clvl 12 +3 Dex is when it becomes wearable; before that a Short Bow (15 Dex). Start gear is a Javelin + Buckler (V charstats.txt, V live smoke-amazon): buy bow + Arrows at Charsi first (docs/gear.md). Prerequisites and reqlevels checked against skills.txt (V Patch_D2.mpq).",
      bow, stat_table(ama_stats),
      [{"key": "F1", "skill": "Magic Arrow", "hand": "right", "from_clvl": 2},
       {"key": "F2", "skill": "Fire Arrow", "hand": "right", "from_clvl": 3},
       {"key": "F3", "skill": "Multiple Shot", "hand": "right", "from_clvl": 11},
       {"key": "F4", "skill": "Exploding Arrow", "hand": "right", "from_clvl": 12}],
      [{"from_clvl": 1, "until_clvl": 2, "skill": "Throw", "weapon": "jave", "note": "start javelin stack until a bow and arrows are bought"},
       {"from_clvl": 3, "until_clvl": 11, "skill": "Fire Arrow", "weapon": "bow", "ammo": "bowq", "note": "Magic Arrow (free) when mana is low (V smoke-amazon: costs 0 mana at 6)"},
       {"from_clvl": 12, "until_clvl": 30, "skill": "Exploding Arrow", "weapon": "bow", "ammo": "bowq", "note": "Fire Arrow as the cheap fallback; Magic Arrow when out of mana"}])

# ---- Amazon javelin: U own design from Maxroll's Poison Javelin early part, Lightning Bolt behind it
jav = {}
jav[6] = ["Poison Javelin"] * 5  # 4 points saved from clvl 2-5 plus the clvl 6 point
for c in range(7, 14): jav[c] = ["Poison Javelin"]
for c in range(14, 31): jav[c] = ["Lightning Bolt"]
build("amazon_javelin_normal30", "amazon",
      "Skills: U (own design; V only that Poison Javelin is Maxroll's clvl 6-10 skill in the Act 1 leveling guide https://maxroll.gg/d2/guides/amazon-leveling-guide, fetched 2026-10-29, and Lightning Bolt requires Poison Javelin at clvl 12, V skills.txt). Rows 2-5 are empty (points saved) and row 6 spends 5 points, because Poison Javelin unlocks at clvl 6 (V reqlevel). No weapon swap needed: javelins only. Javelins are a stack with tiny durability (V smoke-amazon: start Javelin max stack 60, durability 2) so restock is part of the plan (docs/gear.md). Stats: U (Vit, then 3 Str 2 Dex per level 12-16; this does NOT reach Maiden Javelin str 33 / dex 47, V weapons.txt, so plain Javelin/Pilum stacks are used in Normal). Prerequisites checked against skills.txt (V Patch_D2.mpq).",
      jav, stat_table(lambda c: (0, 0, 5, 0) if c <= 11 else (3, 2, 0, 0) if c <= 16 else (0, 0, 5, 0)),
      [{"key": "F1", "skill": "Poison Javelin", "hand": "right", "from_clvl": 6},
       {"key": "F2", "skill": "Lightning Bolt", "hand": "right", "from_clvl": 14}],
      [{"from_clvl": 1, "until_clvl": 5, "skill": "Throw", "weapon": "jave", "ammo": "self", "note": "plain Throw until Poison Javelin at clvl 6"},
       {"from_clvl": 6, "until_clvl": 13, "skill": "Poison Javelin", "weapon": "jave", "ammo": "self"},
       {"from_clvl": 14, "until_clvl": 30, "skill": "Lightning Bolt", "weapon": "jave", "ammo": "self", "note": "Poison Javelin when the target is lightning immune"}])

# ---- Druid fire: V Maxroll Druid leveling https://maxroll.gg/d2/guides/druid-leveling (docs/classes/druid.md section 1; Firestorm 1-5,
# Molten Boulder 6-7, Firestorm 8-11, Fissure (Eruption) 12+, Firestorm from Radament/Izual). The respec to Tornado/Hurricane at 38 is not used.
# U extras for a bot that has no positioning micro: Oak Sage 1 (clvl 16), Arctic Blast 1 + Cyclone Armor 1 (clvl 17-18) before the Fissure points.
def dru_stats(c):
    if c <= 11: return (0, 0, 5, 0)
    if c <= 13: return (5, 0, 0, 0)
    if c <= 21: return (0, 0, 5, 0)
    if c <= 25: return (5, 0, 0, 0)
    if c == 26: return (4, 0, 1, 0)
    return (0, 0, 5, 0)

dru = {2: ["Firestorm"], 3: ["Firestorm"], 4: ["Firestorm"], 5: ["Firestorm", "Firestorm"],  # 2-5 + Den of Evil point
       6: ["Molten Boulder"], 7: ["Molten Boulder"], 8: ["Firestorm"], 9: ["Firestorm"], 10: ["Firestorm"], 11: ["Firestorm"],
       12: ["Eruption"], 13: ["Eruption"], 14: ["Eruption"],
       15: ["Eruption", "Firestorm"],  # Radament point folded (Maxroll: Firestorm from the quest points)
       16: ["Oak Sage"], 17: ["Arctic Blast"], 18: ["Cyclone Armor"]}
for c in range(19, 31): dru[c] = ["Eruption"]
dru[21] = ["Eruption", "Firestorm"]  # Izual point folded
build("druid_fire_normal30", "druid",
      "Skills 2-15 and the quest points: V Maxroll Druid leveling https://maxroll.gg/d2/guides/druid-leveling (fetched earlier, docs/classes/druid.md): Firestorm 1-5, Molten Boulder 6-7, Firestorm 8-11, Fissure (skills.txt Eruption) 12+, Firestorm from the Radament and Izual points (folded into rows 15 and 21; Den point into row 5). Skills 16-30: U (Oak Sage 16, Arctic Blast 17 and Cyclone Armor 18 added as cheap survivability for a bot; the rest into Fissure; Maxroll's respec to Tornado/Hurricane at 38 is not used). Stats: V Maxroll (2-11 Vit, 12-13 Str, 14-21 Vit, 22-26 Str); 26 split 4 Str 1 Vit and 27-30 Vit are U. Staff of any kind works: all skills are spells, no weapon swap (V docs/classes/druid.md). Prerequisites and reqlevels checked by gen_builds.py against skills.txt (V Patch_D2.mpq).",
      dru, stat_table(dru_stats),
      [{"key": "F1", "skill": "Firestorm", "hand": "right", "from_clvl": 2},
       {"key": "F2", "skill": "Molten Boulder", "hand": "right", "from_clvl": 6},
       {"key": "F3", "skill": "Eruption", "hand": "right", "from_clvl": 12},
       {"key": "F4", "skill": "Cyclone Armor", "hand": "right", "from_clvl": 18},
       {"key": "F5", "skill": "Oak Sage", "hand": "right", "from_clvl": 16}],
      [{"from_clvl": 1, "until_clvl": 11, "skill": "Firestorm", "weapon": "any", "note": "Molten Boulder from 6 for fire immune targets"},
       {"from_clvl": 12, "until_clvl": 30, "skill": "Eruption", "weapon": "any", "note": "Fissure (skills.txt Eruption); Firestorm and Molten Boulder as fallbacks; ground-aimed, 2 s cooldown tuning is D2R only (U in 1.14d)"}])

# ---- Assassin Fire Blast / Wake of Fire: V Maxroll Assassin leveling https://maxroll.gg/d2/guides/assassin-leveling (docs/classes/assassin.md):
# Fire Blast (skills.txt Fire Trauma) 2-5 + Claw Mastery from Den of Evil, Burst of Speed (Quickness) at 6, Fire Blast to 11, Wake of Fire (Wake of Fire Sentry) 12+.
def ass_stats(c):
    if c <= 3: return (0, 0, 0, 5)
    if c <= 10: return (0, 0, 5, 0)
    if c == 11: return (5, 0, 0, 0)
    if c <= 21: return (0, 0, 5, 0)
    if c <= 24: return (5, 0, 0, 0)
    if c == 25: return (3, 0, 2, 0)
    return (0, 0, 5, 0)

ass = {2: ["Fire Trauma"], 3: ["Fire Trauma"], 4: ["Fire Trauma"], 5: ["Fire Trauma", "Claw Mastery"],  # Den of Evil point = Claw Mastery
       6: ["Quickness"]}
for c in range(7, 12): ass[c] = ["Fire Trauma"]
for c in range(12, 31): ass[c] = ["Wake of Fire Sentry"]
ass[15] = ["Wake of Fire Sentry", "Wake of Fire Sentry"]  # Radament point
ass[21] = ["Wake of Fire Sentry", "Wake of Fire Sentry"]  # Izual point
ass[30] = ["Fire Trauma"]                                  # Wake of Fire reaches its cap of 20 at clvl 29: the last point goes into Fire Blast (Maxroll: Fire Blast 32-38)
build("assassin_fireblast_normal30", "assassin",
      "Skills: V Maxroll Assassin leveling https://maxroll.gg/d2/guides/assassin-leveling (docs/classes/assassin.md): Fire Blast 2-5, Claw Mastery from the Den of Evil point, Burst of Speed at 6, Fire Blast to 11, Wake of Fire from 12; Radament and Izual points folded into rows 15 and 21 (Wake of Fire). U: exact split of 6-11 (the guide text is ambiguous) and the last point (clvl 30) into Fire Blast once Wake of Fire is at 20; the respec to Lightning Sentry traps at 36-38 is not used. Stats: V Maxroll (2-3 Energy, 4-10 Vit, 11 Str, then Vit; 22-25 add 18 Str); the 3/2 split in row 25 and Vit 26-30 are U. No weapon is required (Fire Blast and traps are skill attacks, claws only add Claw Mastery, V skills.txt itypea empty); martial arts is not used (needs two-step timing, docs/classes/assassin.md). Prerequisites and reqlevels checked by gen_builds.py against skills.txt (V Patch_D2.mpq).",
      ass, stat_table(ass_stats),
      [{"key": "F1", "skill": "Fire Trauma", "hand": "right", "from_clvl": 2},
       {"key": "F2", "skill": "Quickness", "hand": "right", "from_clvl": 6},
       {"key": "F3", "skill": "Wake of Fire Sentry", "hand": "right", "from_clvl": 12}],
      [{"from_clvl": 1, "until_clvl": 11, "skill": "Fire Trauma", "weapon": "any", "note": "Fire Blast; melee Attack is the fallback"},
       {"from_clvl": 12, "until_clvl": 30, "skill": "Wake of Fire Sentry", "weapon": "any", "note": "trap laid at the pack; Fire Blast for bosses and while the trap stands"}])

# ---- Barbarian Bash / Double Swing / Frenzy melee leveler: V Maxroll Barbarian leveling https://maxroll.gg/d2/guides/barbarian-leveling (docs/classes/barbarian.md):
# Bash 2-5, weapon mastery from the Den point, Double Swing from 6 (its reqlevel), Howl/Shout on the way to Battle Orders, Double Throw as the
# prerequisite of Frenzy (reqlevel 24), Iron Skin at 18, War Cry/Battle Command/Whirlwind (reqlevel 30) are not taken.
def bar_stats(c):
    if c <= 7: return (0, 0, 5, 0)
    if c <= 10: return (0, 5, 0, 0)
    if c <= 13: return (5, 0, 0, 0)
    if c <= 21: return (0, 0, 5, 0)
    if c <= 24: return (5, 0, 0, 0)
    return (0, 0, 5, 0)

bar = {2: ["Bash"], 3: ["Bash"], 4: ["Bash"], 5: ["Axe Mastery", "Axe Mastery"],  # Den of Evil point = Axe Mastery (starting weapon is an axe, U)
       6: ["Double Swing"], 7: ["Howl"], 8: ["Shout"], 9: ["Double Swing"], 10: ["Double Swing"], 11: ["Axe Mastery"],
       12: ["Double Throw"], 13: ["Axe Mastery"], 14: ["Axe Mastery"],
       15: ["Increased Stamina", "Increased Stamina"],  # Radament point
       16: ["Bash"], 17: ["Bash"], 18: ["Iron Skin"], 19: ["Iron Skin"], 20: ["Axe Mastery"],
       21: ["Bash", "Bash"],  # Izual point
       22: ["Double Swing"], 23: ["Double Swing"], 24: ["Frenzy"], 25: ["Frenzy"], 26: ["Battle Orders"], 27: ["Battle Orders"],
       28: ["Frenzy"], 29: ["Frenzy"], 30: ["Frenzy"]}
build("barbarian_melee_normal30", "barbarian",
      "Skills: V Maxroll Barbarian leveling https://maxroll.gg/d2/guides/barbarian-leveling (docs/classes/barbarian.md): Bash then Double Swing, Howl/Shout/Battle Orders, masteries, Double Throw only as the prerequisite of Frenzy (reqlevel 24). U: the exact row order is this agent's reading of the guide (BARBLIVE), Axe Mastery instead of the guide's Mace Mastery because the starting weapon is an axe (U, check live), the Whirlwind/War Cry respec at 30+ is not used (needs a physical-immune answer, docs/classes/barbarian.md section 1). Quest points (Den, Radament, Izual) folded into rows 5, 15, 21. Stats: V Maxroll (mostly Vitality, Dexterity then Strength early); the 5-point rows are U. Weapon: any melee weapon (Bash/Double Swing/Frenzy itypea mele, V skills.txt). Prerequisites and reqlevels checked by gen_builds.py against skills.txt (V Patch_D2.mpq).",
      bar, stat_table(bar_stats),
      [{"key": "F1", "skill": "Battle Orders", "hand": "right", "from_clvl": 26},  # same keys as the combat profile layout; attacks are LEFT skills (SetLeft)
       {"key": "F2", "skill": "Shout", "hand": "right", "from_clvl": 8},
       {"key": "F3", "skill": "Iron Skin", "hand": "right", "from_clvl": 18}],
      [{"from_clvl": 1, "until_clvl": 5, "skill": "Bash", "weapon": "any"},
       {"from_clvl": 6, "until_clvl": 23, "skill": "Double Swing", "weapon": "any", "note": "Bash is the fallback"},
       {"from_clvl": 24, "until_clvl": 30, "skill": "Frenzy", "weapon": "any", "note": "Double Swing and Bash are the fallbacks"}])
