#!/usr/bin/env python3
"""Writes the proposed endgame (clvl 2..90, no respec) build tables, same schema as internal/gear/builds/*.json.
Run from this directory: python3 gen_endgame.py. Validates skill prerequisites and reqlevels against
the d2tables gear.json (V from skills.txt; run cmd/d2tables first). Skill points: greedy over a priority plan; when no plan entry is
unlocked the point is banked (saved) and spent in the row where the skill unlocks (precedent: amazon_javelin_normal30)."""
import json, os
HERE = os.path.dirname(os.path.abspath(__file__))
TABLES = os.environ.get("D2_TABLES_DIR") or os.path.expanduser("~/.cache/d2lod/tables")  # run cmd/d2tables first
DATA = json.load(open(os.path.join(TABLES, "gear.json")))
D, C = DATA["skills"], DATA["classes"]
MAXCLVL = 90

ARMOR = json.load(open(os.path.join(TABLES, "armor.json")))["armor"]
MS = json.load(open(os.path.join(HERE, "../../../../internal/gear/milestones.json")))["builds"]

def req(code):
    """(str, dex) a base needs: armor.json (V armor.txt, no Dex column), else data.json weapons/shields (V)."""
    if code in ARMOR:
        return ARMOR[code]["req_str"], 0
    b = DATA["bases"][code]
    return b["req_str"], b["req_dex"]

def stages_for(name, dex_floor=()):
    """Str/Dex stages (until_clvl, str, dex, ene) computed from internal/gear/milestones.json (U item choice, V
    requirements); dex_floor = [(clvl, dex)] adds Dex that is not an armor requirement (U, e.g. block)."""
    pts = sorted({m["clvl"] for m in MS[name]} | {c for c, _ in dex_floor})
    out = []
    for c in pts:
        s = d = 0
        for m in MS[name]:
            if m["clvl"] <= c:
                for code in m["items"]:
                    rs, rd = req(code); s, d = max(s, rs), max(d, rd)
        for fc, fd in dex_floor:
            if fc <= c: d = max(d, fd)
        out.append((c, s, d, 0))
    if out[-1][0] != MAXCLVL:
        out.append((MAXCLVL,) + out[-1][1:])
    return out
# U timing: Den, Radament, Izual (2) per difficulty, folded into the row where the planner pulls them forward.
QUEST = {8: 1, 15: 1, 26: 2, 38: 1, 44: 1, 56: 2, 68: 1, 72: 1, 82: 2}

def skill_rows(name, plan):
    lv, bank, rows = {}, 0, []
    for clvl in range(2, MAXCLVL + 1):
        bank += 1 + QUEST.get(clvl, 0)
        got = {}
        while bank:
            pick = None
            for ent in plan:
                sk, cap = ent[0], ent[1]
                i = D[sk]
                if lv.get(sk, 0) < cap and clvl >= max(i["req_level"], ent[2] if len(ent) > 2 else 0) and all(lv.get(q, 0) > 0 for q in i["req"]):
                    pick = sk; break
            if pick is None: break
            lv[pick] = lv.get(pick, 0) + 1; got[pick] = got.get(pick, 0) + 1; bank -= 1
            assert lv[pick] <= 20
        rows.append((clvl, got))
    return rows, lv, bank

def stat_rows(cls, stages):
    """stages: list of (until_clvl, str, dex, ene) targets; deficits first (str, dex, ene), rest vitality."""
    c = C[cls]; cur = [c["str"], c["dex"], c["vit"], c["ene"]]; out = {}
    for clvl in range(2, MAXCLVL + 1):
        tgt = next((s for s in stages if clvl <= s[0]), stages[-1])
        add = [0, 0, 0, 0]; left = 5
        for idx, t in ((0, tgt[1]), (1, tgt[2]), (3, tgt[3])):
            n = max(0, min(left, t - cur[idx])); add[idx] = n; left -= n
        # Look-ahead: a later stage whose deficit no longer fits into the levels left before its deadline pulls points
        # forward from Vitality (found by TestMilestoneStats: the Amazon Dex 110 by clvl 35 was out of reach otherwise).
        extra, why = 0, None
        for st in stages:
            if st[0] < clvl:
                continue
            after = [cur[i] + add[i] for i in range(4)]
            need = sum(max(0, t - after[i]) for i, t in ((0, st[1]), (1, st[2]), (3, st[3])))
            f = min(left, max(0, need - 5 * (st[0] - clvl)))
            if f > extra:
                extra, why = f, st
        if why:
            for idx, t in ((0, why[1]), (1, why[2]), (3, why[3])):
                n = max(0, min(extra, t - cur[idx] - add[idx])); add[idx] += n; extra -= n; left -= n
        add[2] = left
        for i in range(4): cur[i] += add[i]
        out[clvl] = tuple(add)
    return out, cur

def build(name, cls, tags, plan, stages, hotkeys, main):
    srows, lv, bank = skill_rows(name, plan)
    stats, final = stat_rows(cls, stages)
    rows = []
    for clvl, got in srows:
        s, d, v, e = stats[clvl]
        rows.append({"clvl": clvl, "strength": s, "dexterity": d, "vitality": v, "energy": e,
                     "skills": [{"skill": k, "points": p} for k, p in got.items()]})
    out = {"build": name, "class": CLS_NAME[cls], "schema": SCHEMA, "tags": tags, "respec": {"enabled": False},
           "hotkeys": hotkeys, "main_attack": main, "rows": rows}
    json.dump(out, open(os.path.join(HERE, name + ".json"), "w"), indent=1)
    print(f"{name}: skills {dict(sorted(lv.items(), key=lambda x: -x[1]))} unspent_bank_at_90={bank} stats(str,dex,vit,ene)@90={final}")

CLS_NAME = {"sor": "sorceress", "pal": "paladin", "nec": "necromancer", "dru": "druid", "ass": "assassin", "bar": "barbarian", "ama": "amazon"}
SCHEMA = ("per-clvl allocation, same shape as internal/gear/builds/*.json (leveler.ParseBuild) plus main_attack; rows clvl 2..90, "
          "NO respec. A row with no skills means the point is saved (no unlocked plan skill); the planner spends it in the row where the skill unlocks. "
          "Quest points (Den, Radament, Izual in each difficulty) are folded into rows 8, 15, 26, 38, 44, 56, 68, 72, 82 (U timing; planner pulls rows forward).")
def hk(*a): return [{"key": k, "skill": s, "hand": h, "from_clvl": f} for k, s, h, f in a]
def ma(*a): return [dict(zip(("from_clvl", "until_clvl", "skill", "weapon", "note"), x)) for x in a]

# 1 Sorceress: Blizzard (V Maxroll Blizzard Sorceress https://maxroll.gg/d2/guides/blizzard-sorceress fetched 2026-10-29: max Blizzard, Glacial Spike, Ice Blast, Ice Bolt, Cold Mastery; 1 each Static Field, Telekinesis, Teleport, Frozen Armor, Warmth)
build("sorceress_blizzard_endgame90", "sor",
      "Skills: V end state from Maxroll Blizzard Sorceress (fetched 2026-10-29). U ordering/no-respec adaptation: Ice Bolt, Ice Blast and Glacial Spike are all maxed in the endgame, so the early attack skills are never wasted; Blizzard from 24 (V skills.txt reqlevel), Cold Mastery from 30, Static Field 1 early for cold immunes (the existing combat fallback). Stats: Str 43 by clvl 24 (Spirit Crystal Sword, V data.json), Str 55 by 55 (Enigma in a Mage Plate, V armor.txt; item choice U), derived from internal/gear/milestones.json, rest Vitality, no Energy. Prerequisites and reqlevels checked against skills.txt (V data.json).",
      [("Ice Bolt", 6), ("Frozen Armor", 1), ("Telekinesis", 1), ("Teleport", 1), ("Static Field", 1), ("Frost Nova", 1), ("Warmth", 1),
       ("Ice Blast", 12), ("Glacial Spike", 6), ("Blizzard", 20), ("Cold Mastery", 20), ("Glacial Spike", 20), ("Ice Blast", 20), ("Ice Bolt", 20), ("Static Field", 20)],
      stages_for("sorceress_blizzard_endgame90"),
      hk(("F1", "Ice Bolt", "right", 2), ("F2", "Static Field", "right", 6), ("F3", "Frozen Armor", "right", 2), ("F4", "Teleport", "right", 18), ("F5", "Glacial Spike", "right", 18), ("F6", "Blizzard", "right", 24), ("F7", "Ice Blast", "right", 8)),
      ma((1, 7, "Ice Bolt", "any", "Fire Bolt (start skill) / staff melee as fallback"), (8, 17, "Ice Blast", "any", "Ice Bolt when mana is low"),
         (18, 23, "Glacial Spike", "any", "Ice Blast single target"), (24, 90, "Blizzard", "any", "ground-targeted, has a cast delay; Glacial Spike/Ice Blast during the delay; Static Field on cold immunes and bosses (Hell floor 50% life)")))

# 2 Paladin: Blessed Hammer (Hammerdin), no respec (V Maxroll Blessed Hammer Paladin https://maxroll.gg/d2/guides/blessed-hammer-paladin: singles Holy Shield, Cleansing, Redemption, Fanaticism; max Blessed Hammer, Vigor, Concentration, Blessed Aim, Resist Lightning)
build("paladin_hammerdin_endgame90", "pal",
      "Skills: V end state Maxroll Blessed Hammer Paladin (fetched 2026-10-29). U adaptation: no respec, so clvl 2-17 spend only Might, Smite, Holy Bolt, Charge (Holy Shield prerequisites, V skills.txt) and bank the rest (Prayer/Defiance/Cleansing/Blessed Aim 1 from 12); row 18 spends the bank into Blessed Hammer (reqlevel 18, needs Holy Bolt, V) like amazon_javelin_normal30 rows 2-5/6. Smite (needs a shield) and the starting sword carry clvl 2-17. Holy Shield 1 at 24. Stats: Str 43 by clvl 25 (Spirit sword), 86 by clvl 45 (Sacred Targe, V armor.txt; Fortitude in a Dusk Shroud needs 77, V), from internal/gear/milestones.json; Dex 70 for block with Holy Shield (U; Maxroll: enough Dex for 75% block), rest Vitality.",
      [("Might", 1), ("Smite", 1), ("Holy Bolt", 1), ("Charge", 1), ("Blessed Hammer", 20), ("Holy Shield", 1), ("Prayer", 1, 12), ("Defiance", 1, 12), ("Cleansing", 1, 12), ("Vigor", 5), ("Blessed Aim", 1, 12),
       ("Concentration", 20), ("Vigor", 20), ("Redemption", 1), ("Fanaticism", 1), ("Blessed Aim", 20, 40), ("Resist Lightning", 20, 40), ("Salvation", 1, 60), ("Defiance", 20, 80)],
      stages_for("paladin_hammerdin_endgame90", dex_floor=[(45, 40), (MAXCLVL, 70)]),
      hk(("F1", "Smite", "right", 2), ("F2", "Blessed Hammer", "right", 18), ("F3", "Holy Shield", "right", 24), ("F4", "Concentration", "right", 18), ("F5", "Vigor", "right", 18), ("F6", "Fanaticism", "right", 30)),
      ma((1, 17, "Smite", "any", "melee; needs a shield; Might aura"), (18, 90, "Blessed Hammer", "any", "magic damage, hits all around; stand still in the pack; Concentration aura; Holy Shield up")))

# 2b Paladin variant (W4): Smite has only 1 point in clvl 2-17 of the table above. Offline arithmetic (V, this table): 16 points
# arrive in clvl 2-17 and 15 of them are Might/Smite/Holy Bolt/Charge/bank. This variant (U, needs a live Normal A1-A2 check)
# puts 8 more points in Smite (to 9) before 12 and Sacrifice 1 + Zeal 5 at 12-17 (Zeal needs Sacrifice and clvl 12, V skills.txt);
# the points come out of the last fillers (Blessed Aim, Defiance), so the Hammerdin end state is unchanged otherwise.
build("paladin_hammerdin_zeal_endgame90", "pal",
      "Skills: V end state Maxroll Blessed Hammer Paladin (fetched 2026-10-29), same as paladin_hammerdin_endgame90. U variant for a stronger clvl 2-17: Smite to 9 before clvl 12, Sacrifice 1 then Zeal 5 from clvl 12 (Zeal reqlevel 12, needs Sacrifice, V skills.txt); the 14 extra points are taken from the last fillers (Blessed Aim and Defiance) at clvl 65+. U whether this clears Normal Act 1-2 better than 1 Smite point (needs a live run); Smite needs a shield.",
      [("Might", 1), ("Smite", 1), ("Holy Bolt", 1), ("Smite", 9), ("Charge", 1), ("Sacrifice", 1), ("Zeal", 5, 12), ("Blessed Hammer", 20), ("Holy Shield", 1), ("Prayer", 1, 12), ("Defiance", 1, 12), ("Cleansing", 1, 12), ("Vigor", 5), ("Blessed Aim", 1, 12),
       ("Concentration", 20), ("Vigor", 20), ("Redemption", 1), ("Fanaticism", 1), ("Blessed Aim", 20, 40), ("Resist Lightning", 20, 40), ("Salvation", 1, 60), ("Defiance", 20, 80)],
      stages_for("paladin_hammerdin_zeal_endgame90", dex_floor=[(45, 40), (MAXCLVL, 70)]),
      hk(("F1", "Smite", "right", 2), ("F7", "Zeal", "right", 12), ("F2", "Blessed Hammer", "right", 18), ("F3", "Holy Shield", "right", 24), ("F4", "Concentration", "right", 18), ("F5", "Vigor", "right", 18), ("F6", "Fanaticism", "right", 30)),
      ma((1, 11, "Smite", "any", "melee; needs a shield; Might aura"), (12, 17, "Zeal", "any", "melee, 5 quick hits; needs Sacrifice"), (18, 90, "Blessed Hammer", "any", "magic damage, hits all around; stand still in the pack; Concentration aura; Holy Shield up")))

# 3 Necromancer: Summoner (V Maxroll Summoner Necromancer https://maxroll.gg/d2/guides/summoner-necromancer-guide)
build("necromancer_summoner_endgame90", "nec",
      "Skills: V end state Maxroll Summoner Necromancer (fetched 2026-10-29): 1 each Iron Golem, Summon Resist, Decrepify, Dim Vision, Lower Resist, Revive; max Raise Skeleton, Skeleton Mastery, Corpse Explosion, Amplify Damage (Standard variant). U: prerequisites pulled in (Teeth, Clay/Blood Golem, Golem Mastery, Raise Skeletal Mage, Weaken, Terror, Iron Maiden, Life Tap per skills.txt V), Skeletal Mage 1 point only because Revive needs it; keeps the early order of necromancer_summoner_normal30 in spirit (warriors first). Stats V Maxroll: Strength for gear, rest Vitality; modelled from internal/gear/milestones.json: Str 43 by clvl 24 (Spirit sword), 55 by 55 (Enigma in a Mage Plate, V armor.txt).",
      [("Raise Skeleton", 6), ("Skeleton Mastery", 6), ("Teeth", 1), ("Amplify Damage", 1), ("Clay Golem", 1), ("Bone Armor", 1), ("Corpse Explosion", 10),
       ("Raise Skeleton", 20), ("Skeleton Mastery", 20), ("Raise Skeletal Mage", 1), ("Golem Mastery", 1), ("Summon Resist", 1), ("BloodGolem", 1), ("IronGolem", 1), ("Revive", 1),
       ("Weaken", 1), ("Terror", 1), ("Decrepify", 1), ("Dim Vision", 1), ("Iron Maiden", 1), ("Life Tap", 1), ("Lower Resist", 1), ("Corpse Explosion", 20), ("Amplify Damage", 20), ("Raise Skeletal Mage", 20)],
      stages_for("necromancer_summoner_endgame90"),
      hk(("F1", "Teeth", "right", 2), ("F2", "Raise Skeleton", "right", 1), ("F3", "Amplify Damage", "right", 2), ("F4", "Clay Golem", "right", 6), ("F5", "Bone Armor", "right", 6), ("F6", "Corpse Explosion", "right", 6), ("F7", "Revive", "right", 30), ("F8", "IronGolem", "right", 24)),
      ma((1, 5, "Teeth", "any", "plus Raise Skeleton upkeep"), (6, 90, "Corpse Explosion", "any", "needs a corpse; skeletons/revives do the rest; recast Raise Skeleton/Revive to the cap, Amplify Damage on packs, Bone Armor up")))

# 4 Druid: Tornado/Hurricane (V Maxroll https://maxroll.gg/d2/guides/tornado-hurricane-druid)
build("druid_tornado_endgame90", "dru",
      "Skills: V end state Maxroll Tornado Hurricane Druid (fetched 2026-10-29): 1 each Oak Sage, Summon Spirit Wolf, Dire Wolf (Fenris), Grizzly; max Tornado, Hurricane, Cyclone Armor, Twister, Oak Sage. U no-respec adaptation: Raven 1 (Spirit Wolf prereq), Arctic Blast 6 as the early attack and Cyclone Armor prerequisite (the only wasted points), Twister as attack 18-23 (a maxed skill anyway), Tornado from 24, Hurricane from 30. Armageddon not taken. Stats: Str 43 by clvl 24 (Spirit sword), 55 by 55 (Enigma in a Mage Plate, V armor.txt), from internal/gear/milestones.json, rest Vitality.",
      [("Raven", 1), ("Summon Spirit Wolf", 1), ("Oak Sage", 1), ("Arctic Blast", 6), ("Cyclone Armor", 10), ("Twister", 6), ("Tornado", 20), ("Hurricane", 20), ("Summon Fenris", 1), ("Summon Grizzly", 1),
       ("Cyclone Armor", 20), ("Twister", 20), ("Oak Sage", 20), ("Summon Fenris", 20)],
      stages_for("druid_tornado_endgame90"),
      hk(("F1", "Arctic Blast", "right", 6), ("F2", "Cyclone Armor", "right", 12), ("F3", "Twister", "right", 18), ("F4", "Tornado", "right", 24), ("F5", "Hurricane", "right", 30), ("F6", "Oak Sage", "right", 6), ("F7", "Summon Grizzly", "right", 30), ("F8", "Summon Spirit Wolf", "right", 6)),
      ma((1, 5, "Raven", "any", "start: Raven plus staff/club melee"), (6, 17, "Arctic Blast", "any", "cold, mana hungry: potions"), (18, 23, "Twister", "any", ""),
         (24, 90, "Tornado", "any", "physical, random path, cast at the target and wait; Hurricane (cold) and Cyclone Armor as upkeep; summons recast")))

# 5 Assassin: Lightning Sentry (V Maxroll https://maxroll.gg/d2/guides/lightning-sentry-assassin; names are skills.txt internals: Fire Trauma = Fire Blast, Shock Field = Shock Web, Quickness = Burst of Speed)
build("assassin_sentry_endgame90", "ass",
      "Skills: V end state Maxroll Lightning Sentry Assassin (fetched 2026-10-29): max Lightning Sentry, Shock Web, Charged Bolt Sentry, Death Sentry, Fire Blast; 1 each Burst of Speed, Fade, Cloak of Shadows, Mind Blast, Weapon Block. Internal names Fire Trauma/Shock Field/Quickness are U for the mapping (S: skills.txt naming, matches existing assassin_fireblast_normal30 which uses Fire Trauma and Quickness). U ordering: Fire Blast 6 early attack (as the existing build), Charged Bolt Sentry from 12, Lightning Sentry from 24, Death Sentry from 30; Wake of Fire not taken. Stats: Str and Dex 57 by clvl 50 (Chaos in Quhab claws, V weapons.txt: claws need Dex equal to Str), Str 55 by 43 (Treachery in a Mage Plate, V armor.txt), from internal/gear/milestones.json, rest Vitality.",
      [("Fire Trauma", 6), ("Claw Mastery", 1), ("Quickness", 1), ("Shock Field", 1), ("Charged Bolt Sentry", 8), ("Psychic Hammer", 1), ("Cloak of Shadows", 1), ("Fade", 1), ("Weapon Block", 1),
       ("Lightning Sentry", 20), ("Mind Blast", 1), ("Shock Field", 20), ("Death Sentry", 20), ("Charged Bolt Sentry", 20), ("Fire Trauma", 20), ("Weapon Block", 20)],
      stages_for("assassin_sentry_endgame90"),
      hk(("F1", "Fire Trauma", "right", 2), ("F2", "Quickness", "right", 6), ("F3", "Charged Bolt Sentry", "right", 12), ("F4", "Lightning Sentry", "right", 24), ("F5", "Death Sentry", "right", 30), ("F6", "Fade", "right", 18), ("F7", "Cloak of Shadows", "right", 12)),
      ma((1, 11, "Fire Trauma", "any", "Fire Blast; melee Attack fallback"), (12, 23, "Charged Bolt Sentry", "any", "trap at the pack"), (24, 90, "Lightning Sentry", "any", "5 traps max; Death Sentry on corpses; Fire Blast on lightning-immune targets and bosses; merc finishes immunes")))

# 6 Barbarian: War Cry (V Maxroll https://maxroll.gg/d2/guides/war-cry-barbarian)
build("barbarian_warcry_endgame90", "bar",
      "Skills: V end state Maxroll War Cry Barbarian (fetched 2026-10-29): 1 each Find Item (Find Potion prereq), Grim Ward, Increased Speed, Iron Skin, Natural Resistance, Leap; max War Cry, Battle Cry, Taunt, Battle Orders, Howl. U no-respec adaptation: Bash 8 carries clvl 2-17 (existing barbarian_melee_normal30 used Double Swing/Frenzy, not taken here), Howl/Taunt/Shout and a few Battle Cry/Battle Orders points are prerequisites; Taunt/Howl/Battle Cry/Battle Orders (maxed endgame synergies) absorb the points until War Cry unlocks at 30 (V skills.txt reqlevel, needs Battle Cry + Battle Orders), so there is no bank. Stats: Str 43 by clvl 24 (Spirit swords), 77 by clvl 50 (Fortitude in a Dusk Shroud, V armor.txt; Archon Plate would need 103), from internal/gear/milestones.json, rest Vitality, no Energy.",
      [("Bash", 8), ("Howl", 1), ("Taunt", 1), ("Shout", 1), ("Iron Skin", 1), ("Battle Cry", 3), ("Battle Orders", 3), ("Find Potion", 1), ("Find Item", 1), ("Increased Stamina", 1), ("Increased Speed", 1), ("Leap", 1), ("Grim Ward", 1), ("Natural Resistance", 1),
       ("War Cry", 20), ("Battle Cry", 20), ("Battle Orders", 20), ("Taunt", 20), ("Howl", 20)],
      stages_for("barbarian_warcry_endgame90"),
      hk(("F1", "Bash", "right", 2), ("F2", "Shout", "right", 6), ("F3", "Iron Skin", "right", 18), ("F4", "Battle Orders", "right", 24), ("F5", "War Cry", "right", 30), ("F6", "Grim Ward", "right", 30)),
      ma((1, 29, "Bash", "any", "melee; Shout and Battle Orders buffs from 6/24"), (30, 90, "War Cry", "any", "stuns and damages in a radius around you: stand in the pack; very mana hungry: Insight merc, potions; Grim Ward on corpses for physical immunes")))

# 7 Amazon: Multiple Shot bow (V Maxroll https://maxroll.gg/d2/guides/multiple-shot-amazon)
build("amazon_multishot_endgame90", "ama",
      "Skills: V end state direction Maxroll Multiple Shot Amazon (fetched 2026-10-29): 1 each Dodge, Evade, Avoid, Valkyrie; max Pierce, Multiple Shot, Critical Strike, Penetrate, Guided Arrow, Magic Arrow, Strafe (Strafe for a D2R unique, not taken). U: Pierce 10 (the guide's 'until 69%' is D2R Buriza-less math, not recomputed), Guided Arrow from 18, Critical Strike/Penetrate 12 for attack rating (U). Valkyrie needs Decoy (Dopplezon) <- Slow Missiles <- Inner Sight and Evade <- Avoid <- Dodge (V skills.txt). Stats: Dex 152 and Str 108 (Grand Matron Bow, V data.json, clvl 58; Treachery in a Mage Plate needs 55, V), computed from internal/gear/milestones.json, staged for Ceremonial Bow (73/110, clvl 35) first, rest Vitality.",
      [("Magic Arrow", 1), ("Multiple Shot", 12), ("Critical Strike", 1), ("Penetrate", 1), ("Dodge", 1), ("Avoid", 1), ("Evade", 1), ("Inner Sight", 1), ("Slow Missiles", 1), ("Dopplezon", 1), ("Valkyrie", 1),
       ("Pierce", 10), ("Cold Arrow", 1), ("Guided Arrow", 12), ("Multiple Shot", 20), ("Critical Strike", 12), ("Penetrate", 12), ("Guided Arrow", 20), ("Magic Arrow", 10), ("Pierce", 20)],
      stages_for("amazon_multishot_endgame90"),
      hk(("F1", "Magic Arrow", "right", 2), ("F2", "Multiple Shot", "right", 6), ("F3", "Guided Arrow", "right", 18), ("F4", "Valkyrie", "right", 30), ("F5", "Dopplezon", "right", 24)),
      ma((1, 5, "Magic Arrow", "bow", "Short/Stag Bow + Arrows (docs/gear.md); Magic Arrow costs no ammo"), (6, 17, "Multiple Shot", "bow", "ammo restock chore"), (18, 90, "Multiple Shot", "bow", "Guided Arrow for single targets/bosses; Magic Arrow for physical immunes and ammo-free stragglers; Valkyrie upkeep")))
