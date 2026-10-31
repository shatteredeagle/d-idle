# SME class file: Sorceress (classic LoD 1.14d, offline single player, softcore)

Owner: SMEKB-B2. Companions: `builds-leveling.md`, `builds-endgame-casters.md`, `builds-gear.md`, `builds-open-questions.md`, `docs/classes/ranking.md`. Feeds an unattended bot with no respec (`docs/leveler.md`), so section 4 is a no-respec line.
Tags: **V** fetched or computed from the install's tables (URL or "T"), **V1** carried from first-pass SME files (Maxroll D2R guides, not re-fetched), **S** secondary/snippet, **U** memory or judgement. D2R-only facts are marked "D2R".

## 0. Verdict

The Sorceress is the fastest class to level and the most gear-dependent at the end. Her life is the lowest in the game (+1 per clvl, +2 per Vit point, T), so everything depends on killing before being touched: Teleport (1 point at clvl 18), Frozen Armor, Static Field and a merc. For the bot, a **cold line (Ice Bolt -> Ice Blast -> Glacial Spike -> Blizzard -> Frozen Orb)** is the best no-respec choice: every point feeds the later skills through Ice Bolt synergy (15% per level to Ice Blast, Glacial Spike, Blizzard, Frozen Orb, Frost Nova, T), nothing is wasted, and cold immunity is rare before Hell (T: 0 cold-immune regular types in Normal and in Acts 1-4 of Nightmare). Mana is the binding constraint (section 3).

## 1. Class basics (T `charstats.tsv`)

| item | value |
|---|---|
| Starting stats | Str 10, Dex 25, Int 35, Vit 10 |
| Life | +1 per clvl, +2 per Vit point; Mana +2 per clvl, +2 per Energy point; Stamina +1/clvl, +1/Vit |
| Starting life/mana (U, not in the extract) | about 40 / 35 |
| Start skill | Fire Bolt (T `StartSkill`) |
| Starting equipment | Short Staff, 4 minor healing potions, TP and ID scrolls (T codes sst, hp1 x4, tsc, isc) |
| Block / AR | block 20; ToHitFactor -15 (never melee) |

Skill points: 1 per clvl from 2 plus 4 per difficulty (Den +1, Radament +1, Izual +2, V1); 47 at clvl 40 on our timeline. Stat points 5 per clvl (V1).

## 2. Skills, synergies, per-level numbers (T)

Added damage per hit unless marked /s (per second per source). Mana at skill level 1/10/20.

| skill | clvl | prereqs | mana L1/L10/L20 | added dmg L1 / L10 / L20 (type) | synergy |
|---|---|---|---|---|---|
| Fire Bolt | 1 | - | 2.5/2.5/2.5 | 3-6 / 18-22 / 46-60 fire | +16%/lvl: Fire Ball, Meteor |
| Warmth | 1 | - | - | - | - |
| Charged Bolt | 1 | - | 3/7.5/12.5 | 2-4 / 6-8 / 14-16 ltng | +6%/lvl: Lightning |
| Ice Bolt | 1 | - | 3/3/3 | 3-5 / 14-20 / 38-50 cold | +15%/lvl: Frost Nova, Ice Blast, Glacial Spike, Blizzard, Frozen Orb |
| Frozen Armor | 1 | - | 7/7/7 | - | - |
| Inferno | 6 | - | 0.6/0.7/0.9 | 12-25 / 98-112 / 203-220 /sec per source fire | +13%/lvl: Warmth |
| Static Field | 6 | - | 9/9/9 | - | - |
| Telekinesis | 6 | - | 7/7/7 | 1-2 / 10-11 / 20-21 ltng | - |
| Frost Nova | 6 | - | 9/18/28 | 2-4 / 22-28 / 56-68 cold | +10%/lvl: Blizzard, Frozen Orb |
| Ice Blast | 6 | Ice Bolt | 6/10.5/15.5 | 8-12 / 85-94 / 253-266 cold | +8%/lvl: Ice Bolt, Blizzard, Frozen Orb |
| Blaze | 12 | Inferno | 11/15.5/20.5 | 6-12 / 38-44 / 91-97 /sec per source fire | +4%/lvl: Warmth, Fire Wall |
| Fire Ball | 12 | Fire Bolt | 5/9.5/14.5 | 6-14 / 74-92 / 200-226 fire | +14%/lvl: Fire Bolt, Meteor |
| Nova | 12 | Static Field | 15/24/34 | 1-20 / 57-94 / 131-188 ltng | - |
| Lightning | 12 | Charged Bolt | 8/12.5/17.5 | 1-40 / 1-120 / 1-272 ltng | +8%/lvl: Charged Bolt, Chain Lightning, Nova |
| Shiver Armor | 12 | Ice Blast, Frozen Armor | 11/11/11 | 6-8 / 26-32 / 60-72 cold | +9%/lvl: Frozen Armor, Chilling Armor |
| Fire Wall | 18 | Blaze | 22/31/41 | 23-31 / 166-173 / 428-436 /sec per source fire | +4%/lvl: Warmth, Inferno |
| Enchant | 18 | Warmth, Fire Ball | 25/34/44 | 8-10 / 26-36 / 68-90 fire | +9%/lvl: Warmth |
| Chain Lightning | 18 | Lightning | 9/18/28 | 1-40 / 1-143 / 1-281 ltng | +4%/lvl: Charged Bolt, Lightning, Nova |
| Teleport | 18 | Telekinesis | 24/15/5 | - | - |
| Glacial Spike | 18 | Ice Blast | 10/14.5/19.5 | 16-24 / 91-104 / 225-242 cold | +5%/lvl: Ice Bolt, Ice Blast, Frozen Orb |
| Meteor | 24 | Fire Ball, Fire Wall | 17/21.5/26.5 | 80-100 / 319-357 / 869-927 fire | +5%/lvl: Fire Bolt, Fire Ball |
| Thunder Storm | 24 | Nova, Chain Lightning | 19/19/19 | 1-100 / 91-190 / 195-294 ltng | - |
| Energy Shield | 24 | Teleport, Chain Lightning | 5/5/5 | - | - |
| Blizzard | 24 | Frost Nova, Glacial Spike | 23/32/42 | 45-75 / 210-249 / 570-619 cold | +5%/lvl: Ice Bolt, Ice Blast, Glacial Spike |
| Chilling Armor | 24 | Shiver Armor | 17/17/17 | 4-6 / 15-22 / 39-50 cold | +7%/lvl: Frozen Armor, Shiver Armor |
| Fire Mastery | 30 | - | - | - | - |
| Hydra | 30 | Enchant | 20/24.5/29.5 | 14-20 / 68-82 / 150-175 fire | +3%/lvl: Fire Bolt, Fire Ball |
| Lightning Mastery | 30 | - | - | - | - |
| Frozen Orb | 30 | Blizzard | 25/29.5/34.5 | 40-45 / 134-144 / 262-276 cold | +2%/lvl: Ice Bolt |
| Cold Mastery | 30 | - | - | - | - |

Notes read from the table (T unless noted):
- **Ice Bolt** is the only cold skill at clvl 1 and gives +15%/level to Ice Blast, Glacial Spike, Blizzard, Frozen Orb and Frost Nova, so 4 points in it are never wasted. Ice Blast (clvl 6, prerequisite Ice Bolt) deals 16-24/170-187/506-533 at levels 1/10/20 for 6-11 mana. Glacial Spike (clvl 18) 32-48 at level 1 with a freeze. Blizzard (clvl 24, Frost Nova + Glacial Spike): 45-75 at level 1 to 570-619 at level 20 per shard, mana 39 at level 17; **cast delay 45 frames (1.8 s)** and Frozen Orb 25 frames (T `delay`).
- **Warmth** (clvl 1): +30% mana recovery and +12% per extra level (T Param1/2). One point is the cheapest mana fix there is.
- **Cold Mastery** (clvl 30): Param 20/5; in 1.14d it lowers monsters' cold resistance (S); whether it breaks a full immunity is U (my belief: no, it only affects non-immune monsters), so cold immune packs need another element or a merc.
- **Fire Mastery** (clvl 30) is +30% fire damage +7% per level (T Param 30/7); Fire Bolt gets +16% per Fire Ball and Meteor level, so a fire hybrid needs about 20 more points.
- **Teleport** (clvl 18, prerequisite Telekinesis): mana 24 at level 1, then -1 per level down to the floor (T `lvlmana -1`); delay none. **Energy Shield** (clvl 24, Teleport + Chain Lightning) needs 3 more points and makes mana the life bar (S) - skipped.
- **Frozen Armor** (clvl 1): defense bonus plus chilling attackers (T Param 30/5 per level); skipped on the path so the points feed Blizzard (U).

## 3. Mana, potions, early game problems

Life/mana without gear for this path (T formula, starting values U):

| clvl | Vit points added | Energy points added | life (no gear) | mana (no gear) |
|---|---|---|---|---|
| 10 | 41 | 4 | 131 | 61 |
| 20 | 56 | 4 | 171 | 81 |
| 30 | 96 | 4 | 261 | 101 |
| 40 | 146 | 4 | 371 | 121 |

- At clvl 40 the unaided pool is about 120 mana and Blizzard 17 costs 39: three casts. All real Sorceress play in Hell uses gear mana (+Energy, mana per kill) and a belt of mana potions (V1 "lots of Mana"). A bot should treat Sorceress as **potion-fed**: 2 health + 2 mana columns, buy full stacks every town trip, and plan an Energy-heavier alternative: each 5 Energy points is +10 mana against +10 life (T per-point values 2 and 2), so 3 Energy / 2 Vit per level from clvl 25 gives about +96 mana by 40 (U trade).
- Early (clvl 1-12): Ice Bolt costs 3 and Ice Blast 6-10, a 40-60 mana pool is 5-8 casts. Maxroll plans use Charged Bolt then Nova because Nova needs 13 clvls of Vit and Strength first (V1); our path avoids that.
- Out of potions early: stop leveling in the field, go to town and rest at the well; do not run Blizzard on a bare pool. The path always keeps Ice Bolt (4 points) as the cheap attack: 3 mana per cast, 3-5 damage per hit at level 1, 14-20 at level 10 before synergy (T).
- Mana potions: Akara sells minor/light mana potions (V1); belts matter (V1).

## 4. Leveling path, clvl 1-40, point by point

Assumptions: Den of Evil at clvl 5, Radament 15, Izual 24, NM Den/Radament/Izual at 30/34/38 (U). Machine-validated against `skills.tsv` (reqlevel and prerequisites OK, T). Final allocation at clvl 40: Blizzard 17, Ice Blast 11, Glacial Spike 8, Ice Bolt 4, Cold Mastery 3, Warmth 1, Telekinesis 1, Frost Nova 1, Teleport 1.

| pt | at | skill point | stat points (5/clvl) | Ice Blast / Glacial Spike / Blizzard |
|---|---|---|---|---|
| 1 | clvl 2 | Ice Bolt | 3 Vit, 2 Ene |  |
| 2 | clvl 3 | Ice Bolt | 3 Vit, 2 Ene |  |
| 3 | clvl 4 | Ice Bolt | 5 Vit |  |
| 4 | clvl 5 | Warmth | 5 Vit |  |
| 5 | Den of Evil N (clvl ~5) | Ice Bolt | - |  |
| 6 | clvl 6 | Ice Blast | 5 Vit | Ice Blast 1 mana 6.0 11-16 (+32% syn) |
| 7 | clvl 7 | Ice Blast | 5 Vit | Ice Blast 2 mana 6.5 20-26 (+32% syn) |
| 8 | clvl 8 | Ice Blast | 5 Vit | Ice Blast 3 mana 7.0 29-36 (+32% syn) |
| 9 | clvl 9 | Ice Blast | 5 Vit | Ice Blast 4 mana 7.5 38-46 (+32% syn) |
| 10 | clvl 10 | Ice Blast | 5 Vit | Ice Blast 5 mana 8.0 48-55 (+32% syn) |
| 11 | clvl 11 | Ice Blast | 5 Vit | Ice Blast 6 mana 8.5 57-65 (+32% syn) |
| 12 | clvl 12 | Ice Blast | 5 Vit | Ice Blast 7 mana 9.0 66-75 (+32% syn) |
| 13 | clvl 13 | Ice Blast | 5 Vit | Ice Blast 8 mana 9.5 75-85 (+32% syn) |
| 14 | clvl 14 | Ice Blast | 5 Str | Ice Blast 9 mana 10.0 94-104 (+32% syn) |
| 15 | clvl 15 | Telekinesis | 5 Str | Ice Blast 9 mana 10.0 94-104 (+32% syn) |
| 16 | Radament N (clvl ~15) | Frost Nova | - | Ice Blast 9 mana 10.0 94-104 (+32% syn) |
| 17 | clvl 16 | Ice Blast | 5 Str | Ice Blast 10 mana 10.5 112-123 (+32% syn) |
| 18 | clvl 17 | Ice Blast | 5 Str | Ice Blast 11 mana 11.0 131-143 (+32% syn) |
| 19 | clvl 18 | Teleport | 5 Str | Ice Blast 11 mana 11.0 131-143 (+32% syn) |
| 20 | clvl 19 | Glacial Spike | 5 Str | Ice Blast 11 mana 11.0 131-143 (+32% syn); Glacial Spike 1 mana 10.0 28-42 (+75% syn) |
| 21 | clvl 20 | Glacial Spike | 5 Str | Ice Blast 11 mana 11.0 131-143 (+32% syn); Glacial Spike 2 mana 10.5 40-55 (+75% syn) |
| 22 | clvl 21 | Glacial Spike | 5 Str | Ice Blast 11 mana 11.0 131-143 (+32% syn); Glacial Spike 3 mana 11.0 52-68 (+75% syn) |
| 23 | clvl 22 | Glacial Spike | 5 Str | Ice Blast 11 mana 11.0 131-143 (+32% syn); Glacial Spike 4 mana 11.5 65-81 (+75% syn) |
| 24 | clvl 23 | Glacial Spike | 5 Vit | Ice Blast 11 mana 11.0 131-143 (+32% syn); Glacial Spike 5 mana 12.0 77-94 (+75% syn) |
| 25 | clvl 24 | Blizzard | 5 Vit | Ice Blast 11 mana 11.0 139-151 (+40% syn); Glacial Spike 5 mana 12.0 77-94 (+75% syn); Blizzard 1 mana 23.0 90-150 (+100% syn) |
| 26 | Izual N (clvl ~24) | Blizzard | - | Ice Blast 11 mana 11.0 147-160 (+48% syn); Glacial Spike 5 mana 12.0 77-94 (+75% syn); Blizzard 2 mana 24.0 120-182 (+100% syn) |
| 27 | Izual N 2 (clvl ~24) | Blizzard | - | Ice Blast 11 mana 11.0 154-168 (+56% syn); Glacial Spike 5 mana 12.0 77-94 (+75% syn); Blizzard 3 mana 25.0 150-214 (+100% syn) |
| 28 | clvl 25 | Blizzard | 5 Vit | Ice Blast 11 mana 11.0 162-177 (+64% syn); Glacial Spike 5 mana 12.0 77-94 (+75% syn); Blizzard 4 mana 26.0 180-246 (+100% syn) |
| 29 | clvl 26 | Blizzard | 5 Vit | Ice Blast 11 mana 11.0 170-186 (+72% syn); Glacial Spike 5 mana 12.0 77-94 (+75% syn); Blizzard 5 mana 27.0 210-278 (+100% syn) |
| 30 | clvl 27 | Blizzard | 5 Vit | Ice Blast 11 mana 11.0 178-194 (+80% syn); Glacial Spike 5 mana 12.0 77-94 (+75% syn); Blizzard 6 mana 28.0 240-310 (+100% syn) |
| 31 | clvl 28 | Blizzard | 5 Vit | Ice Blast 11 mana 11.0 186-203 (+88% syn); Glacial Spike 5 mana 12.0 77-94 (+75% syn); Blizzard 7 mana 29.0 270-342 (+100% syn) |
| 32 | clvl 29 | Blizzard | 5 Vit | Ice Blast 11 mana 11.0 194-212 (+96% syn); Glacial Spike 5 mana 12.0 77-94 (+75% syn); Blizzard 8 mana 30.0 300-374 (+100% syn) |
| 33 | clvl 30 | Cold Mastery | 5 Vit | Ice Blast 11 mana 11.0 194-212 (+96% syn); Glacial Spike 5 mana 12.0 77-94 (+75% syn); Blizzard 8 mana 30.0 300-374 (+100% syn) |
| 34 | Den of Evil NM (clvl ~30) | Cold Mastery | - | Ice Blast 11 mana 11.0 194-212 (+96% syn); Glacial Spike 5 mana 12.0 77-94 (+75% syn); Blizzard 8 mana 30.0 300-374 (+100% syn) |
| 35 | clvl 31 | Cold Mastery | 5 Vit | Ice Blast 11 mana 11.0 194-212 (+96% syn); Glacial Spike 5 mana 12.0 77-94 (+75% syn); Blizzard 8 mana 30.0 300-374 (+100% syn) |
| 36 | clvl 32 | Blizzard | 5 Vit | Ice Blast 11 mana 11.0 202-220 (+104% syn); Glacial Spike 5 mana 12.0 77-94 (+75% syn); Blizzard 9 mana 31.0 360-436 (+100% syn) |
| 37 | clvl 33 | Blizzard | 5 Vit | Ice Blast 11 mana 11.0 210-229 (+112% syn); Glacial Spike 5 mana 12.0 77-94 (+75% syn); Blizzard 10 mana 32.0 420-498 (+100% syn) |
| 38 | clvl 34 | Blizzard | 5 Vit | Ice Blast 11 mana 11.0 218-238 (+120% syn); Glacial Spike 5 mana 12.0 77-94 (+75% syn); Blizzard 11 mana 33.0 480-560 (+100% syn) |
| 39 | Radament NM (clvl ~34) | Blizzard | - | Ice Blast 11 mana 11.0 226-246 (+128% syn); Glacial Spike 5 mana 12.0 77-94 (+75% syn); Blizzard 12 mana 34.0 540-622 (+100% syn) |
| 40 | clvl 35 | Blizzard | 5 Vit | Ice Blast 11 mana 11.0 234-255 (+136% syn); Glacial Spike 5 mana 12.0 77-94 (+75% syn); Blizzard 13 mana 35.0 600-684 (+100% syn) |
| 41 | clvl 36 | Blizzard | 5 Vit | Ice Blast 11 mana 11.0 242-264 (+144% syn); Glacial Spike 5 mana 12.0 77-94 (+75% syn); Blizzard 14 mana 36.0 660-746 (+100% syn) |
| 42 | clvl 37 | Blizzard | 5 Vit | Ice Blast 11 mana 11.0 249-272 (+152% syn); Glacial Spike 5 mana 12.0 77-94 (+75% syn); Blizzard 15 mana 37.0 720-808 (+100% syn) |
| 43 | clvl 38 | Blizzard | 5 Vit | Ice Blast 11 mana 11.0 257-281 (+160% syn); Glacial Spike 5 mana 12.0 77-94 (+75% syn); Blizzard 16 mana 38.0 780-870 (+100% syn) |
| 44 | Izual NM (clvl ~38) | Blizzard | - | Ice Blast 11 mana 11.0 265-289 (+168% syn); Glacial Spike 5 mana 12.0 77-94 (+75% syn); Blizzard 17 mana 39.0 870-962 (+100% syn) |
| 45 | Izual NM 2 (clvl ~38) | Glacial Spike | - | Ice Blast 11 mana 11.0 265-289 (+168% syn); Glacial Spike 6 mana 12.5 89-108 (+75% syn); Blizzard 17 mana 39.0 892-986 (+105% syn) |
| 46 | clvl 39 | Glacial Spike | 5 Vit | Ice Blast 11 mana 11.0 265-289 (+168% syn); Glacial Spike 7 mana 13.0 102-121 (+75% syn); Blizzard 17 mana 39.0 914-1010 (+110% syn) |
| 47 | clvl 40 | Glacial Spike | 5 Vit | Ice Blast 11 mana 11.0 265-289 (+168% syn); Glacial Spike 8 mana 13.5 114-134 (+75% syn); Blizzard 17 mana 39.0 935-1034 (+115% syn) |

Stat plan used by the table: clvl 2-3 two points to Energy, otherwise Vitality through clvl 13, Strength 5 per clvl from 14 to 22 for gear (about 65 Str, enough for Stealth/Lore armor and a staff), Vitality again from 23 (U; Maxroll is Vit with Str 22+ at clvl 17-21, V1).
Why this order: Ice Bolt 4 for early synergy; Warmth 1 for mana recovery; Ice Blast to 11 by clvl 18 because it is the only good attack at clvl 6-23 (ice blast hits and freezes, S); Telekinesis + Teleport at 15-18 to move fast (Teleport lets the bot skip packs, U); Glacial Spike 5 as the Blizzard prerequisite; Blizzard 8 by clvl 27, Cold Mastery 3 at 30-32, then Blizzard 17 by 40. Humans do something faster: Charged Bolt -> Static Field/Frost Nova -> Nova at 13, respec at 26 into Blizzard (V1).

## 5. Mercenary

- Act 2 Desert merc (V AS hirelings, T hireling.tsv): Defiance (Normal), Holy Freeze (NM; slows packs, helps the squishy caster) or Might; Hell Defiance/Prayer. Normal cost 350 gold at level 9, 7,900 NM (T). A Defiance merc survives the first minute of every pack.
- Act 3 Iron Wolf **Cold** (Glacial Spike, Frozen Armor, Ice Blast) is a free second caster but fragile: Normal level 15 hire costs 1,500 (cold) vs 1,000 (fire/lightning), NM 12,000, Hell 27,000 (T Gold). Not recommended for a bot because the merc stands in melee range.
- Gear for the merc: the Act 2 merc takes spear/polearm and armor; no Infinity offline (ladder, V `builds-open-questions.md` Q3).

## 6. Gear per act on a budget

| stage | weapon | armor / helm / shield | rest | notes |
|---|---|---|---|---|
| Normal Act 1-2 | start Short Staff; any magic staff with +skills or FCR from drops or Charsi/Akara stock (U) | Quilted/Stealth armor, Cap; Lore helm (Ort+Sol, 3 sockets, T) at clvl ~20 | Belt (12 slots) from Fara (V1) | Charsi socketing does not exist; Larzuk is Act 5 (V AS rewards). |
| Normal Act 3-5 | a magic/rare staff with +skills or FCR; Staff of Teleportation is for classes without Teleport (V1) | Stealth armor (Tal+Eth, T) at clvl 17 (V1), Rhyme/Rune shield | rings/amulets with FCR | Maxroll hits "63% FCR" for Sorc in Normal/NM (V1) |
| Nightmare | Memory (Lum+Io+Sol+Eth staff, +3 sorc skills, +3 Energy Shield, 33% FCR, 20% max mana, T `runes.tsv` and AB) or Heart of the Oak (Ko+Vex+Pul+Thul staff/mace, T) | Stone (Shael+Um+Pul+Lum, +40 FHR; T) body or Chains of Honor (Dol+Um+Ber+Ist, V open-questions Q2); Delirium helm is risky (rat man transform) | resist ring/amulet, FCR jewelry (V1) | Nightmare resist target +20 after -40 -> raw 60 each (V AS, V1) |
| Hell | Memory/Heart of the Oak staff; Eschuta's Temper/Oculus uniques (V1) | Enigma (Jah+Ith+Ber, +2 skills, T) | Arachnid Mesh, Skin of the Vipermagi, SoJ (V1) | Enigma gives Teleport via +2 skills only if the bot has a point in Teleport (U) |

## 7. Resists and immunities

Targets: Normal 0 needed. Nightmare entry raw 60 fire/cold/lightning (+20 after -40, V AS penalty). Hell raw 125 minimum (+25 after -100), 175 to cap (V AS). Poison is not covered by any standard gear rule; Cleansing-type fixes are not available to a Sorceress (U).

Immunity landscape (T, distinct regular spawn types immune, NM / Hell):
| act | distinct spawn types | physical imm NM / Hell | fire imm NM / Hell | lightning imm NM / Hell | cold imm NM / Hell |
|---|---|---|---|---|---|
| 1 | NM 55 / Hell 55 | 0 / 2 | 0 / 16 | 0 / 14 | 0 / 11 |
| 2 | NM 57 / Hell 57 | 0 / 4 | 1 / 13 | 0 / 11 | 0 / 13 |
| 3 | NM 57 / Hell 57 | 0 / 2 | 0 / 10 | 0 / 15 | 0 / 17 |
| 4 | NM 20 / Hell 21 | 0 / 0 | 1 / 4 | 1 / 8 | 0 / 4 |
| 5 | NM 158 / Hell 47 | 0 / 4 | 2 / 11 | 4 / 8 | 10 / 21 |

- Hell cold-immune spawn types by act:
  - Act 1: Zombie, Gargantuan Beast, Dark Hunter, Moon Clan, Vile Hunter, Brute, Dark Stalker, Bone Warrior, Yeti, Black Rogue, The Banished
  - Act 2: Huntress, Slinger, Cave Leaper, Marauder, Claw Viper, Salamander, Crusher, Hell Cat, Horror Mage, Spear Cat, Unraveler, Ghoul Lord, Gorebelly
  - Act 3: Thorned Hulk, Bog Creature, Bramble Hulk, Flayer, Thrasher, Faithful, Sexton, Infidel, Heirophant, Night Lord, Gloombat, Flesh Hunter, Serpent Magus, Flesh Archer, Dark Lord, Blood Lord
  - Act 4: Flesh Spawner, Stygian Hag, Grotesque, Oblivion Knight
  - Act 5: Lasher, Snow Drifter, Frozen Terror, Death Slasher, Abominable, Frozen Scourge, Death Brigadier, Frozen Abyss, Korlic, Reanimated Horde, Prowling Dead, Ice Spawn, Hell Lord, Overlord, Blood Boss, Vile Witch, Flesh Hunter, Dark Archer, Hell Whip, Oblivion Knight, Pit Lord

How humans handle it: the cold Sorceress uses (a) **a second element on the same character** - Lightning (Charged Bolt, Lightning, Chain Lightning) or Fire (Fire Ball, Meteor, Hydra) which cost 20+ points (V1 recommends separate builds), (b) a merc and **Static Field** against bosses (V1 for Mephisto; Static Field cannot take life below 33% NM / 50% Hell, V AS), (c) a Wand of Lower Resist on the weapon swap (V1; the bot has no swap) or Cold Mastery (lowers resist; I believe it does not remove a full immunity, U), (d) skipping the pack. In Hell the cold-immune share is 11/55 types in Act 1, 13/57 Act 2, 17/57 Act 3, 4/21 Act 4, 21/47 Act 5 (T), so a pure cold Sorceress faces immune packs in roughly one pack in four. **Not recommended in Hell without a second element or a respec** (V1 + T).

## 8. Strengths, weaknesses, deaths by act/difficulty

- Normal Act 1-2 (clvl 1-24): fastest start; weak spots are mana (first 10 clvls), the Countess/Blood Raven archers, Andariel's poison, Duriel's cold/stun (U). Frost Nova + Teleport away is the human technique; a bot that stands still dies (U).
- Normal Act 3-5: fire/lightning casters (Flayer Shamans, Council of Zakarum, Sexton); Mephisto's lightning and Diablo's pools kill 500-life casters (U). Act 5 Barbarian soldiers close fast (U).
- Nightmare: -40 resists with 170-260 life without Vit (T table above) is the main killer; the Mephisto/Izual/Diablo bolts one-shot a low-life Sorc in unlucky rolls (U). Merc revival costs scale (V AS: capped 50,000).
- Hell: fire/lightning/cold immune packs, Act 5 Barbarian soldiers, Hell's -100 resist makes every elemental enchant lethal (T penalty V AS).
- Typical bot deaths (U, predicted): standing in Blizzard's cooldown (1.8 s, T) while a melee pack closes; no mana mid-fight; Teleport misclick into a wall of monsters.

## 9. Three common builds and their trade-offs

| build | skills | pros | cons | no-respec fit |
|---|---|---|---|---|
| Blizzard/Frozen Orb (this file) | Ice Bolt..Blizzard, Cold Mastery | safest boss/crowd clear, freezes, V1 "best safe Sorceress" | cold immunes (T above), mana, slow to start | best, with Warmth and potions |
| Lightning/Chain Lightning | Charged Bolt, Lightning, Chain Lightning, Static Field, Lightning Mastery (T prereqs) | highest single-target damage (V1 Maxroll Lightning Sorc) | huge damage range, lightning immunes (T: 8-15 types per Hell act), bad early | needs 20+ points before it works |
| Fire (Fire Wall / Hydra / Meteor) | Fire Bolt, Blaze, Fire Wall at 18, Hydra at 30, Fire Mastery | cheap bosses (V1 Mephisto) | fire immunes are the largest group (T Hell Acts 1-3: 10-16 types each) | worst as sole skill |

## 10. When is the character "ready" (bot checks; U unless tagged)

| step | minimum | extra checks |
|---|---|---|
| Den of Evil / Blood Raven | clvl 6-8, Ice Bolt 4, Ice Blast 1, 4 mana + 4 health potions | belt >= 8 slots |
| Andariel | clvl 16-18, Ice Blast 9, life >= 160, mana pool >= 75 or 6+ mana potions | Frost Nova (prereq of Blizzard) not required for this fight |
| Duriel / Act 3 | clvl 21-23, life >= 220, Teleport 1, cold resist >= 20 | staff or gem for FCR 20+ |
| Mephisto | clvl 26-28, Blizzard 1-3, life >= 260, all res >= 40 | life and resists are the limiter |
| Diablo | clvl 28-30, life >= 300, all res >= 50 | 4 mana potions |
| Baal | clvl 31-34, life >= 350, +60 res | Rite of Passage claimed (V AS rewards: needs clvl 20+ in Normal) |
| Enter Nightmare | clvl >= 38-40 (V AS: Act 5 exp good until 40), raw res >= 60, life >= 370 (T path value at 40; real gear adds more), Teleport + Blizzard >= 15 | 6+ mana potions, 4 health, merc alive |
| Enter Hell | clvl >= 64 (project gate V `docs/progression/difficulty.md`), raw res >= 125, life >= 900, a second element or an immune-skip plan, mana >= 300 from gear | Enigma or Teleport charges |

## 11. Open questions

1. Whether Cold Mastery pierces full cold immunity in 1.14d (U).
2. Real life and mana at clvl 40 on a bot: read from the live game; the T table above omits gear and Vit from items.
3. Starting life/mana for all classes are U (not in the extract).

## Shared facts (apply to every class; V = Arreat Summit archive, still served by Blizzard)

- Resist floors by difficulty: Normal 0, Nightmare -40, Hell -100; absolute floor -100; default cap 75 (V `https://classic.battle.net/diablo2exp/basics/resistances.shtml`). Hell therefore needs raw 175 to cap at 75, raw 125 for +25.
- Nightmare: unique/champion damage +75% (Normal +90%, Hell +66%), summons and hirelings deal only 35% damage to act bosses (Normal 50%, Hell 25%), life/mana steal halved (Hell cut by 2/3), freeze/chill/curses halved (Hell cut by 3/4), Static Field cannot take a monster below 33% (NM) / 50% (Hell), monster skill levels +3 (NM) / +7 (Hell), death costs 5% (NM) / 10% (Hell) of the exp gap to the next level, 75% of it restored when the corpse is picked up (V `https://classic.battle.net/diablo2exp/basics/difficulty.shtml`).
- Mercenary aura table, identical in the Arreat page and our tables (V `https://classic.battle.net/diablo2exp/basics/hirelings.shtml`, `hireling.tsv` of this install): Act 2 Desert mercenary Combat = Prayer (Normal) / Thorns (NM) / Prayer (Hell); Defense = Defiance / Holy Freeze / Defiance; Offense = Blessed Aim / Might / Blessed Aim. So Might is an NM-only option; in Hell the Act 2 offense merc runs Blessed Aim. Act 1 Rogue: Fire or Cold Arrow plus Inner Sight (bow). Act 3 Iron Wolf: Fire (Inferno/Fire Ball), Cold (Glacial Spike/Frozen Armor/Ice Blast) or Lightning (Charged Bolt/Lightning), sword + shield, "rarely swing". Act 5 Barbarian: Bash and Stun, one sword. No Act 4 merc. Revival never costs more than 50,000 gold; if the player dies the merc dies too.
- "T" below means a value computed from this install's own game tables (`skills.tsv`, `charstats.tsv`, `hireling.tsv`, `monstats.tsv`, `levels.tsv`, `runes.tsv`; the worker host `~/.local/share/d2bot/cache/data/v2-6ff77ee7fff5/`), V by construction. Skill damage/mana formulas: mana = (mana + lvlmana*(lvl-1)) * 2^manashift / 256 with a floor of minmana; added elemental damage = (EMin + sum of the per-level steps for levels 2-8, 9-16, 17-22, 23-28, 29+) * 2^HitShift / 256, times (1 + synergy%). Cross-checked against Amazon Basin pages (Exploding Arrow mana 4.5+0.5/lvl, Blessed Hammer 4.75+0.25/lvl, Firestorm +23% per Molten Boulder/Fissure level, Bone Spear +7% per Teeth/Bone Wall/Prison/Spirit level, `https://theamazonbasin.com/wiki/index.php?title=<Skill>`). Skills with HitShift 2-4 (Firestorm, Inferno, Fire Wall, Blaze, Arctic Blast, Wake of Inferno) are damage per second per source (x25 frames); poison skills show the total over their duration where resolvable (S).
- Immunity tables are built from `levels.tsv` spawn lists (mon/nmon/umon columns) joined to `monstats.tsv` resists (>=100 means immune); they count distinct regular spawn types per act and difficulty, excluding bosses/super uniques (T). Immunities on unique and champion monsters are rolled on top (enchants and auras), so these are lower bounds for "something nasty is immune". The Act 5 Nightmare spawn lists in `levels.tsv` are unusually long (158 types), so read that row as a rough shape.
- Tag "V1" = taken from the first-pass SME files (`builds-leveling.md`, `builds-gear.md`, `builds-endgame-*.md`), where it was fetched from Maxroll D2R Season 14 guides (D2R text; mechanics match 1.14d except where flagged). I did not re-fetch those pages. Ladder-only runewords (Spirit, Insight, Fortitude, Infinity, Dragon, Dream, Edge, Faith, Harmony, Ice, Brand, Death, Destruction, Last Wish, Lawbringer, Oath, Obedience, Phoenix, Pride, Rift, Grief, Voice of Reason, Wrath) are not craftable in vanilla offline 1.14d (V `builds-open-questions.md` Q3); Enigma, Chains of Honor, Stone, Memory, Delirium, Lore, Rhyme, Smoke, Stealth, Heart of the Oak, Beast, Fury, Honor, Doom, Hand of Justice, Ancient's Pledge, Bone, Peace, Treachery, Duress, Gloom are available (T: server flag empty).


## BUILDV2 implementation note (2026-10-30, fresh Sorceress only)
Table `fallback-lightning-v2` (`internal/plandata/build-fallback-lightning-v2.json`, selected by `leveler.SelectBuild`): Charged Bolt to 4 at clvl 5 with the Den of Evil reward point on Frozen Armor 1 (table order: the 5th skill point is Frozen Armor whenever the reward arrives, V by `TestBuildV2SorceressSequence` for Den at clvl 3, 5, 8), Frost Nova 1 at clvl 6 instead of Static Field, Telekinesis at 11, Charged Bolt 8 at clvl 10 and 12 at clvl 15; 5 Vitality per level after the Str floor of 12 (no Energy shift, U). A character that owns Static Field keeps the old `fallback` table. Frost Nova on a crowd of 3+ needs `combat.novaManaFloor` mana (default 25, U), Frozen Armor is recast every `combat.buffRecastSec` (150 s, duration U).
