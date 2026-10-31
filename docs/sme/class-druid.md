# SME class file: Druid (classic LoD 1.14d, offline single player, softcore)

Owner: SMEKB-B2. Companions: `builds-leveling.md`, `builds-endgame-casters.md`, `builds-gear.md`, `builds-open-questions.md`, `docs/classes/druid.md`. Feeds an unattended bot with no respec (`docs/leveler.md`).
Tags: **V** fetched or computed from the install's tables (URL or "T"), **V1** carried from first-pass SME files (Maxroll D2R guides, not re-fetched), **S** secondary, **U** memory or judgement. D2R-only facts are marked "D2R".

## 0. Verdict

The Druid is the weakest of the seven in the first 20 levels for gear and pool reasons (life +1.5 per clvl, mana only +2 per clvl, T) but has a very strong fire line with free-standing area damage: **Firestorm (4 mana at every level), Molten Boulder, Fissure (15 mana flat), Volcano**. The synergies are lavish: Firestorm gets +23% per Molten Boulder and Fissure level, Fissure +12% per Firestorm and Volcano level, Molten Boulder +8% per Firestorm level (V AB for Firestorm and Fissure; T Param8). The cost: **fire immune types are the largest immune group in Hell** (T: 83 fire-immune monster entries in `monstats.tsv` Hell, 54 across the per-act spawn lists) and mana (Fissure 15, Volcano 25, Molten Boulder 10-20) must be potion-fed. The Druid cannot cast Firestorm/Fissure while shape-shifted (V AB), so never put points in the were-forms for a bot.

## 1. Class basics (T `charstats.tsv`)

| item | value |
|---|---|
| Starting stats | Str 15, Dex 20, Int 20, Vit 25 |
| Life | +1.5 per clvl, +2 per Vit; Mana +2 per clvl, +2 per Energy; Stamina +1/+1 |
| Starting life/mana (U) | about 55 / 20 |
| Starting equipment | Club, Buckler, 4 minor healing potions, TP and ID scrolls (T codes clb, buc, hp1 x4, tsc, isc) |
| Block / AR | block 20, ToHitFactor 5 per Dex |

Points: 1 per clvl from 2 plus 4 per difficulty; 47 at clvl 40 on our timeline. Stat points 5 per clvl.

## 2. Skills, synergies, per-level numbers (T)

Damage columns are added damage per hit or per second per source (/s). Mana at skill level 1/10/20. Shapeshift entries show raw table values only.

| skill | clvl | prereqs | mana L1/L10/L20 | added dmg L1 / L10 / L20 (type) | synergy |
|---|---|---|---|---|---|
| Raven | 1 | - | 6/6/6 | - | - |
| Poison Creeper | 1 | - | 8/8/8 | 5-6 / 33-35 / 85-86 poison total (S) pois | - |
| Werewolf | 1 | - | 15/15/15 | - | - |
| Lycanthropy | 1 | Werewolf | - | - | - |
| Firestorm | 1 | - | 4/4/4 | 1-2 / 13-15 / 36-42 /sec per source fire | +23%/lvl: Molten Boulder, Fissure |
| Oak Sage | 6 | - | 15/24/34 | - | - |
| Summon Spirit Wolf | 6 | Raven | 15/15/15 | - | - |
| Werebear | 6 | - | 15/15/15 | - | - |
| Molten Boulder | 6 | Firestorm | 10/14.5/19.5 | 6-12 / 48-63 / 130-155 fire | +8%/lvl: Firestorm |
| Arctic Blast | 6 | - | 0.4/0.5/0.7 | 8-16 / 66-74 / 139-152 /sec per source cold | +15%/lvl: Cyclone Armor, Hurricane |
| Carrion Vine | 12 | Poison Creeper | 10/10/10 | - | - |
| Feral Rage | 12 | Werewolf | 3/3/3 | - | - |
| Maul | 12 | Werebear | 3/3/3 | - | - |
| Fissure | 12 | Molten Boulder | 15/15/15 | 15-25 / 81-91 / 217-227 fire | +12%/lvl: Firestorm, Volcano |
| Cyclone Armor | 12 | Arctic Blast | 5/14/24 | - | - |
| Heart of Wolverine | 18 | Oak Sage | 20/29/39 | - | - |
| Summon Dire Wolf | 18 | Oak Sage, Summon Spirit Wolf | 20/20/20 | - | - |
| Rabies | 18 | Feral Rage | 10/10/10 | poison, per-frame units (not resolved) | +18%/lvl: Poison Creeper |
| Fire Claws | 18 | Feral Rage, Maul | 4/4/4 | 15-20 / 95-100 / 247-260 fire | +22%/lvl: Firestorm, Molten Boulder, Volcano, Fissure |
| Twister | 18 | Cyclone Armor | 7/7/7 | - | +10%/lvl: Tornado, Hurricane |
| Solar Creeper | 24 | Carrion Vine | 14/23/33 | - | - |
| Hunger | 24 | Fire Claws | 3/3/3 | - | - |
| Shock Wave | 24 | Maul | 7/7/7 | - | +5%/lvl: Maul |
| Volcano | 24 | Fissure | 25/25/25 | 8-10 / 30-32 / 78-80 fire | +12%/lvl: Fissure, Armageddon |
| Tornado | 24 | Twister | 10/10/10 | - | +9%/lvl: Cyclone Armor, Twister, Hurricane |
| Spirit of Barbs | 30 | Heart of Wolverine | 25/34/44 | - | - |
| Summon Grizzly | 30 | Summon Dire Wolf | 40/40/40 | - | - |
| Fury | 30 | Rabies | 4/4/4 | - | - |
| Armageddon | 30 | Volcano, Hurricane | 35/35/35 | 25-75 / 170-231 / 390-471 fire | +14%/lvl: Firestorm, Molten Boulder, Volcano |
| Hurricane | 30 | Tornado | 30/30/30 | 25-50 / 94-119 / 202-227 cold | +9%/lvl: Twister, Tornado |

Formulas (V AB `https://theamazonbasin.com/wiki/index.php?title=<Skill>`):
- **Firestorm**: three piercing missiles leave fire trails (19 static fires each, V AB); +23% fire damage per Molten Boulder and Fissure level; delay 0.6 s; best at point-blank range; cast rate frames 14.
- **Fissure** (clvl 12): fire damage min 6*slvl+9 / 12*slvl-39 / 16*slvl-103 / 18*slvl-147 / 22*slvl-259 per crack for slvl 1-8 / 9-16 / 17-22 / 23-28 / 29+ (max 10 higher), 14 cracks per cast over 3.2 s, +12% per Firestorm and Volcano level, mana 15 flat, delay 2 s (V AB).
- **Cyclone Armor** (clvl 12, prereq Arctic Blast): absorbs 28+12*slvl elemental damage, +7% per Twister/Tornado/Hurricane level, mana 4+slvl (V AB). One point = 40 damage absorbed, recast needed.
- **Oak Sage** (clvl 6): party spirit giving +5*slvl% max life, mana 14+slvl, only one of the three spirits at a time (V AB). **Summon Dire Wolf** (clvl 18, prereq Oak Sage + Spirit Wolf): mana 20, count min(slvl,3), life 98-130 x (1 + 25+25*slvl %), +% defense (V AB). **Summon Spirit Wolf/Raven** are prerequisites (Raven 1 gives the chain).
- **Volcano**: lava eruptions (T 8-10 base at level 1), +12% per Fissure and Armageddon level; mana 25 flat.

## 3. Mana, potions, early game problems

| clvl | Vit points added | Energy points added | life (no gear) | mana (no gear) |
|---|---|---|---|---|
| 10 | 42 | 3 | 152 | 44 |
| 20 | 71 | 3 | 225 | 64 |
| 30 | 121 | 3 | 340 | 84 |
| 40 | 171 | 3 | 455 | 104 |

- Pool at clvl 20 is about 64 (T, 3 Energy points only): Fissure at 15 mana is 4 casts, Volcano at 25 is 2. Firestorm at 4 mana is the sustainable attack and gets +230-414% damage from the other fire skills by clvl 40 (T table), so **the bot should attack with Firestorm and cast Fissure/Volcano only when mana > 60%** (U). Mana potions and a mana belt column are mandatory (V1 "Druid ... Firestorm hits big monsters with all projectiles").
- Early game (clvl 1-11): Firestorm 4 mana, 3-6 damage per missile line at level 1 (T), Raven 1 (prereq for Spirit Wolf) and Oak Sage are life savers; the Druid has the fewest early defensive tools besides Cyclone Armor. Maxroll opens with Firestorm 1-5, Molten Boulder 6-7, Fissure at 12 (V1); our path matches until clvl 17.
- If potions run out: do not cast Volcano; stand in town, use wells; Cyclone Armor absorbs 40 at level 1 and fails after that (V AB).
- Stat plan: 3 Energy at clvl 2 for the first pool, Vit from clvl 3, 3 Str / 2 Vit between clvl 12 and 18 for gear (Leaf staff needs low Str), then Vit (U).

## 4. Leveling path, clvl 1-40, point by point

Assumptions: Den of Evil at clvl 5, Radament 15, Izual 24, NM Den/Radament/Izual at 30/34/38 (U). Machine-validated against `skills.tsv` (reqlevel and prerequisites OK, T). Final allocation at clvl 40: Firestorm 17, Fissure 12, Molten Boulder 6, Volcano 6, Arctic Blast 1, Cyclone Armor 1, Raven 1, Summon Spirit Wolf 1, Oak Sage 1, Summon Dire Wolf 1. The right column shows the four fire skills with mana and added damage including synergy.

| pt | at | skill point | stat points (5/clvl) | Firestorm / Molten Boulder / Fissure / Volcano |
|---|---|---|---|---|
| 1 | clvl 2 | Firestorm | 3 Ene, 2 Vit | Firestorm 1 mana 4.0 1-2/s |
| 2 | clvl 3 | Firestorm | 5 Vit | Firestorm 2 mana 4.0 2-4/s |
| 3 | clvl 4 | Firestorm | 5 Vit | Firestorm 3 mana 4.0 4-5/s |
| 4 | clvl 5 | Firestorm | 5 Vit | Firestorm 4 mana 4.0 5-6/s |
| 5 | Den of Evil N (clvl ~5) | Firestorm | - | Firestorm 5 mana 4.0 6-7/s |
| 6 | clvl 6 | Molten Boulder | 5 Vit | Firestorm 5 mana 4.0 7-9/s (+23% syn); Molten Boulder 1 mana 10.0 8-17 (+40% syn) |
| 7 | clvl 7 | Molten Boulder | 5 Vit | Firestorm 5 mana 4.0 9-10/s (+46% syn); Molten Boulder 2 mana 10.5 14-24 (+40% syn) |
| 8 | clvl 8 | Firestorm | 5 Vit | Firestorm 6 mana 4.0 10-12/s (+46% syn); Molten Boulder 2 mana 10.5 15-25 (+48% syn) |
| 9 | clvl 9 | Firestorm | 5 Vit | Firestorm 7 mana 4.0 12-14/s (+46% syn); Molten Boulder 2 mana 10.5 16-27 (+56% syn) |
| 10 | clvl 10 | Firestorm | 5 Vit | Firestorm 8 mana 4.0 14-15/s (+46% syn); Molten Boulder 2 mana 10.5 16-28 (+64% syn) |
| 11 | clvl 11 | Firestorm | 5 Vit | Firestorm 9 mana 4.0 17-19/s (+46% syn); Molten Boulder 2 mana 10.5 17-29 (+72% syn) |
| 12 | clvl 12 | Fissure | 3 Str, 2 Vit | Firestorm 9 mana 4.0 19-22/s (+69% syn); Molten Boulder 2 mana 10.5 17-29 (+72% syn); Fissure 1 mana 15.0 31-52 (+108% syn) |
| 13 | clvl 13 | Fissure | 3 Str, 2 Vit | Firestorm 9 mana 4.0 22-25/s (+92% syn); Molten Boulder 2 mana 10.5 17-29 (+72% syn); Fissure 2 mana 15.0 44-64 (+108% syn) |
| 14 | clvl 14 | Fissure | 3 Str, 2 Vit | Firestorm 9 mana 4.0 24-28/s (+115% syn); Molten Boulder 2 mana 10.5 17-29 (+72% syn); Fissure 3 mana 15.0 56-77 (+108% syn) |
| 15 | clvl 15 | Fissure | 3 Str, 2 Vit | Firestorm 9 mana 4.0 27-31/s (+138% syn); Molten Boulder 2 mana 10.5 17-29 (+72% syn); Fissure 4 mana 15.0 69-89 (+108% syn) |
| 16 | Radament N (clvl ~15) | Fissure | - | Firestorm 9 mana 4.0 30-34/s (+161% syn); Molten Boulder 2 mana 10.5 17-29 (+72% syn); Fissure 5 mana 15.0 81-102 (+108% syn) |
| 17 | clvl 16 | Fissure | 3 Str, 2 Vit | Firestorm 9 mana 4.0 32-37/s (+184% syn); Molten Boulder 2 mana 10.5 17-29 (+72% syn); Fissure 6 mana 15.0 94-114 (+108% syn) |
| 18 | clvl 17 | Arctic Blast | 3 Str, 2 Vit | Firestorm 9 mana 4.0 32-37/s (+184% syn); Molten Boulder 2 mana 10.5 17-29 (+72% syn); Fissure 6 mana 15.0 94-114 (+108% syn) |
| 19 | clvl 18 | Cyclone Armor | 3 Str, 2 Vit | Firestorm 9 mana 4.0 32-37/s (+184% syn); Molten Boulder 2 mana 10.5 17-29 (+72% syn); Fissure 6 mana 15.0 94-114 (+108% syn) |
| 20 | clvl 19 | Raven | 5 Vit | Firestorm 9 mana 4.0 32-37/s (+184% syn); Molten Boulder 2 mana 10.5 17-29 (+72% syn); Fissure 6 mana 15.0 94-114 (+108% syn) |
| 21 | clvl 20 | Summon Spirit Wolf | 5 Vit | Firestorm 9 mana 4.0 32-37/s (+184% syn); Molten Boulder 2 mana 10.5 17-29 (+72% syn); Fissure 6 mana 15.0 94-114 (+108% syn) |
| 22 | clvl 21 | Oak Sage | 5 Vit | Firestorm 9 mana 4.0 32-37/s (+184% syn); Molten Boulder 2 mana 10.5 17-29 (+72% syn); Fissure 6 mana 15.0 94-114 (+108% syn) |
| 23 | clvl 22 | Summon Dire Wolf | 5 Vit | Firestorm 9 mana 4.0 32-37/s (+184% syn); Molten Boulder 2 mana 10.5 17-29 (+72% syn); Fissure 6 mana 15.0 94-114 (+108% syn) |
| 24 | clvl 23 | Fissure | 5 Vit | Firestorm 9 mana 4.0 35-40/s (+207% syn); Molten Boulder 2 mana 10.5 17-29 (+72% syn); Fissure 7 mana 15.0 106-127 (+108% syn) |
| 25 | clvl 24 | Volcano | 5 Vit | Firestorm 9 mana 4.0 35-40/s (+207% syn); Molten Boulder 2 mana 10.5 17-29 (+72% syn); Fissure 7 mana 15.0 112-134 (+120% syn); Volcano 1 mana 25.0 15-18 (+84% syn) |
| 26 | Izual N (clvl ~24) | Fissure | - | Firestorm 9 mana 4.0 37-43/s (+230% syn); Molten Boulder 2 mana 10.5 17-29 (+72% syn); Fissure 8 mana 15.0 125-147 (+120% syn); Volcano 1 mana 25.0 16-20 (+96% syn) |
| 27 | Izual N 2 (clvl ~24) | Fissure | - | Firestorm 9 mana 4.0 40-46/s (+253% syn); Molten Boulder 2 mana 10.5 17-29 (+72% syn); Fissure 9 mana 15.0 152-174 (+120% syn); Volcano 1 mana 25.0 17-21 (+108% syn) |
| 28 | clvl 25 | Fissure | 5 Vit | Firestorm 9 mana 4.0 43-48/s (+276% syn); Molten Boulder 2 mana 10.5 17-29 (+72% syn); Fissure 10 mana 15.0 178-200 (+120% syn); Volcano 1 mana 25.0 18-22 (+120% syn) |
| 29 | clvl 26 | Molten Boulder | 5 Vit | Firestorm 9 mana 4.0 45-51/s (+299% syn); Molten Boulder 3 mana 11.0 24-38 (+72% syn); Fissure 10 mana 15.0 178-200 (+120% syn); Volcano 1 mana 25.0 18-22 (+120% syn) |
| 30 | clvl 27 | Molten Boulder | 5 Vit | Firestorm 9 mana 4.0 48-54/s (+322% syn); Molten Boulder 4 mana 11.5 31-46 (+72% syn); Fissure 10 mana 15.0 178-200 (+120% syn); Volcano 1 mana 25.0 18-22 (+120% syn) |
| 31 | clvl 28 | Molten Boulder | 5 Vit | Firestorm 9 mana 4.0 50-57/s (+345% syn); Molten Boulder 5 mana 12.0 38-55 (+72% syn); Fissure 10 mana 15.0 178-200 (+120% syn); Volcano 1 mana 25.0 18-22 (+120% syn) |
| 32 | clvl 29 | Molten Boulder | 5 Vit | Firestorm 9 mana 4.0 53-60/s (+368% syn); Molten Boulder 6 mana 12.5 45-64 (+72% syn); Fissure 10 mana 15.0 178-200 (+120% syn); Volcano 1 mana 25.0 18-22 (+120% syn) |
| 33 | clvl 30 | Volcano | 5 Vit | Firestorm 9 mana 4.0 53-60/s (+368% syn); Molten Boulder 6 mana 12.5 45-64 (+72% syn); Fissure 10 mana 15.0 188-211 (+132% syn); Volcano 2 mana 25.0 22-26 (+120% syn) |
| 34 | Den of Evil NM (clvl ~30) | Volcano | - | Firestorm 9 mana 4.0 53-60/s (+368% syn); Molten Boulder 6 mana 12.5 45-64 (+72% syn); Fissure 10 mana 15.0 198-222 (+144% syn); Volcano 3 mana 25.0 26-31 (+120% syn) |
| 35 | clvl 31 | Volcano | 5 Vit | Firestorm 9 mana 4.0 53-60/s (+368% syn); Molten Boulder 6 mana 12.5 45-64 (+72% syn); Fissure 10 mana 15.0 207-233 (+156% syn); Volcano 4 mana 25.0 31-35 (+120% syn) |
| 36 | clvl 32 | Volcano | 5 Vit | Firestorm 9 mana 4.0 53-60/s (+368% syn); Molten Boulder 6 mana 12.5 45-64 (+72% syn); Fissure 10 mana 15.0 217-244 (+168% syn); Volcano 5 mana 25.0 35-40 (+120% syn) |
| 37 | clvl 33 | Volcano | 5 Vit | Firestorm 9 mana 4.0 53-60/s (+368% syn); Molten Boulder 6 mana 12.5 45-64 (+72% syn); Fissure 10 mana 15.0 227-255 (+180% syn); Volcano 6 mana 25.0 40-44 (+120% syn) |
| 38 | clvl 34 | Fissure | 5 Vit | Firestorm 9 mana 4.0 56-63/s (+391% syn); Molten Boulder 6 mana 12.5 45-64 (+72% syn); Fissure 11 mana 15.0 260-288 (+180% syn); Volcano 6 mana 25.0 42-46 (+132% syn) |
| 39 | Radament NM (clvl ~34) | Fissure | - | Firestorm 9 mana 4.0 58-66/s (+414% syn); Molten Boulder 6 mana 12.5 45-64 (+72% syn); Fissure 12 mana 15.0 294-322 (+180% syn); Volcano 6 mana 25.0 44-49 (+144% syn) |
| 40 | clvl 35 | Firestorm | 5 Vit | Firestorm 10 mana 4.0 68-78/s (+414% syn); Molten Boulder 6 mana 12.5 47-67 (+80% syn); Fissure 12 mana 15.0 307-336 (+192% syn); Volcano 6 mana 25.0 44-49 (+144% syn) |
| 41 | clvl 36 | Firestorm | 5 Vit | Firestorm 11 mana 4.0 78-90/s (+414% syn); Molten Boulder 6 mana 12.5 49-70 (+88% syn); Fissure 12 mana 15.0 319-350 (+204% syn); Volcano 6 mana 25.0 44-49 (+144% syn) |
| 42 | clvl 37 | Firestorm | 5 Vit | Firestorm 12 mana 4.0 88-102/s (+414% syn); Molten Boulder 6 mana 12.5 51-73 (+96% syn); Fissure 12 mana 15.0 332-363 (+216% syn); Volcano 6 mana 25.0 44-49 (+144% syn) |
| 43 | clvl 38 | Firestorm | 5 Vit | Firestorm 13 mana 4.0 98-114/s (+414% syn); Molten Boulder 6 mana 12.5 53-75 (+104% syn); Fissure 12 mana 15.0 344-377 (+228% syn); Volcano 6 mana 25.0 44-49 (+144% syn) |
| 44 | Izual NM (clvl ~38) | Firestorm | - | Firestorm 14 mana 4.0 108-126/s (+414% syn); Molten Boulder 6 mana 12.5 55-78 (+112% syn); Fissure 12 mana 15.0 357-391 (+240% syn); Volcano 6 mana 25.0 44-49 (+144% syn) |
| 45 | Izual NM 2 (clvl ~38) | Firestorm | - | Firestorm 15 mana 4.0 118-139/s (+414% syn); Molten Boulder 6 mana 12.5 57-81 (+120% syn); Fissure 12 mana 15.0 370-405 (+252% syn); Volcano 6 mana 25.0 44-49 (+144% syn) |
| 46 | clvl 39 | Firestorm | 5 Vit | Firestorm 16 mana 4.0 128-151/s (+414% syn); Molten Boulder 6 mana 12.5 59-84 (+128% syn); Fissure 12 mana 15.0 382-419 (+264% syn); Volcano 6 mana 25.0 44-49 (+144% syn) |
| 47 | clvl 40 | Firestorm | 5 Vit | Firestorm 17 mana 4.0 143-167/s (+414% syn); Molten Boulder 6 mana 12.5 61-87 (+136% syn); Fissure 12 mana 15.0 395-432 (+276% syn); Volcano 6 mana 25.0 44-49 (+144% syn) |

Why: Firestorm 9 by clvl 11 (T: 18-21 /s per source at level 9, +92% synergy once Fissure 2 exists); Molten Boulder 2 is only a prerequisite (6 by clvl 36 for the synergy); Fissure 6 by clvl 17, Fissure 12 by 40; Arctic Blast 1 + Cyclone Armor 1 for 40 absorbed elemental damage; Raven 1 + Spirit Wolf 1 + Oak Sage 1 + Dire Wolf 1 give the bot a tank pet (V AB: Dire Wolf life 98-130 +50% at level 1, only 1 wolf at level 1); Volcano from clvl 24 (T prereq) and 6 by 40; remaining points back into Firestorm 17. Human version (V1 Maxroll): Firestorm 1-5, Molten Boulder 6-7, Firestorm 8-11, Fissure 12-26, Firestorm/Volcano later, and a respec before Mephisto in NM into Tornado/Hurricane/Cyclone Armor/Grizzly.

## 5. Mercenary

- Act 2 Desert merc (V AS/T): Prayer / Thorns / Prayer (Combat), Defiance / Holy Freeze / Defiance (Defense), Blessed Aim / Might / Blessed Aim (Offense) by difficulty. Auras apply to the Druid's pets too (V AS "share their auras with ... Minions"). Best bot pick: Defiance in Normal and Hell, Holy Freeze (slow) or Might in NM (U). A merc is the Druid's main tank early.
- Act 3 Iron Wolf casters are not recommended (fragile). Act 5 Barbarian has the most life (T).
- No Infinity offline (ladder, V `builds-open-questions.md` Q3).

## 6. Gear per act on a budget

| stage | weapon | armor / helm | rest | notes |
|---|---|---|---|---|
| Normal Act 1-2 | start Club; Leaf (Tir+Ral staff, 2 sockets: +3 fire skills, +3 Fire Bolt/Warmth/Inferno are sorceress-only, +16 defense per level, 33% cold res, T) as soon as Ral is available (Act 5 quest V Q1) or from Countess drops (V1) | Stealth (Tal+Eth) in 2os armor at clvl 17 (V1); Lore (Ort+Sol) | belt from Fara | Druid pelts (+skills) from drops (U) |
| Normal Act 3-5 | Leaf or a magic staff with +fire skills | druid pelt, Smoke (Nef+Lum) | rings with FCR | Maxroll lists Leaf as the Normal staff (V1) |
| Nightmare | Heart of the Oak (Ko+Vex+Pul+Thul staff/mace, +3 skills, FCR 40, T) | Chains of Honor (+2 skills, +65 res with Um, V Q2), Stone for FHR (T) | resist jewels | raw res 60 each |
| Hell | Heart of the Oak, Memory is sorceress-oriented (+3 sorc skills, V Q2) | Enigma (+2 skills, Teleport) | Harlequin Crest | raw res 125 each |

## 7. Resists and immunities

Targets: Normal 0 needed; Nightmare entry raw 60 fire/cold/lightning (+20 after -40); Hell raw 125 (+25 after -100), cap 175 (V AS).

Immunity landscape (T; NM / Hell types):
| act | distinct spawn types | physical imm NM / Hell | fire imm NM / Hell | cold imm NM / Hell | lightning imm NM / Hell |
|---|---|---|---|---|---|
| 1 | NM 55 / Hell 55 | 0 / 2 | 0 / 16 | 0 / 11 | 0 / 14 |
| 2 | NM 57 / Hell 57 | 0 / 4 | 1 / 13 | 0 / 13 | 0 / 11 |
| 3 | NM 57 / Hell 57 | 0 / 2 | 0 / 10 | 0 / 17 | 0 / 15 |
| 4 | NM 20 / Hell 21 | 0 / 0 | 1 / 4 | 0 / 4 | 1 / 8 |
| 5 | NM 158 / Hell 47 | 0 / 4 | 2 / 11 | 10 / 21 | 4 / 8 |

- Hell fire-immune spawn types by act:
  - Act 1: Fallen, Fallen Shaman, Carver, Vile Lancer, Carver Shaman, Vile Archer, Returned Mage, Devilkin, Blood Clan, Devilkin Shaman, Bone Mage, Dark One, Dark Shaman, Rat Man, Arach, Ghoul
  - Act 2: Sand Leaper, Saber Cat, Tomb Creeper, Guardian, Hell Slinger, Burning Dead, Sand Raider, Burning Dead Mage, Invader, Blunderbore, Horror Mage, Desert Wing, Hell Clan
  - Act 3: Swamp Dweller, Drowned Carcass, Fetish Shaman, Flayer Shaman, Soul Killer Shaman, Hell Buzzard, Zakarumite, Flame Spider, Spider Magus, Blood Diver
  - Act 4: Doom Knight, Balrog, Pit Lord, Venom Lord
  - Act 5: Demon Rascal, Moon Lord, Demon Trickster, Succubus, Talic, Death Lord, Vile Witch, Greater Hell Spawn, Arach, Pit Lord

How humans handle it: the **fire** Druid is exactly the class hurt most by immunity, because fire immune types are the largest group in Hell (T above: 16 in Act 1, 13 in Act 2, 10 in Act 3, 4 in Act 4, 11 in Act 5 of 47-57 each), so humans swap to **Hurricane/Tornado/Arctic Blast** (cold, T Hurricane and Arctic Blast rows) or **Armageddon** (fire again) or a wolf/summon druid (physical). Our path has no second element: it can only keep the Dire Wolf and merc. **Not recommended in Hell without a respec** (V1 Maxroll respecs into Tornado/Hurricane at clvl 38; U on whether Tornado is physical).

## 8. Strengths, weaknesses, deaths by act/difficulty

- Normal Act 1-2 (clvl 1-24): Firestorm is excellent from level 1; weak early life and mana; Fissure from clvl 12 is the first real clear. Deaths: Blood Raven's arrows, Andariel, Duriel, wasp swarms in Act 2 jungle (U).
- Normal Act 3-5: the big problem is melee packs while mana is empty; summons/merc tank; Baal waves with fire immunes begin only in Hell (T: Normal has 1 fire-immune type, the Fire Golem).
- Nightmare: fire immunes are rare (T: 0 in Act 1, 1 each in Act 2 and 4, 2 in Act 5); the -40 resists and 340-455 life (T) are the danger; Mephisto, Izual and Diablo are the usual deaths (U).
- Hell: fire-immune packs everywhere (T), -100 resists (V AS), little life; not viable as a bot without a second element.
- Typical bot deaths (U, predicted): out of mana mid-pack; fire immune packs in Hell; standing in melee after the wolf/merc died; Fissure cast delay 2 s (V AB) with a pack on top.

## 9. Three common builds and their trade-offs

| build | skills | pros | cons | no-respec fit |
|---|---|---|---|---|
| Fire (Firestorm/Fissure/Volcano; this file) | Firestorm, Molten Boulder, Fissure, Volcano, Armageddon | great synergy, cheap Firestorm, no weapon needed | fire immunes (T: 54 spawn types across Acts, 83 monster entries in Hell), mana, low life | best to NM |
| Tornado / Hurricane / Cyclone Armor | Arctic Blast 1, Cyclone Armor, Twister, Tornado c24, Hurricane c30 | strong, immune coverage differs (V1 Maxroll tornado-hurricane) | Twister prereq points, needs gear, bad early | respec build (V1) |
| Summoner (Oak Sage, Spirit Wolf, Dire Wolf, Grizzly, Raven, vines) | Summon skills + Oak Sage/Heart of Wolverine | safest, immunity-robust like the Necromancer summoner | slow, pets die, boss damage cut for summons (V AS 50/35/25%) | possible, 12+ points of setup |

## 10. When is the character "ready" (bot checks; U unless tagged)

| step | minimum | extra checks |
|---|---|---|
| Den of Evil / Blood Raven | clvl 6-8, Firestorm 5+, 4 health + 3 mana potions | Raven 1 not required |
| Andariel | clvl 16-18, Fissure 5+, life >= 190, 4 mana + 4 health potions, merc hired | Cyclone Armor cast before the fight |
| Duriel / Act 3 | clvl 21-23, life >= 230, cold res >= 20, pool >= 60 | Dire Wolf 1 |
| Mephisto | clvl 26-28, life >= 270, Fissure 8+, all res >= 40 | Volcano 1 |
| Diablo / Baal | clvl 30-34, life >= 330-380, Fissure 10+ | all res >= 50-60 |
| Enter Nightmare | clvl >= 38-40, raw res >= 60, life >= 440 (T path 455 at clvl 40), Firestorm 17, Fissure 12 | 6 mana potions, merc alive |
| Enter Hell | clvl >= 64 (project gate V), raw res >= 125, life >= 900, a second element or skip plan for fire-immune packs (T: 39 types in Acts 1-3 spawn lists) | not ready on this path without respec |

## 11. Open questions

1. Tornado/Hurricane damage types in 1.14d (the table shows Hurricane 94-119 base cold; Tornado column empty: physical? U).
2. Whether the Dire Wolf survives Hell packs and how Oak Sage's life bonus stacks with merc auras (U).
3. Starting life/mana for all classes are U.

## Shared facts (apply to every class; V = Arreat Summit archive, still served by Blizzard)

- Resist floors by difficulty: Normal 0, Nightmare -40, Hell -100; absolute floor -100; default cap 75 (V `https://classic.battle.net/diablo2exp/basics/resistances.shtml`). Hell therefore needs raw 175 to cap at 75, raw 125 for +25.
- Nightmare: unique/champion damage +75% (Normal +90%, Hell +66%), summons and hirelings deal only 35% damage to act bosses (Normal 50%, Hell 25%), life/mana steal halved (Hell cut by 2/3), freeze/chill/curses halved (Hell cut by 3/4), Static Field cannot take a monster below 33% (NM) / 50% (Hell), monster skill levels +3 (NM) / +7 (Hell), death costs 5% (NM) / 10% (Hell) of the exp gap to the next level, 75% of it restored when the corpse is picked up (V `https://classic.battle.net/diablo2exp/basics/difficulty.shtml`).
- Mercenary aura table, identical in the Arreat page and our tables (V `https://classic.battle.net/diablo2exp/basics/hirelings.shtml`, `hireling.tsv` of this install): Act 2 Desert mercenary Combat = Prayer (Normal) / Thorns (NM) / Prayer (Hell); Defense = Defiance / Holy Freeze / Defiance; Offense = Blessed Aim / Might / Blessed Aim. So Might is an NM-only option; in Hell the Act 2 offense merc runs Blessed Aim. Act 1 Rogue: Fire or Cold Arrow plus Inner Sight (bow). Act 3 Iron Wolf: Fire (Inferno/Fire Ball), Cold (Glacial Spike/Frozen Armor/Ice Blast) or Lightning (Charged Bolt/Lightning), sword + shield, "rarely swing". Act 5 Barbarian: Bash and Stun, one sword. No Act 4 merc. Revival never costs more than 50,000 gold; if the player dies the merc dies too.
- "T" below means a value computed from this install's own game tables (`skills.tsv`, `charstats.tsv`, `hireling.tsv`, `monstats.tsv`, `levels.tsv`, `runes.tsv`; the worker host `~/.local/share/d2bot/cache/data/v2-6ff77ee7fff5/`), V by construction. Skill damage/mana formulas: mana = (mana + lvlmana*(lvl-1)) * 2^manashift / 256 with a floor of minmana; added elemental damage = (EMin + sum of the per-level steps for levels 2-8, 9-16, 17-22, 23-28, 29+) * 2^HitShift / 256, times (1 + synergy%). Cross-checked against Amazon Basin pages (Exploding Arrow mana 4.5+0.5/lvl, Blessed Hammer 4.75+0.25/lvl, Firestorm +23% per Molten Boulder/Fissure level, Bone Spear +7% per Teeth/Bone Wall/Prison/Spirit level, `https://theamazonbasin.com/wiki/index.php?title=<Skill>`). Skills with HitShift 2-4 (Firestorm, Inferno, Fire Wall, Blaze, Arctic Blast, Wake of Inferno) are damage per second per source (x25 frames); poison skills show the total over their duration where resolvable (S).
- Immunity tables are built from `levels.tsv` spawn lists (mon/nmon/umon columns) joined to `monstats.tsv` resists (>=100 means immune); they count distinct regular spawn types per act and difficulty, excluding bosses/super uniques (T). Immunities on unique and champion monsters are rolled on top (enchants and auras), so these are lower bounds for "something nasty is immune". The Act 5 Nightmare spawn lists in `levels.tsv` are unusually long (158 types), so read that row as a rough shape.
- Tag "V1" = taken from the first-pass SME files (`builds-leveling.md`, `builds-gear.md`, `builds-endgame-*.md`), where it was fetched from Maxroll D2R Season 14 guides (D2R text; mechanics match 1.14d except where flagged). I did not re-fetch those pages. Ladder-only runewords (Spirit, Insight, Fortitude, Infinity, Dragon, Dream, Edge, Faith, Harmony, Ice, Brand, Death, Destruction, Last Wish, Lawbringer, Oath, Obedience, Phoenix, Pride, Rift, Grief, Voice of Reason, Wrath) are not craftable in vanilla offline 1.14d (V `builds-open-questions.md` Q3); Enigma, Chains of Honor, Stone, Memory, Delirium, Lore, Rhyme, Smoke, Stealth, Heart of the Oak, Beast, Fury, Honor, Doom, Hand of Justice, Ancient's Pledge, Bone, Peace, Treachery, Duress, Gloom are available (T: server flag empty).

