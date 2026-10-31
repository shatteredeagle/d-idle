# SME class file: Assassin (classic LoD 1.14d, offline single player, softcore)

Owner: SMEKB-B2. Companions: `builds-leveling.md`, `builds-endgame-melee.md` (assassin section), `builds-gear.md`, `builds-open-questions.md`, `docs/classes/assassin.md`. Feeds an unattended bot with no respec (`docs/leveler.md`).
Tags: **V** fetched or computed from the install's tables (URL or "T"), **V1** carried from first-pass SME files (Maxroll D2R guides, not re-fetched), **S** secondary, **U** memory or judgement. D2R-only facts are marked "D2R".

## 0. Verdict

The Assassin is a trap caster: traps are cast at the Assassin's feet or at a target, hit what stands in them, and many traps can be active at once (the classic cap is 5 per type, S). Her ranged safety is good and her life is average (+2 per clvl, +3 per Vit, T), but **trap placement is a planning problem** (where to stand, when to lay them) and **trap mana is high**: Wake of Fire costs 13 mana per trap, Wake of Inferno 20, a Lightning Sentry 20 (T), so a 90 mana pool is 4-5 traps. The no-respec path below is the **fire trap line (Fire Blast -> Wake of Fire -> Wake of Inferno)** that the project already uses (`assassin_fireblast_normal30.json`, V1) and that Maxroll uses to clvl 38 before the respec into Lightning/Death Sentry (V1). Cold and fire immunes limit it in Hell (T below).

## 1. Class basics (T `charstats.tsv`)

| item | value |
|---|---|
| Starting stats | Str 20, Dex 20, Int 25, Vit 20 |
| Life | +2 per clvl, +3 per Vit; Mana +1.5 per clvl, +1.75 per Energy (best Energy ratio among melee hybrids); Stamina +1.25/clvl, +1.25/Vit |
| Starting life/mana (U) | about 50 / 25 |
| Starting equipment | Katar, Buckler, 4 minor healing potions, TP and ID scrolls (T codes ktr, buc, hp1 x4, tsc, isc) |
| Block / AR | block 25, ToHitFactor 15 per Dex |

Points: 1 per clvl from 2 plus 4 per difficulty; 47 at clvl 40 on our timeline. Stat points 5 per clvl.

## 2. Skills, synergies, per-level numbers (T)

Traps show added damage per hit or per second per source (/s). Mana at skill level 1/10/20 is per trap. Assassin internal names: Fire Trauma = Fire Blast, Shock Field = Shock Web, Quickness = Burst of Speed (V1, T `skills.tsv`).

| skill | clvl | prereqs | mana L1/L10/L20 | added dmg L1 / L10 / L20 (type) | synergy |
|---|---|---|---|---|---|
| Fire Blast | 1 | - | 3/4.1/5.4 | 3-4 / 22-32 / 86-114 fire | +9%/lvl: Shock Web, Death Sentry, Charged Bolt Sentry, Lightning Sentry, Wake of Fire, Wake of Inferno |
| Claw Mastery | 1 | - | - | - | - |
| Psychic Hammer | 1 | - | 4/6.2/8.8 | 1-3 / 11-18 / 28-40 mag | - |
| Tiger Strike | 1 | - | 1/1/1 | - | - |
| Dragon Talon | 1 | - | 6/6/6 | - | - |
| Shock Web | 6 | Fire Blast | 6/6/6 | 1-10 / 1-43 / 1-119 ltng | +11%/lvl: Charged Bolt Sentry, Lightning Sentry, Death Sentry |
| Blade Sentinel | 6 | - | 7/7/7 | - | - |
| Burst of Speed | 6 | Claw Mastery | 10/10/10 | - | - |
| Fists of Fire | 6 | - | 2/2/2 | 6-10 / 61-67 / 201-221 fire | +12%/lvl: Phoenix Strike |
| Dragon Claw | 6 | Dragon Talon | 2/2/2 | - | - |
| Charged Bolt Sentry | 12 | Shock Web | 13/13/13 | 1-7 / 1-36 / 1-84 ltng | +6%/lvl: Fire Blast, Lightning Sentry, Death Sentry |
| Wake of Fire | 12 | Fire Blast | 13/13/13 | 5-10 / 25-30 / 63-72 fire | +8%/lvl: Fire Blast, Wake of Inferno |
| Weapon Block | 12 | Claw Mastery | - | - | - |
| Cloak of Shadows | 12 | Psychic Hammer | 13/13/13 | - | - |
| Cobra Strike | 12 | Tiger Strike | 2/2/2 | - | - |
| Blade Fury | 18 | Blade Sentinel, Wake of Fire | 1/2.1/3.4 | - | - |
| Fade | 18 | Burst of Speed | 10/10/10 | - | - |
| Shadow Warrior | 18 | Cloak of Shadows, Weapon Block | 27/31.5/36.5 | - | - |
| Claws of Thunder | 18 | Fists of Fire | 4/4/4 | 1-80 / 1-300 / 1-780 ltng | +8%/lvl: Phoenix Strike |
| Dragon Tail | 18 | Dragon Claw | 10/10/10 | - | - |
| Lightning Sentry | 24 | Charged Bolt Sentry | 20/20/20 | 1-20 / 1-122 / 1-314 ltng | +12%/lvl: Shock Web, Charged Bolt Sentry, Death Sentry |
| Wake of Inferno | 24 | Wake of Fire | 20/20/20 | 31-78 / 283-358 / 642-748 /sec per source fire | +7%/lvl: Fire Blast, Death Sentry, Wake of Fire |
| Mind Blast | 24 | Cloak of Shadows | 15/15/15 | - | - |
| Blades of Ice | 24 | Claws of Thunder | 3/3/3 | 15-35 / 91-111 / 231-259 cold | +8%/lvl: Phoenix Strike |
| Dragon Flight | 24 | Dragon Tail | 15/15/15 | - | - |
| Death Sentry | 30 | Lightning Sentry | 20/20/20 | 1-50 / 1-134 / 1-306 ltng | +12%/lvl: Lightning Sentry |
| Blade Shield | 30 | Blade Fury | 27/45/65 | - | - |
| Venom | 30 | Fade | 12/12/12 | poison, per-frame units (not resolved) | - |
| Shadow Master | 30 | Shadow Warrior | 35/39.5/44.5 | - | - |
| Phoenix Strike | 30 | Cobra Strike, Blades of Ice | 4/4/4 | - | - |

Formulas read from the table (T unless noted):
- **Fire Blast** (clvl 1, 3 mana): +9% fire damage per Shock Web, Death Sentry, Charged Bolt Sentry, Lightning Sentry, Wake of Fire and Wake of Inferno level, so every trap skill raises it; Wake of Fire gets +8% per Fire Blast and Wake of Inferno level; Wake of Inferno gets +10% per Fire Blast level and +7% per Wake of Fire level (T Param7/8; AB not re-fetched). It is a fire trap that shoots at the target.
- **Wake of Fire** (clvl 12): a trap that fires flames at the target, 13 mana, 5-10 / 25-30 / 63-72 added fire at levels 1/10/20 before synergy (T). **Wake of Inferno** (clvl 24, prereq Wake of Fire): 20 mana, burning ground, damage per second per source in the table (hit shift 4; /s).
- **Lightning Sentry** (clvl 24, prereq Charged Bolt Sentry <- Shock Web <- Fire Blast): 20 mana, +12% per Shock Web/Charged Bolt Sentry/Death Sentry level. **Death Sentry** (clvl 30): same +12% (T). This is the human endgame line (V1), costing 15+ points of prerequisites.
- **Burst of Speed** (clvl 6, prereq Claw Mastery): 10 mana, speed aura-like buff (T Param 15/70); **Fade** (clvl 18) gives resistances and Param 10/75 chance (T); **Cloak of Shadows** (clvl 12, prereq Psychic Hammer): 13 mana, 30/30 Param (T).
- Claw Mastery (clvl 1, Param 30/10/35/4/0/25, T): a mastery for claws only; on this path it is a one-point prerequisite of Burst of Speed.

## 3. Mana, potions, early game problems

| clvl | Vit points added | Energy points added | life (no gear) | mana (no gear) |
|---|---|---|---|---|
| 10 | 39 | 6 | 185 | 49 |
| 20 | 66 | 6 | 286 | 64 |
| 30 | 116 | 6 | 456 | 79 |
| 40 | 166 | 6 | 626 | 94 |

- Mana is the binding constraint: the unaided pool is about 94 at clvl 40 (T, 6 Energy points); Wake of Fire costs 13 per trap, Wake of Inferno 20, so one pack = 4-5 traps and the bot must buy a mana column in the belt (V1 Maxroll: "lots of mana"). The Assassin's Energy ratio (1.75 per point, T) is the best of the non-casters, so putting 2 points per level into Energy is cheaper than for a Barbarian (U).
- Early (clvl 1-11): Fire Blast at 3-4 mana and 3-8 damage at level 1-5 (T); the Katar (start) is a martial-arts weapon but our path uses no martial arts. Burst of Speed (10 mana, Claw Mastery 1) is the movement tool for kiting (V1 Maxroll puts Burst at clvl 6-11).
- If potions run out: stop fighting, trap damage is mana-limited anyway; run to town and rest. Do not stand next to your own Wake of Inferno (it is a ground fire that hurts only enemies; U).
- Stat plan: 3 Energy 2 Vit at clvl 2-3 (Maxroll: Energy first, then Vit, V1), Vit mostly, 3 Str / 2 Vit at 12-18 for armor (U).

## 4. Leveling path, clvl 1-40, point by point

Assumptions: Den of Evil at clvl 5, Radament 15, Izual 24, NM Den/Radament/Izual at 30/34/38 (U). Machine-validated against `skills.tsv` (reqlevel and prerequisites OK, T). Final allocation at clvl 40: Wake of Fire 20, Fire Blast 13, Wake of Inferno 12, Claw Mastery 1, Burst of Speed 1. The right column shows Fire Blast, Wake of Fire, Wake of Inferno with mana and added damage including synergy.

| pt | at | skill point | stat points (5/clvl) | Fire Blast / Wake of Fire / Wake of Inferno |
|---|---|---|---|---|
| 1 | clvl 2 | Fire Blast | 3 Ene, 2 Vit | Fire Blast 1 mana 3.0 3-4 |
| 2 | clvl 3 | Fire Blast | 3 Ene, 2 Vit | Fire Blast 2 mana 3.1 4-6 |
| 3 | clvl 4 | Fire Blast | 5 Vit | Fire Blast 3 mana 3.2 6-9 |
| 4 | clvl 5 | Fire Blast | 5 Vit | Fire Blast 4 mana 3.4 8-12 |
| 5 | Den of Evil N (clvl ~5) | Claw Mastery | - | Fire Blast 4 mana 3.4 8-12 |
| 6 | clvl 6 | Burst of Speed | 5 Vit | Fire Blast 4 mana 3.4 8-12 |
| 7 | clvl 7 | Fire Blast | 5 Vit | Fire Blast 5 mana 3.5 9-14 |
| 8 | clvl 8 | Fire Blast | 5 Vit | Fire Blast 6 mana 3.6 10-16 |
| 9 | clvl 9 | Fire Blast | 5 Vit | Fire Blast 7 mana 3.8 12-19 |
| 10 | clvl 10 | Fire Blast | 5 Vit | Fire Blast 8 mana 3.9 14-22 |
| 11 | clvl 11 | Fire Blast | 3 Vit, 2 Str | Fire Blast 9 mana 4.0 18-27 |
| 12 | clvl 12 | Wake of Fire | 3 Str, 2 Vit | Fire Blast 9 mana 4.0 19-29 (+9% syn); Wake of Fire 1 mana 13.0 9-17 (+72% syn) |
| 13 | clvl 13 | Wake of Fire | 3 Str, 2 Vit | Fire Blast 9 mana 4.0 21-32 (+18% syn); Wake of Fire 2 mana 13.0 12-21 (+72% syn) |
| 14 | clvl 14 | Wake of Fire | 3 Str, 2 Vit | Fire Blast 9 mana 4.0 22-34 (+27% syn); Wake of Fire 3 mana 13.0 15-24 (+72% syn) |
| 15 | clvl 15 | Wake of Fire | 3 Str, 2 Vit | Fire Blast 9 mana 4.0 24-37 (+36% syn); Wake of Fire 4 mana 13.0 19-28 (+72% syn) |
| 16 | Radament N (clvl ~15) | Wake of Fire | - | Fire Blast 9 mana 4.0 25-39 (+45% syn); Wake of Fire 5 mana 13.0 22-31 (+72% syn) |
| 17 | clvl 16 | Wake of Fire | 3 Str, 2 Vit | Fire Blast 9 mana 4.0 27-42 (+54% syn); Wake of Fire 6 mana 13.0 26-34 (+72% syn) |
| 18 | clvl 17 | Wake of Fire | 3 Str, 2 Vit | Fire Blast 9 mana 4.0 29-44 (+63% syn); Wake of Fire 7 mana 13.0 29-38 (+72% syn) |
| 19 | clvl 18 | Wake of Fire | 3 Str, 2 Vit | Fire Blast 9 mana 4.0 30-46 (+72% syn); Wake of Fire 8 mana 13.0 33-41 (+72% syn) |
| 20 | clvl 19 | Wake of Fire | 5 Vit | Fire Blast 9 mana 4.0 32-49 (+81% syn); Wake of Fire 9 mana 13.0 38-46 (+72% syn) |
| 21 | clvl 20 | Wake of Fire | 5 Vit | Fire Blast 9 mana 4.0 33-51 (+90% syn); Wake of Fire 10 mana 13.0 43-52 (+72% syn) |
| 22 | clvl 21 | Wake of Fire | 5 Vit | Fire Blast 9 mana 4.0 35-54 (+99% syn); Wake of Fire 11 mana 13.0 48-57 (+72% syn) |
| 23 | clvl 22 | Wake of Fire | 5 Vit | Fire Blast 9 mana 4.0 36-56 (+108% syn); Wake of Fire 12 mana 13.0 53-62 (+72% syn) |
| 24 | clvl 23 | Wake of Fire | 5 Vit | Fire Blast 9 mana 4.0 38-59 (+117% syn); Wake of Fire 13 mana 13.0 58-67 (+72% syn) |
| 25 | clvl 24 | Wake of Inferno | 5 Vit | Fire Blast 9 mana 4.0 40-61 (+126% syn); Wake of Fire 13 mana 13.0 61-70 (+80% syn); Wake of Inferno 1 mana 20.0 79-198/s (+154% syn) |
| 26 | Izual N (clvl ~24) | Wake of Fire | - | Fire Blast 9 mana 4.0 41-63 (+135% syn); Wake of Fire 14 mana 13.0 67-76 (+80% syn); Wake of Inferno 1 mana 20.0 82-204/s (+161% syn) |
| 27 | Izual N 2 (clvl ~24) | Wake of Fire | - | Fire Blast 9 mana 4.0 43-66 (+144% syn); Wake of Fire 15 mana 13.0 72-81 (+80% syn); Wake of Inferno 1 mana 20.0 84-209/s (+168% syn) |
| 28 | clvl 25 | Wake of Fire | 5 Vit | Fire Blast 9 mana 4.0 44-68 (+153% syn); Wake of Fire 16 mana 13.0 77-86 (+80% syn); Wake of Inferno 1 mana 20.0 86-215/s (+175% syn) |
| 29 | clvl 26 | Wake of Fire | 5 Vit | Fire Blast 9 mana 4.0 46-71 (+162% syn); Wake of Fire 17 mana 13.0 86-97 (+80% syn); Wake of Inferno 1 mana 20.0 88-220/s (+182% syn) |
| 30 | clvl 27 | Wake of Fire | 5 Vit | Fire Blast 9 mana 4.0 47-73 (+171% syn); Wake of Fire 18 mana 13.0 95-108 (+80% syn); Wake of Inferno 1 mana 20.0 90-226/s (+189% syn) |
| 31 | clvl 28 | Wake of Fire | 5 Vit | Fire Blast 9 mana 4.0 49-76 (+180% syn); Wake of Fire 19 mana 13.0 104-119 (+80% syn); Wake of Inferno 1 mana 20.0 92-231/s (+196% syn) |
| 32 | clvl 29 | Wake of Fire | 5 Vit | Fire Blast 9 mana 4.0 51-78 (+189% syn); Wake of Fire 20 mana 13.0 113-130 (+80% syn); Wake of Inferno 1 mana 20.0 95-237/s (+203% syn) |
| 33 | clvl 30 | Wake of Inferno | 5 Vit | Fire Blast 9 mana 4.0 52-80 (+198% syn); Wake of Fire 20 mana 13.0 118-135 (+88% syn); Wake of Inferno 2 mana 20.0 175-327/s (+203% syn) |
| 34 | Den of Evil NM (clvl ~30) | Wake of Inferno | - | Fire Blast 9 mana 4.0 54-83 (+207% syn); Wake of Fire 20 mana 13.0 123-141 (+96% syn); Wake of Inferno 3 mana 20.0 256-417/s (+203% syn) |
| 35 | clvl 31 | Wake of Inferno | 5 Vit | Fire Blast 9 mana 4.0 55-85 (+216% syn); Wake of Fire 20 mana 13.0 129-147 (+104% syn); Wake of Inferno 4 mana 20.0 336-507/s (+203% syn) |
| 36 | clvl 32 | Fire Blast | 5 Vit | Fire Blast 10 mana 4.1 68-103 (+216% syn); Wake of Fire 20 mana 13.0 134-153 (+112% syn); Wake of Inferno 4 mana 20.0 344-518/s (+210% syn) |
| 37 | clvl 33 | Fire Blast | 5 Vit | Fire Blast 11 mana 4.2 81-120 (+216% syn); Wake of Fire 20 mana 13.0 139-158 (+120% syn); Wake of Inferno 4 mana 20.0 352-530/s (+217% syn) |
| 38 | clvl 34 | Fire Blast | 5 Vit | Fire Blast 12 mana 4.4 93-137 (+216% syn); Wake of Fire 20 mana 13.0 144-164 (+128% syn); Wake of Inferno 4 mana 20.0 359-542/s (+224% syn) |
| 39 | Radament NM (clvl ~34) | Fire Blast | - | Fire Blast 13 mana 4.5 106-155 (+216% syn); Wake of Fire 20 mana 13.0 149-170 (+136% syn); Wake of Inferno 4 mana 20.0 367-553/s (+231% syn) |
| 40 | clvl 35 | Wake of Inferno | 5 Vit | Fire Blast 13 mana 4.5 109-159 (+225% syn); Wake of Fire 20 mana 13.0 154-176 (+144% syn); Wake of Inferno 5 mana 20.0 455-652/s (+231% syn) |
| 41 | clvl 36 | Wake of Inferno | 5 Vit | Fire Blast 13 mana 4.5 112-164 (+234% syn); Wake of Fire 20 mana 13.0 159-181 (+152% syn); Wake of Inferno 6 mana 20.0 543-750/s (+231% syn) |
| 42 | clvl 37 | Wake of Inferno | 5 Vit | Fire Blast 13 mana 4.5 115-168 (+243% syn); Wake of Fire 20 mana 13.0 164-187 (+160% syn); Wake of Inferno 7 mana 20.0 631-848/s (+231% syn) |
| 43 | clvl 38 | Wake of Inferno | 5 Vit | Fire Blast 13 mana 4.5 118-172 (+252% syn); Wake of Fire 20 mana 13.0 169-193 (+168% syn); Wake of Inferno 8 mana 20.0 719-946/s (+231% syn) |
| 44 | Izual NM (clvl ~38) | Wake of Inferno | - | Fire Blast 13 mana 4.5 121-177 (+261% syn); Wake of Fire 20 mana 13.0 174-199 (+176% syn); Wake of Inferno 9 mana 20.0 828-1065/s (+231% syn) |
| 45 | Izual NM 2 (clvl ~38) | Wake of Inferno | - | Fire Blast 13 mana 4.5 124-181 (+270% syn); Wake of Fire 20 mana 13.0 179-204 (+184% syn); Wake of Inferno 10 mana 20.0 936-1184/s (+231% syn) |
| 46 | clvl 39 | Wake of Inferno | 5 Vit | Fire Blast 13 mana 4.5 127-186 (+279% syn); Wake of Fire 20 mana 13.0 184-210 (+192% syn); Wake of Inferno 11 mana 20.0 1045-1303/s (+231% syn) |
| 47 | clvl 40 | Wake of Inferno | 5 Vit | Fire Blast 13 mana 4.5 130-190 (+288% syn); Wake of Fire 20 mana 13.0 189-216 (+200% syn); Wake of Inferno 12 mana 20.0 1153-1422/s (+231% syn) |

Why: Fire Blast 4 + 5 gives the only damage until clvl 12 (V1 Maxroll: Fire Blast 2-5, Burst of Speed 6, Fire Blast 8-11, Wake of Fire from 12); Claw Mastery 1 + Burst of Speed 1 are the prerequisite chain for movement; Wake of Fire 13 by clvl 24, then 20 by 40; Wake of Inferno from clvl 24, 12 by clvl 40; the last 4 Fire Blast points raise all fire traps by +9% each (T). The Maxroll human line: Wake of Fire to 31, Fire Blast to 38, then respec at 38 into Lightning Sentry 15, Death Sentry 9, Charged Bolt Sentry 10 (V1).

## 5. Mercenary

- Act 2 Desert merc (V AS/T): Prayer / Thorns / Prayer, Defiance / Holy Freeze / Defiance, Blessed Aim / Might / Blessed Aim by difficulty. Traps need time to place, so a tanking merc is essential: Defiance (Normal/Hell) or Holy Freeze (NM, slows) (U).
- Act 5 Barbarian has the most life (T: 1,339 at level 80, NM) but his damage is small (V AS).
- No Infinity offline (V `builds-open-questions.md` Q3).

## 6. Gear per act on a budget

| stage | weapon | armor / helm | rest | notes |
|---|---|---|---|---|
| Normal Act 1-2 | start Katar; any 1h weapon or claw; traps do not use the weapon | Stealth (Tal+Eth, +25% FCR) at clvl 17 (V1) | belt from Fara | FCR matters for trap laying speed (U) |
| Normal Act 3-5 | a wand or staff with +trap skills (Assassin class staves/claws: Death's Hand/Guard, V1) | Lore (Ort+Sol, +1 all skills, T), Smoke | rings with FCR/mana | V1 mentions Twitchthroe |
| Nightmare | Heart of the Oak (T) or a class claw with +skills | Chains of Honor (+2 all skills, V Q2) | resists | raw res 60 each |
| Hell | claw with +skills, Heart of the Oak | Enigma (+2 skills, Teleport), Chains of Honor | Harlequin Crest | raw res 125 each |

## 7. Resists and immunities

Targets: Normal 0 needed; Nightmare entry raw 60 fire/cold/lightning (+20 after -40); Hell raw 125 (+25 after -100), cap 175 (V AS). Fade gives resistances for fighting (T Param 10/75), not on the path.

Immunity landscape (T; NM / Hell types):
| act | distinct spawn types | physical imm NM / Hell | fire imm NM / Hell | lightning imm NM / Hell | cold imm NM / Hell |
|---|---|---|---|---|---|
| 1 | NM 55 / Hell 55 | 0 / 2 | 0 / 16 | 0 / 14 | 0 / 11 |
| 2 | NM 57 / Hell 57 | 0 / 4 | 1 / 13 | 0 / 11 | 0 / 13 |
| 3 | NM 57 / Hell 57 | 0 / 2 | 0 / 10 | 0 / 15 | 0 / 17 |
| 4 | NM 20 / Hell 21 | 0 / 0 | 1 / 4 | 1 / 8 | 0 / 4 |
| 5 | NM 158 / Hell 47 | 0 / 4 | 2 / 11 | 4 / 8 | 10 / 21 |

- Hell fire-immune spawn types by act:
  - Act 1: Fallen, Fallen Shaman, Carver, Vile Lancer, Carver Shaman, Vile Archer, Returned Mage, Devilkin, Blood Clan, Devilkin Shaman, Bone Mage, Dark One, Dark Shaman, Rat Man, Arach, Ghoul
  - Act 2: Sand Leaper, Saber Cat, Tomb Creeper, Guardian, Hell Slinger, Burning Dead, Sand Raider, Burning Dead Mage, Invader, Blunderbore, Horror Mage, Desert Wing, Hell Clan
  - Act 3: Swamp Dweller, Drowned Carcass, Fetish Shaman, Flayer Shaman, Soul Killer Shaman, Hell Buzzard, Zakarumite, Flame Spider, Spider Magus, Blood Diver
  - Act 4: Doom Knight, Balrog, Pit Lord, Venom Lord
  - Act 5: Demon Rascal, Moon Lord, Demon Trickster, Succubus, Talic, Death Lord, Vile Witch, Greater Hell Spawn, Arach, Pit Lord

How humans handle it: the Assassin changes damage type by choosing a different trap rather than a different class skill: fire traps hit fire immunes only with synergy from lightning sentries and the Charged Bolt line; the usual human fix is the respec to **Lightning Sentry/Death Sentry** (V1), and **Blade/martial** physical skills for lightning immunes. Our path has Fire Blast only, so every fire-immune pack (T: 16 in Act 1, 13 in Act 2, 10 in Act 3, 4 in Act 4, 11 in Act 5 in Hell) is a skip. **Not recommended in Hell without a second element or a respec** (V1 + T).

## 8. Strengths, weaknesses, deaths by act/difficulty

- Normal Act 1-2 (clvl 1-24): decent ranged damage at clvl 1-11 with only Fire Blast, weak between clvl 12 and 18 until Wake of Fire has points; Andariel's poison and Duriel's cold are the first walls (U).
- Normal Act 3-5: Wake of Fire is the "set and forget" trap (V1); Mephisto and Diablo are tractable; Baal waves in Normal are fine.
- Nightmare: -40 resists with 450-ish life (T clvl 40 value 626); fire immune types appear (T: 1 Act 2, 1 Act 4, 2 Act 5) but are rare; deaths: Mephisto, Izual (U).
- Hell: all elements; the fire line is blocked by 39 fire-immune regular spawn types across Acts 1-3 (16+13+10, T), -100 resists (V AS).
- Typical bot deaths (U, predicted): trap-laying while a pack is already on top; running out of mana mid-pack (traps cost 13-20 each); no merc.

## 9. Three common builds and their trade-offs

| build | skills | pros | cons | no-respec fit |
|---|---|---|---|---|
| Fire traps (this file) | Fire Blast, Wake of Fire, Wake of Inferno, Burst of Speed | fast, ranged-safe, strong early/mid | fire immunes (T), mana | best to NM |
| Lightning / Death Sentry | Shock Web, Charged Bolt Sentry, Lightning Sentry c24, Death Sentry c30 (T prereqs) | top endgame damage (V1), lays 5 sentries on one spot | needs Fire Blast+Shock Web setup (~15 points), lightning immunes (T: 8-15 per Hell act), slow early | respec build (V1 at 38) |
| Martial arts (Phoenix Strike, Dragon Claw/Tail/Flight) | charge-up skills, claws | no mana, physical/elemental charges | melee, needs claws, Dex/Str gear, hard automation | poor |

## 10. When is the character "ready" (bot checks; U unless tagged)

| step | minimum | extra checks |
|---|---|---|
| Den of Evil / Blood Raven | clvl 6-8, Fire Blast 4+, 4 health + 3 mana potions | Burst of Speed optional |
| Andariel | clvl 16-18, Wake of Fire 4+ (clvl 12), life >= 230, 4 mana + 4 health | merc hired |
| Duriel / Act 3 | clvl 21-23, life >= 290, Wake of Fire 10+, cold res >= 20 | pool >= 60 |
| Mephisto | clvl 26-28, life >= 360, Wake of Fire 13+, all res >= 40 | Wake of Inferno 1 (clvl 24) |
| Diablo / Baal | clvl 30-34, life >= 450-520, Wake of Inferno 4+ | all res >= 50-60 |
| Enter Nightmare | clvl >= 38-40, raw res >= 60, life >= 620 (T path 626 at clvl 40), Wake of Fire 20, Wake of Inferno 10+ | 6 mana potions, merc alive |
| Enter Hell | clvl >= 64 (project gate V), raw res >= 125, life >= 1100, a lightning/physical second source (fire immunes are 39 spawn types in Acts 1-3 alone, T) | not ready on this path without respec |

## 11. Open questions

1. Trap count/limits and spacing in 1.14d are S/U (not fetched); read from `skills.tsv` `summon` and Assassin page of AB.
2. How a bot places traps (planner): `docs/classes/assassin.md`.
3. Starting life/mana for all classes are U.

## Shared facts (apply to every class; V = Arreat Summit archive, still served by Blizzard)

- Resist floors by difficulty: Normal 0, Nightmare -40, Hell -100; absolute floor -100; default cap 75 (V `https://classic.battle.net/diablo2exp/basics/resistances.shtml`). Hell therefore needs raw 175 to cap at 75, raw 125 for +25.
- Nightmare: unique/champion damage +75% (Normal +90%, Hell +66%), summons and hirelings deal only 35% damage to act bosses (Normal 50%, Hell 25%), life/mana steal halved (Hell cut by 2/3), freeze/chill/curses halved (Hell cut by 3/4), Static Field cannot take a monster below 33% (NM) / 50% (Hell), monster skill levels +3 (NM) / +7 (Hell), death costs 5% (NM) / 10% (Hell) of the exp gap to the next level, 75% of it restored when the corpse is picked up (V `https://classic.battle.net/diablo2exp/basics/difficulty.shtml`).
- Mercenary aura table, identical in the Arreat page and our tables (V `https://classic.battle.net/diablo2exp/basics/hirelings.shtml`, `hireling.tsv` of this install): Act 2 Desert mercenary Combat = Prayer (Normal) / Thorns (NM) / Prayer (Hell); Defense = Defiance / Holy Freeze / Defiance; Offense = Blessed Aim / Might / Blessed Aim. So Might is an NM-only option; in Hell the Act 2 offense merc runs Blessed Aim. Act 1 Rogue: Fire or Cold Arrow plus Inner Sight (bow). Act 3 Iron Wolf: Fire (Inferno/Fire Ball), Cold (Glacial Spike/Frozen Armor/Ice Blast) or Lightning (Charged Bolt/Lightning), sword + shield, "rarely swing". Act 5 Barbarian: Bash and Stun, one sword. No Act 4 merc. Revival never costs more than 50,000 gold; if the player dies the merc dies too.
- "T" below means a value computed from this install's own game tables (`skills.tsv`, `charstats.tsv`, `hireling.tsv`, `monstats.tsv`, `levels.tsv`, `runes.tsv`; the worker host `~/.local/share/d2bot/cache/data/v2-6ff77ee7fff5/`), V by construction. Skill damage/mana formulas: mana = (mana + lvlmana*(lvl-1)) * 2^manashift / 256 with a floor of minmana; added elemental damage = (EMin + sum of the per-level steps for levels 2-8, 9-16, 17-22, 23-28, 29+) * 2^HitShift / 256, times (1 + synergy%). Cross-checked against Amazon Basin pages (Exploding Arrow mana 4.5+0.5/lvl, Blessed Hammer 4.75+0.25/lvl, Firestorm +23% per Molten Boulder/Fissure level, Bone Spear +7% per Teeth/Bone Wall/Prison/Spirit level, `https://theamazonbasin.com/wiki/index.php?title=<Skill>`). Skills with HitShift 2-4 (Firestorm, Inferno, Fire Wall, Blaze, Arctic Blast, Wake of Inferno) are damage per second per source (x25 frames); poison skills show the total over their duration where resolvable (S).
- Immunity tables are built from `levels.tsv` spawn lists (mon/nmon/umon columns) joined to `monstats.tsv` resists (>=100 means immune); they count distinct regular spawn types per act and difficulty, excluding bosses/super uniques (T). Immunities on unique and champion monsters are rolled on top (enchants and auras), so these are lower bounds for "something nasty is immune". The Act 5 Nightmare spawn lists in `levels.tsv` are unusually long (158 types), so read that row as a rough shape.
- Tag "V1" = taken from the first-pass SME files (`builds-leveling.md`, `builds-gear.md`, `builds-endgame-*.md`), where it was fetched from Maxroll D2R Season 14 guides (D2R text; mechanics match 1.14d except where flagged). I did not re-fetch those pages. Ladder-only runewords (Spirit, Insight, Fortitude, Infinity, Dragon, Dream, Edge, Faith, Harmony, Ice, Brand, Death, Destruction, Last Wish, Lawbringer, Oath, Obedience, Phoenix, Pride, Rift, Grief, Voice of Reason, Wrath) are not craftable in vanilla offline 1.14d (V `builds-open-questions.md` Q3); Enigma, Chains of Honor, Stone, Memory, Delirium, Lore, Rhyme, Smoke, Stealth, Heart of the Oak, Beast, Fury, Honor, Doom, Hand of Justice, Ancient's Pledge, Bone, Peace, Treachery, Duress, Gloom are available (T: server flag empty).

