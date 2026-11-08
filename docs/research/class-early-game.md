# Per-class early game (veteran estimate)

> status: veteran estimate, not measured. Date 2026-11-07. Source: the d2-sme veteran report below, copied unchanged. Tags in the body: SRC (repo file read), EXP (veteran experience), UNSURE (not verified). Nothing here is V (verified live) except the repo facts the SME read from `internal/plandata`. The measured replacement is card `classpacing`.

## How the repo uses this

- Principle (user, 2026-11-07, decisions F10): world-generic first, class overlay second. Corpsefire at the Den, waypoint milestones and the Act 1 firsts are the same for every class; unlock schedules, chore defaults, charm tiers, pacing and first-skill tables are per-class DATA, never Go literals.
- Default test class is the Paladin (decisions F11); the first-run wizard offers the classes that have a normal30 leveling table.
- Section 2 feeds the per-class unlock files (cards `classunlocks`, `skillunlocks`); section 4 the policy-sheet chore defaults (`classchores`); section 8 the charm tiers (`classcharm`, legacycharm); section 5 the recommended-level table until measured (`classpacing`, `proposezone`); the gap in the repo facts is card `sorcnormal30`.
- Skill clvl requirements in section 2 stay UNVERIFIED until `skillreqcheck` has checked them against the game tables.

---


Source tags: SRC = repo file read by the SME; EXP = veteran experience; UNSURE = not verified. No web lookups. Skill tables and times are EXP.

Verdict: a single "Corpsefire at the Den, waypoint milestones, firsts" world is safe for all seven classes. Unlock schedules, mana/potion policy, charms, pacing and chores must be per-class data.

## Repo facts (SRC internal/plandata)
- normal30 leveling tables exist for: Amazon (bow, javelin), Assassin (fireblast), Barbarian (melee), Druid (fire), Necromancer (summoner, leveling), Paladin (normal30, v2, zealot-lite).
- Sorceress has NO normal30 table: only build-sorceress_blizzard_endgame90, build-primary-blizzard, and the fallback-lightning files (v1 to v3). Gap for this design.
- Endgame90 only (no early table of their own): hammerdin (+zeal variant), multishot, tornado, sentry, warcry, necro summoner.
- main_attack per clvl in the repo: Amazon bow: Throw 1-2, Fire Arrow 3-11, Exploding Arrow 12+. Amazon javelin: Throw 1-5, Poison Javelin 6-13, Lightning Bolt 14+. Assassin: Fire Blast 1-11, Wake of Fire sentry 12+. Barbarian: Bash 1-5, Double Swing 6-23, Frenzy 24+. Druid: Firestorm 1-11, Eruption (Fissure) 12+. Necromancer: Raise Skeleton 1-14, Corpse Explosion 15+.

## 1. Early build and main skill (clvl 1 / 6 / 12 / 18) EXP
- Amazon (primary javelin, idle friendly): Throw / Poison Javelin / Poison Javelin / Lightning Bolt (18). Alt: bow Fire Arrow then Exploding Arrow (mana-hungry, needs arrows). Caveat: javelin self-ammo pickup loop is fragile (UNSURE how well the bot handles it).
- Assassin (Fire Blast to Wake of Fire sentry): Fire Blast / Fire Blast / Wake of Fire / Wake of Fire. Alt martial arts charge-up needs micro.
- Barbarian (melee): Bash / Double Swing / Double Swing / Double Swing or Frenzy. Gear matters most here.
- Druid (fire): Firestorm / Firestorm / Eruption (Fissure) / Eruption + Molten Boulder. Plain Werebear/Werewolf is NOT idle-safe (shapeshift breaks potion and skill hotkeys).
- Necromancer (summoner): Raise Skeleton + Teeth / Skeleton + Clay Golem / + Skeleton Mastery / Corpse Explosion with Raise Skeleton.
- Paladin: Sacrifice or Attack / Zeal (6-12) / Zeal + Holy Fire aura / Zeal + Holy Fire (Might early). Vengeance later.
- Sorceress for comparison: Fire Bolt / Ice Bolt / Fire Ball or Glacial Spike / Fire Ball.

## 2. Unlock schedule by quest or find (EXP; gating points themselves is not possible, only what the bot may spend them on)
Rule: nothing the bot needs to survive locks later than Bishibosh. Must stay free for all classes: the main attack at its clvl, one defensive buff (Frozen Armor, Bone Armor, Oak Sage, Cloak/Cyclone, Battle Orders, Prayer or Might), and a skill for immunities (Molten Boulder, Fire Arrow).
- Amazon: free clvl 1 Throw/Jab, Magic Arrow, Inner Sight (optional); clvl 6 Poison Javelin, Fire Arrow; Den: Slow Missiles or Critical Strike; Bishibosh: Decoy/Valkyrie may stay locked; Blood Raven: Penetrate/Pierce, Valkyrie; Cain rescued: Lightning Bolt, Fend; Andariel: Lightning Fury, Exploding Arrow, Multiple Shot.
- Assassin: free clvl 1 Fire Blast, Tiger Strike; clvl 6 Psychic Hammer, Cloak of Shadows; Den: Burst of Speed; Bishibosh: Dragon Talon; Blood Raven: Wake of Fire; Andariel: Lightning Sentry, Death Sentry, Shadow Warrior.
- Barbarian: free clvl 1 Bash, Sword Mastery, Howl (may stay locked); clvl 6 Double Swing, Stun; Den: Natural Resistance or Iron Skin; Bishibosh: Taunt, Shout (Shout must stay free); Blood Raven: Battle Orders, Find Item; Andariel: Whirlwind, Frenzy, Berserk.
- Druid: free clvl 1 Firestorm, Raven, Poison Creeper, Werewolf; clvl 6 Molten Boulder, Oak Sage (must stay free); Den: Cyclone Armor; Bishibosh: Fissure; Blood Raven: Spirit wolves/Hunger; Andariel: Tornado, Hurricane, Volcano.
- Necromancer: free clvl 1 Raise Skeleton, Teeth, Bone Armor; clvl 6 Clay Golem, Amplify Damage; Den: Skeleton Mastery; Bishibosh: Corpse Explosion, Poison Dagger; Blood Raven: Iron Maiden, Bone Wall; Andariel: Revive, Bone Spear, Raise Skeletal Mage.
- Paladin: free clvl 1 Sacrifice, Smite, Might (survival aura); clvl 6 Holy Fire, Zeal, Charge; Den: Resist Fire/Lightning; Bishibosh: Prayer or Cleansing (Prayer must stay free); Blood Raven: Vengeance, Holy Shield; Andariel: Blessed Hammer, Fist of the Heavens.
- Sorceress (earlier schedule): free Fire Bolt, Charged Bolt; clvl 6 Ice Bolt, Frozen Armor; Den: Frost Nova; Bishibosh/Cold Plains waypoint: Static Field; Blood Raven: Warmth, Telekinesis; Countess: Ice Blast, Fire Wall, Blaze; Cain: Inferno, Fire Ball; Andariel: Teleport (not needed to reach Andariel), Glacial Spike, Nova.

## 3. Corpsefire, Bishibosh, Blood Raven (EXP)
- Den of Evil: everyone goes at clvl 5-6; shut-in cave, 15-25 monsters plus Corpsefire; a clvl 5-6 character with 2-3 skill points and about 8 potions can do it.
- Corpsefire (fire; resist fire matters): Amazon javelin/bow from range, low risk. Assassin Fire Blast ranged. Barbarian melee tank, low to moderate (fire ring dangerous with no fire resist, UNSURE numbers). Druid Firestorm does little, Raven/Creeper help, moderate. Necromancer skeleton army + Clay Golem, lowest risk. Paladin Zeal + Holy Fire, lowest risk. Sorceress highest risk because of mana.
- Bishibosh (Cold Plains, small fast pack): melee and summoners fine at clvl 7-8; Sorceress struggles if mana short; Amazon/Assassin kite well; Druid moderate.
- Blood Raven (Burial Grounds, clvl 8-10): ranged and summoners handle her well; Barbarian/Paladin kill her in melee but her fire arrows hurt; Druid and Sorceress need mana or pets; escorts are reviving skeletons: Necromancer, Amazon, Assassin best.
- Verdict: Corpsefire as a universal early goal is sensible; tactic differs per class.

## 4. Chores per class (EXP)
- Potions: Sorceress, Druid, Amazon-bow, Necromancer, Assassin use mana (about 1:1 mana:health at start); Barbarian no mana potions; Paladin mostly health (about 4:1).
- Repair: Barbarian and Paladin most often (every trip); Assassin (claws) and Necromancer rarely; casters almost never.
- Ammo: Amazon bow needs arrows (quiver 250-350, restock at 20-30%); javelins picked up from the floor; only Amazon has an ammo trip trigger.
- Summons upkeep: Necromancer recasts skeletons and golem after every pack (mana cost significant clvl 6-12); Druid Raven/Creeper/wolves similar but cheaper; Assassin traps lay then fire.
- Auras and buffs: Paladin Might while farming, Holy Fire against a pack, Prayer when hurt, Resist Fire against Corpsefire; Barbarian Battle Orders and Shout refreshed per fight; Druid Oak Sage, Cyclone Armor, Raven.
- Town-trip triggers: Barbarian/Paladin health potions below 3 or durability below 25%; Amazon ammo below 20% or potions below 3; Necromancer/Druid/Sorceress mana potions below 3; all: belt below half, inventory full, TP tome below 1.

## 5. Pacing (UNSURE, veteran estimate, not measured)
- Project fact: the Sorceress is mana-bound, clvl 3-4 in about 49 minutes (measured, TASK-138).
- Fast (no mana bound): Barbarian, Paladin (Zeal), Amazon-javelin: clvl 5-6 in about 20 minutes, Den done by about 35-40 minutes, 1.5x to 2x faster than the Sorceress in the first hour. Medium: Assassin, Necromancer (slow start, faster from clvl 6), Druid. The Sorceress is the slowest.
- Minutes by zone for a fast class (Sorceress in brackets): Blood Moor 8 (10); Den 12 (15); Cold Plains 8 (10); Burial Grounds/Blood Raven 10 (12); Stony Field 8 (10); Caves 12 (15); Dark Wood 10 (12); Black Marsh + Forgotten Tower 16 (20); Tamoe 6 (8); Monastery to Catacombs 4 30 (35); Andariel 10 (10). Modifiers: Barbarian and Paladin -20%; Amazon-javelin -15%; Necromancer +10% until clvl 6 then -10%; Assassin and Druid about the Sorceress.

## 6. Watchable moments in the first 30 minutes (EXP)
Amazon javelin volleys and Poison Javelin clouds; Assassin Fire Blast bursts, Tiger Strike chains, first sentry; Barbarian Bash stun, Double Swing, dropping a pack in melee; Druid Firestorm fire carpets, Raven swarms, Molten Boulder; Necromancer first skeleton army and Clay Golem; Paladin Holy Fire aura pulse and Zeal chains.

## 7. Risks (EXP)
A missing first-skill table is a risk for any class, and the Sorceress has none in plandata. Druid first skill pick must be fixed in data (the table pins Firestorm); Amazon needs a bow-vs-javelin choice (two tables); Barbarian weapon type and gear matter most; Sorceress and Paladin are the least ambiguous.

## 8. Legacy charm per class (EXP)
The same tiers do NOT suit every class: Barbarian and Paladin do not need +mana (replace with +life, +resists, damage/defense); Amazon-bow, Necromancer, Druid and Sorceress need mana. Tier I with 2 Minor Healing suits everyone. Recommendation: charm tiers as per-class data (life, mana, MF, resists, potion); melee gets mana 0.

## 9. Safe vs per-class
SAFE for all classes: Corpsefire as the early goal (clvl 5-6, tactic per class), waypoint milestones (Cold Plains, Stony Field, Dark Wood), firsts (Den, Bishibosh, Blood Raven, Countess, Andariel), tally on events, "nothing the bot needs to survive locks later than Bishibosh".
MUST be per-class DATA: the skill unlock schedule and must-stay-free list, mana vs health potion mix, repair and ammo triggers, aura and buff choice, charm tiers, pacing by zone, watchable moments, the first-skill table.

## Gaps noted by the SME
docs/sme/ has class-*.md and builds-leveling.md but the SME did not read them; no per-class early-game timing table exists: measured data should be queued.
