# AREAMON: where a leveling Paladin should farm (Act 1 Normal, XP per kill, per clear, per minute)

> status: research, 2026-11-14, agent decomp18 (TASK-408). Offline: no game, no d2d slot, nothing posted. Extends xp-gold-trace.md (decomp14) from 9 to 36 Act 1 levels and from "per kill" to "per clear, per minute and per clvl band". Tags: V run (game.exe code under the emulator, `$D2_EMU_DIR` image of docs/emu-image.md), D (patched 1.14d tables via mpq-cat), S (an earlier trace of ours, named), I inferred, U unverified. Consumer: the plan table (internal/plandata/progression-normal.json) and docs/levelplan.md bands. Sources: docs/research/xp-gold-trace.md, docs/research/monster-stats-trace.md, docs/probes/hour1-analysis-2026-11-12.md, docs/probes/hour4-analysis-2026-11-14.md (branch agent/hour4), docs/levelplan.md.

## Answer in eight lines

1. **Per minute at equal safety, Blood Moor is the worst field of the first six levels.** A full Blood Moor clear is 97 units and 2,128 XP at clvl 1 to 5 (about 156 XP per minute at its observed rate); Cold Plains is 198 units and 5,915 XP (about 435 per minute): 2.8x. Stony Field (625), Dark Wood (1,020), Black Marsh (960) follow (V run population, I rate). The plan keeps the hero in Blood Moor until clvl 4 and Den until 6; hour 4 spent 51 percent of its time in Blood Moor (S, hour4 section 1.1).
2. **The plan's rows are in the wrong order for XP.** Cave 1 (385 XP per min at clvl 3, entry 6) pays less than Stony Field (625, entry 7 melee) and Dark Wood (1,022, entry 9); Dark Wood and Black Marsh, both mlvl 5 to 6, pay the same 14 minutes of work 1.6x what Stony pays. The dungeon floors with mlvl 7 to 8 (Tower Cellars, Pit 1/2) and Tamoe Highland (mlvl 7.7, 1,050 to 1,209 per min) are the best fields of clvl 6 to 10 and are not in the farm rows (Tamoe has a row without the farm flag, cellars none).
3. **From clvl 11 the Catacombs (levels 2 and 3) and the Jails are the best Act 1 fields** (Catacombs 3: 1,629 XP per min at clvl 12 and 14 minutes, 18,400 XP per clear; Catacombs 2: 1,265; Jail 2: 1,020; Barracks 1,018): the plan enters Catacombs 2 at 13 and 3 at 14, two clvl later than the data support (U: survival).
4. **Per-clear numbers make the horizon concrete.** One clear of Dark Wood is 13,900 XP at clvl 4 to 8: that is clvl 5 to 6 in one area. Clvl 9 needs 44,400 XP; model route "best safe field" reaches 53,700 XP (clvl 9) in 60 minutes, the plan rows 13,100 XP (clvl 5): the same model on the plan rows reproduces hour 4 (13,883 XP, clvl 5) and hour 1 (17,032, clvl 6), so the model is not generous to itself (I, section 5).
5. **The caveat that matters: survival is not modeled.** The "best safe" route puts a clvl 3 to 5 Paladin in Stony Field and Dark Wood (monster level 3.6 and 5.2, packs of Fallen with shamans, Corrupt Rogue archers, Quill Rats, Goatmen at Black Marsh); whether the bot lives there with 3 to 5 potions is U (the plan entry levels 7 to 9 are the SME's S numbers). The proposed band table is therefore conditional (section 8).
6. **XP per kill is capped, not the formula.** Every area's per-kill XP falls once clvl is 6 or more above the mean monster level (Blood Moor 22 at clvl 3, 13 at 8, 0.4 at 12; V run, same level-gap table as xp-gold-trace.md, 119,790 of 119,790 grid lines). A field "expires" 6 clvl above its mean level and is worth less than 5 percent 10 above.
7. **No d2d deviation proven; no PR candidate.** Population counts per area (d2d's rooms against game.exe's) were not compared (D1, U, a differential can be built).
8. **Not done:** the 150-seed d2d `debug monsters` count (needs the fork build; only recipe, section 9) and any live check; 12 seeds of game.exe's own population per area instead of 150 (V run, enough for means: units per clear range shown).

## Q1..: questions and answers

| id | question | answer | tag |
|---|---|---|---|
| Q1 | which Levels.txt fields set the spawn | MonDen (density; the Normal and Nightmare columns hold the same number in every Act 1 row), MonUMin/MonUMax (uniques per level; Blood Moor 0, Cold Plains 1, Stony Field to Tamoe 1 to 2, the Cellars 1, Barracks to Catacombs 1 to 2), MonWndr (wandering spawns: 1 everywhere except Den of Evil and Tristram), MonSpcWalk (only Jail 1-3), mon1..mon10 and the NumMon count (3 species in every Act 1 level, 4 to 6 in the fields and Monastery), MonLvl1Ex (level per difficulty) | D |
| Q2 | monster level per area (Normal) | MonLvl1Ex: Blood Moor 1, Den 1, Cold Plains 2, Cave 1 / 2 2, Burial 3, Crypt / Mausoleum 3, Stony 4, UP 1 / 2 4, Dark Wood 5, Hole 1 / 2 5, Black Marsh 6, Tristram 6, Tamoe 8, Pit 7, Cellars 7, Monastery Gate 8, Outer Cloister / Barracks 9, Jails / Inner Cloister 10, Cathedral / Catacombs 1-2 11, Catacombs 3-4 12. The units' actual level is the area level except bosses and minions (leveladd), so the mean in the table sits above it | D + V run |
| Q3 | how many monsters at full clear | table section 3, units per clear over 12 seeds of game.exe's real room population (`FUN_00573cb0` writes under the emulator, ms_spawn.py), no walkable-area estimate needed: the rooms the DRLG places are the rooms populated | V run (12 seeds, not 150) |
| Q4 | pack size | MonStats MinGrp/MaxGrp per species (1-3 to 3-5 at Normal Act 1), summoned by the room fill; fallen and shamans carry minion rows (Fallen with 2-6 party members); counted inside the units above | D (counts in the V run) |
| Q5 | XP per kill by area and clvl | sections 3, 4; unit's stat 0xd (MonLvl XP x MonStats Exp %) then the level-gap table | V run |
| Q6 | time per clear | I: sections 4 and 5 | I |
| Q7 | xp/h | section 6 | V run (XP) x I (time) |
| Q8 | threat | section 7 | D |

## 2. Method

1. `tools/emu/am_env.sh table levels|monstats|monlvl|experience` reads the patched 1.14d tables (mpq-cat, D). `tools/emu/am_batch.sh OUT 2 12 3` runs upstream's room population of every Act 1 level for seeds 1 to 12 (Normal, expansion byte set; ids 2 to 19, 21 to 38; `ms_spawn.py`, three seeds per call, 73 s per seed for all 36 levels at nice 19 on a busy worker host). Level 20 (the tower surface) has no monster row; Inner Cloister has none in 3 of 12 seeds and Tower Cellars 1 and 4 in one seed each (counted per run).
2. `tools/emu/am_model.py POP.txt table|csv|safe N|sim H` turns the unit lines (level stat 0xc, experience stat 0xd, flags) into per-area counts and XP at clvl 1, 3, 5, 8, 12, 16 with the award function of xpg_model.py. The cross-check against xp-gold-trace.md section 3 (same method, same seeds, different area list) holds on every shared row (Cold Plains 29.9 at clvl 3, Stony 55.8, Dark Wood 60.1, Black Marsh 66.9, Den 37.0, Cave 1 53, Cave 2 67) (V run).
3. Time model (I). Overworld (ids 2 to 7): kills per minute = 14.6 x units / units(Cold Plains), clamped 6..19: Cold Plains 14.6 (windows of 14 to 19 per minute in hour 4 section 1.3), Blood Moor 7.1 (7.5 over hour 4), and every field comes out near 14 minutes per clear, which is the logged sweep time of a Blood Moor pass in hour 4 (09:55 to 10:09). Dungeon levels: minutes = T0 + units / 9.5, T0 = 4 for the 200 x 200 levels (stairs walk), 1.5 for the others; Den: 4 + 77 / 9.5 = 12 minutes against 14 logged in hour 1. 1 minute per area change. These are the weakest numbers of the doc: U for any dungeon below Cave 1 (no bot hour has been there).
4. Re-clears pay half (hour 1: Blood Moor re-clear 171 against 337 per minute fresh, S).

## 3. The table (V run population, I time; clvl columns use the exact level-gap award)

| lid | area | units (min-max) | mlvl mean | unique / minion per clear | clear min | XP per clear at clvl 1 / 5 / 8 / 12 / 16 | XP per kill at clvl 3 / 8 / 12 | XP per min at clvl 3 / 8 / 12 |
|---|---|---|---|---|---|---|---|---|
| 2 | Blood Moor | 97 (71-119) | 1.0 | 0.0 / 0.0 | 14 | 2128 / 2128 / 1300 / 42 / 42 | 22 / 13 / 0 | 156 / 96 / 3 |
| 3 | Cold Plains | 198 (115-270) | 1.6 | 2.1 / 5.4 | 14 | 5915 / 5915 / 4591 / 690 / 150 | 30 / 23 / 4 | 435 / 338 / 51 |
| 4 | Stony Field | 152 (100-203) | 3.6 | 2.8 / 13.8 | 14 | 7759 / 8506 / 8010 / 4706 / 1292 | 56 / 52 / 31 | 625 / 589 / 346 |
| 5 | Dark Wood | 231 (161-290) | 5.2 | 3.5 / 9.2 | 14 | 12770 / 13899 / 13899 / 10027 / 1987 | 60 / 60 / 43 | 1022 / 1022 / 737 |
| 6 | Black Marsh | 199 (141-363) | 5.7 | 2.2 / 8.8 | 14 | 11353 / 13327 / 13327 / 10565 / 2560 | 66 / 67 / 53 | 961 / 980 / 777 |
| 7 | Tamoe Highland | 194 (158-257) | 7.7 | 2.2 / 7.6 | 14 | 9410 / 16042 / 16445 / 16151 / 8295 | 74 / 85 / 83 | 1050 / 1209 / 1188 |
| 8 | Den of Evil | 77 (60-97) | 1.4 | 1.0 / 5.0 | 12 | 2839 / 2839 / 2224 / 459 / 87 | 37 / 29 / 6 | 235 / 184 / 38 |
| 9 | Cave 1 | 124 (83-159) | 2.3 | 2.9 / 8.1 | 17 | 6346 / 6538 / 5767 / 2590 / 554 | 53 / 47 / 21 | 385 / 339 / 152 |
| 10 | Underground Pass. 1 | 195 (124-277) | 4.6 | 2.7 / 6.0 | 24 | 9953 / 10654 / 10450 / 7059 / 1286 | 55 / 54 / 36 | 435 / 427 / 288 |
| 11 | Hole 1 | 124 (87-154) | 5.2 | 2.0 / 5.1 | 17 | 7167 / 7783 / 7783 / 5632 / 1099 | 63 / 63 / 45 | 455 / 455 / 329 |
| 12 | Pit 1 | 148 (82-201) | 7.4 | 1.8 / 5.8 | 20 | 7570 / 11886 / 12054 / 12054 / 5276 | 72 / 82 / 82 | 546 / 618 / 618 |
| 13 | Cave 2 | 46 (38-56) | 2.4 | 2.8 / 7.8 | 6 | 3054 / 3126 / 2875 / 1599 / 237 | 67 / 62 / 34 | 488 / 449 / 250 |
| 14 | Underground Pass. 2 | 48 (41-58) | 3.8 | 3.1 / 7.3 | 6 | 3095 / 3980 / 3809 / 3246 / 1224 | 84 / 80 / 68 | 612 / 586 / 499 |
| 15 | Hole 2 | 36 (25-48) | 6.0 | 2.2 / 7.2 | 5 | 3316 / 4273 / 4273 / 3826 / 1338 | 118 / 118 / 105 | 806 / 806 / 722 |
| 16 | Pit 2 | 50 (40-62) | 7.9 | 2.6 / 7.2 | 7 | 2432 / 5865 / 5897 / 5897 / 3534 | 95 / 119 / 119 | 703 / 880 / 880 |
| 17 | Burial Grounds | 38 (20-55) | 2.2 | 1.0 / 0.0 | 6 | 1522 / 1677 / 1370 / 241 / 206 | 42 / 36 / 6 | 294 / 249 / 44 |
| 18 | Crypt | 29 (12-49) | 3.0 | 1.9 / 8.1 | 7 | 2512 / 2512 / 2356 / 1108 / 114 | 86 / 81 / 38 | 354 / 332 / 156 |
| 19 | Mausoleum | 26 (15-51) | 2.8 | 1.2 / 5.2 | 7 | 2036 / 2036 / 1883 / 824 / 92 | 79 / 73 / 32 | 304 / 281 / 123 |
| 21 | Tower Cellar 1 | 20 (7-41) | 8.0 | 1.5 / 5.5 | 6 | 1138 / 3384 / 3384 / 3384 / 2221 | 127 / 166 / 166 | 425 / 555 / 555 |
| 22 | Tower Cellar 2 | 16 (5-34) | 7.9 | 1.3 / 3.8 | 6 | 913 / 2536 / 2536 / 2536 / 1615 | 121 / 156 / 156 | 346 / 445 / 445 |
| 23 | Tower Cellar 3 | 21 (8-34) | 7.7 | 1.5 / 3.1 | 6 | 1163 / 2774 / 2774 / 2774 / 1636 | 107 / 133 / 133 | 360 / 447 / 447 |
| 24 | Tower Cellar 4 | 16 (6-26) | 8.2 | 1.1 / 5.4 | 6 | 894 / 3031 / 3031 / 3031 / 2091 | 142 / 190 / 190 | 396 / 532 / 532 |
| 25 | Tower Cellar 5 | 33 (16-62) | 7.9 | 1.2 / 6.7 | 5 | 1485 / 4311 / 4647 / 4647 / 3358 | 85 / 143 / 143 | 567 / 948 / 948 |
| 26 | Monastery Gate | 13 (2-29) | 8.4 | 0.4 / 1.2 | 3 | 615 / 1418 / 1487 / 1487 / 964 | 83 / 111 / 111 | 384 / 513 / 513 |
| 27 | Outer Cloister | 49 (22-83) | 8.6 | 1.3 / 1.6 | 7 | 1645 / 4472 / 4859 / 4859 / 3222 | 70 / 99 / 99 | 516 / 736 / 736 |
| 28 | Barracks | 100 (72-143) | 9.2 | 8.2 / 10.7 | 14 | 2948 / 11930 / 14758 / 14758 / 11754 | 75 / 148 / 148 | 517 / 1018 / 1018 |
| 29 | Jail 1 | 63 (41-81) | 10.5 | 3.2 / 6.9 | 11 | 942 / 6527 / 9654 / 9755 / 8716 | 54 / 154 / 156 | 317 / 911 / 920 |
| 30 | Jail 2 | 90 (61-140) | 10.5 | 5.5 / 8.2 | 13 | 1326 / 8509 / 13282 / 13688 / 12229 | 52 / 148 / 153 | 345 / 991 / 1021 |
| 31 | Jail 3 | 89 (39-150) | 10.3 | 4.5 / 5.8 | 13 | 1295 / 8406 / 11300 / 11412 / 9906 | 54 / 127 / 129 | 361 / 850 / 858 |
| 32 | Inner Cloister | 8 (1-15) | 10.6 | 0.2 / 1.2 | 2 | 78 / 668 / 991 / 991 / 896 | 44 / 129 / 129 | 147 / 431 / 431 |
| 33 | Cathedral | 36 (19-50) | 10.9 | 2.0 / 2.2 | 5 | 276 / 3070 / 4362 / 4874 / 4580 | 42 / 120 / 134 | 288 / 823 / 920 |
| 34 | Catacombs 1 | 78 (51-93) | 11.0 | 4.7 / 9.2 | 12 | 611 / 6975 / 11816 / 12221 / 11610 | 39 / 152 / 157 | 247 / 969 / 1002 |
| 35 | Catacombs 2 | 96 (53-134) | 11.2 | 4.8 / 14.8 | 14 | 656 / 8513 / 16754 / 17885 / 17355 | 37 / 174 / 186 | 252 / 1188 / 1268 |
| 36 | Catacombs 3 | 69 (54-81) | 12.8 | 5.1 / 11.3 | 11 | 327 / 5124 / 14978 / 18407 / 18398 | 18 / 216 / 266 | 111 / 1325 / 1629 |
| 37 | Catacombs 4 | 50 (40-65) | 10.9 | 1.3 / 1.1 | 7 | 352 / 4726 / 6356 / 6612 / 6234 | 41 / 127 / 132 | 298 / 935 / 972 |
| 38 | Tristram | 56 (30-74) | 6.2 | 5.8 / 7.6 | 7 | 4950 / 7055 / 7055 / 6392 / 2466 | 123 / 127 / 115 | 921 / 953 / 864 |

CSV with every column (units by kind, XP per clear and per kill for clvl 1, 3, 5, 8, 12, 16, XP per hour if the same area were repeated): docs/research/areamon-table.csv (regenerate: `tools/emu/am_model.py POP.txt csv`). "XP per hour if repeated" is `60 x XP per clear / (clear + 1 minute)`; a game's areas do not respawn, so it only ranks fields, it is not a plan.

Reading: (a) unique and minion packs are 5 to 15 units per level and decide the richness: Catacombs 2 has 14.8 minions and 4.8 uniques per clear, Barracks 8 uniques and 10.7 minions (a Barracks clear is 14,758 XP at clvl 8 although mlvl 9.2). (b) Burial Grounds is a small level (38 units, 1,522 to 1,677 XP at clvl 1 to 5; Blood Raven is not in the room population, her 181 XP is a separate kill; xp-gold-trace.md): never a farm. (c) The second floors (Cave 2, Underground Passage 2, Hole 2, Pit 2) are 36 to 50 units each, 5 to 7 minutes, and pay 1.3 to 2.2x the first floor per kill (a boss pack in a small level: Hole 2 118 XP per kill); reaching them costs the first floor's stairs (not in the time model). (d) The Tower Cellars and Monastery levels pay 130 to 190 per kill but have 5 to 20 units per level; cellar 5 (33 units, 4,647 XP at clvl 8) carries the tower's boss. (e) Tristram (56 units, 6.2 mean level, 5.8 uniques) pays 920 per min at clvl 3 to 8, but it is a quest stop (levelplan item 3).

## 4. Where a clvl band's best fields are (rank by XP per minute, mean mlvl <= clvl + 1, no Tristram, no Burial Grounds)

`am_model.py POP.txt safe 1` (the clvl columns nearest to the six computed ones):

| clvl | best by XP/min (clear min; XP/clear) |
|---|---|
| 1 to 2 | Cold Plains 436 (14; 5,915), Den 235 (12; 2,839), Blood Moor 157 (14; 2,128) |
| 3 to 4 | Stony Field 627 (14; 8,506), Cold Plains 436; the second floors Cave 2 490, UP 2 612 (after the first floor) |
| 5 to 6 | Dark Wood 1,024 (14; 13,899), Black Marsh 982 (14; 13,327), Hole 2 803 (5; 4,273) |
| 7 to 10 | Tamoe Highland 1,212 (14; 16,445), Dark Wood 1,024, Barracks 1,019 (14; 14,758), Black Marsh 982, Jail 2 990 |
| 11 to 14 | Catacombs 3 1,629 (11; 18,407), Catacombs 2 1,265 (14; 17,885), Tamoe 1,190, Jail 2 1,020 |
| 15 to 18 | Catacombs 3 1,628, Catacombs 2 1,227, Catacombs 1 953, Catacombs 4 923, Jail 2 911 |

With slack 2 (mean mlvl <= clvl + 2) the same fields come one clvl earlier. The bands above are rankings of XP per minute only; the survival side is U (section 8).

## 5. The route model against two logged hours (I)

`am_model.py POP.txt sim 60`: minute by minute, one area at a time, each area full-rate for its clear time, half after; 1 minute per switch. "Plan rows" = the farm rows of progression-normal.json for a melee Paladin in file order (Blood Moor 1-4, Den 2-6, Cold Plains 4-8, Burial 8-10, Cave 1 6-9, Stony 7-12, Dark Wood 9-13, Black Marsh 10-14, Cave 2 10-15, Crypt 10-17); the first eligible fresh row wins.

| route | XP at 60 min | clvl | at 120 min |
|---|---|---|---|
| plan rows | 13,117 | 5 | 32,974 (clvl 8) |
| observed hour 4 (S) | 13,883 | 5 | |
| observed hour 1 (S) | 17,032 | 6 | |
| best XP/min, mean mlvl <= clvl + 1 | 49,401 | 9 | 115,416 (clvl 13) |
| best XP/min, mean mlvl <= clvl + 2 | 53,723 | 9 | 119,430 (clvl 13) |

The plan row simulation is within 6 percent of hour 4 without any tuning: the plan route is what costs (Blood Moor 4 clvl, Den, Cold Plains, then no fresh field open at clvl 5: Cave 1 enters at 6, Burial at 8, so it re-clears at half rate; a gap in the plan, B2). The "best" route is an upper bound that assumes the bot clears Dark Wood at clvl 4 to 5 with the Cold Plains kill rate (U).

## 6. XP per hour by area and band (the brief's table, repeating-area form)

The `XP/h if the area is repeated` columns of areamon-table.csv (clvl 1 / 3 / 5 / 8 / 12 / 16, thousands): Blood Moor 8.8 / 8.8 / 8.8 / 5.4 / 0.2 / 0.2; Cold Plains 24 / 24 / 24 / 19 / 2.8 / 0.6; Stony Field 32 / 35 / 35 / 33 / 19 / 5.3; Dark Wood 53 / 57 / 57 / 57 / 41 / 8.2; Black Marsh 47 / 54 / 55 / 55 / 44 / 10.5; Tamoe 39 / 59 / 66 / 68 / 67 / 34; Barracks 15 / 39 / 62 / 77 / 77 / 61; Jail 2 7.6 / 27 / 49 / 76 / 79 / 70; Catacombs 3 2.4 / 9 / 37 / 108 / 133 / 133. (V run population, I time.) A real hour is a chain of clears: section 5.

## 7. Threats (D, from MonStats; which skill or gear matters)

- Undead (Skeleton, Zombie, Wraith, Skeleton Archer, Skeleton Mage, Vampire): poison resist 50 percent (Wraith also 20 to 30 percent physical damage reduction), Zombies and Skeletons are slow melee; Skeleton Mage fire / lightning variants resist their own element by 30 to 40 percent and raise skeletons (`SkeletonRaise`) in Pit, Tamoe, Monastery, Barracks, Jail 1-3, Inner Cloister.
- Fallen packs with a Fallen Shaman: the shaman casts `Resurrect` and carries a fire resist (25 to 40); kill the shaman first (Cold Plains, Black Marsh, Den, Outer Cloister, Catacombs 1 to 4, Cathedral).
- Ranged: Corrupt Archers and Skeleton Archers (Stony Field, Dark Wood, UP, Hole, Pit, Barracks, Jail, Inner Cloister), Quill Rats (spike volleys: Blood Moor, Dark Wood, Tamoe, Outer Cloister), `bighead` (UP, Hole, Monastery Gate, Cathedral, Catacombs), a fire / lightning ranged caster with cold-resist 40 to 70 in the later rows.
- Fast or heavy melee: Goatmen (Stony Field, Black Marsh, Tower Cellars, Barracks, Jails, damage up to 160), Brutes, Corrupt Lancers; Wraiths in the Cellars and Jails; Arach (Catacombs 2) and Fetish (Catacombs 1) summon spiders / skeletons.
- Cold Plains is the first field with Corrupt Rogues / Lancers and Brutes (ranged and melee mix); Blood Moor only zombies, Fallen and Quill Rats, the reason for the low XP is the mlvl 1.
- Immunities: none in Act 1 Normal (V read of Res columns: every value below 100; xp-gold-trace.md and docs/research/monster-resist-trace.md for the rules).

## 8. Proposed band table (conditional on survival, U) and comparison

| band (clvl) | plan today | SME (S, levelplan) | model best XP/min (I) | proposed (this doc) |
|---|---|---|---|---|
| 1-2 | Blood Moor | Blood Moor 1-4 | Cold Plains 436, Den 235, Blood Moor 157 | Blood Moor to clvl 2 only for the Den quest (Den 2-4 once: 2,839 XP and a skill point, 235 per min), then Cold Plains from clvl 3 (4 for casters) |
| 3-4 | Cold Plains (4-8) | Cold Plains 4-8 | Stony Field 627, Cold Plains 436 | Cold Plains 3-4, Stony Field 4-6 (mlvl 3.6: U survival, melee first) |
| 5-6 | Cold Plains, Cave 1 at 6 | Burial 5-10, Cave 1 6-9 | Dark Wood 1,024, Black Marsh 982 | Dark Wood 5-8 (needs entry from Stony Field / Black Marsh), Black Marsh 6-9; Cave 1 only as the Cave 2 route |
| 7-10 | Stony 7-12, Dark Wood 9-13 | Stony 7-11, Dark Wood 8-12 | Tamoe 1,212, Dark Wood 1,024, Barracks 1,019 | Dark Wood to 9, Tamoe Highland 7-11 (add a farm row), Barracks / Jail 2 from 9 (on the Andariel spine, Outer Cloister is a pass-through: 99 per kill, 49 units) |
| 11-14 | Black Marsh to 14, Cave 2, Crypt 10-17 | Black Marsh 10-14, Forgotten Tower 12-18, Jail 1 14-20 | Catacombs 3 1,629, Catacombs 2 1,265 | Catacombs 2 and 3 from 11 / 12 (plan 13 / 14), Jail 2, Tamoe; Black Marsh and Cave 2 retire (53 and 34 per kill at 12) |
| 15-18 | Catacombs 2-4, Jail 1 | Jail 1 14-20 | Catacombs 3, 2, 1, 4 | Catacombs 3 and 2 first, then Jails; Catacombs 4 is the Andariel door (the 50-unit level pays 923) |

Conditions: every proposed entry is "mean mlvl <= clvl + 1", the same proxy for safety as the plan's own entry levels (about mlvl + 2 to 3 for casters): U. Moving Stony Field and Dark Wood 1 to 3 clvl earlier needs a live drill that the Paladin with the kit survives them (docs/first-hour.md tiers); until then the table is advice.

## 9. Deviations (D1..) d2d vs game.exe

| id | game.exe | d2d | tag | differential |
|---|---|---|---|---|
| D1 | number of monsters (and uniques / minions) per level from the room population of each Act 1 area | not compared: monster-stats-trace.md compared stat rows, not the population counts | U | build: `ms_d2d_test` (ms_build.sh) on the same 36 levels and seeds, count units per level vs `ms_spawn.py` here; or the lab verb below |

Nothing else proven. Level gap, XP per unit and gold are those of xp-gold-trace.md (no deviation).

## 10. Bot findings (B1..), described, NOT changed

| id | where | what | tag |
|---|---|---|---|
| B1 | internal/plandata/progression-normal.json rows Blood Moor `leave_clvl` 4, Den 2-6 | at clvl 1 to 5 Blood Moor pays 156 per min against 435 in Cold Plains; a run in Blood Moor until clvl 4 costs the hour (hour 4: 51 percent in Blood Moor) | V run, I rate |
| B2 | same file, order of the farm rows | at clvl 5 no fresh field is open (Cave 1 enters 6, Burial 8): the plan re-clears at half rate (model: 5 clvl at minute 40 to 60 of the plan route) | I |
| B3 | same file: Cave 1 6-9 vs Stony 7-12 vs Dark Wood 9-13 | Cave 1 pays 385 per min at clvl 3 to 8 and 17 minutes, Stony 625 and 14, Dark Wood 1,022 and 14 | V run, I |
| B4 | same: Tamoe Highland has no `farm` flag, Tower Cellars, Barracks, Jail 2-3 and Catacombs 1 have no row | the best XP fields of clvl 6 to 12 are missing from the farm list (U: some are route-gated) | V run, I |
| B5 | `internal/bot/recoveryxp.go:44` `areaXPPerKill` / `areaXPTable` | per-kill mean of plain monsters only (xp-gold-trace.md B2); this doc's "all" column is the right input, plus the clear time for a XP per minute rank | V run |
| B6 | `internal/bot/recoveryxp.go:11` `recoveryXPTier` (2,3,4,5,6,9,13) | Cave 2 (13) is a 46-unit second floor, 6 minutes; tier lacks Tamoe, Barracks and the Catacombs | I |

## 11. PR candidates

No PR candidate: no d2d deviation proven (D1 is U).

## 12. Drill recipe for the fork verbs (run later, not here; docs/forkdbg.md, docs/d2d-fork.md, docs/forkcheats.md)

- X1 (D1): lab game, new Normal hero, walk (or `debug warp`) to Blood Moor, Cold Plains, Stony Field, Dark Wood and Cave 1; `debug monsters` after each full sweep, count units, uniques and minions; 150 games or seeds need the headless d2d build (devp6 image) driven by the lab seed verb; expect Cold Plains 198 (115-270), Stony Field 152, Dark Wood 231, Black Marsh 199, Cave 1 124 (this doc's means).
- X2: kill 100 Dark Wood monsters at clvl 5 and 100 Cold Plains monsters at clvl 3; XP per kill 60 and 30 (V run means; observe the stat 13 delta), the pack kills 130 to 310 each.
- X3: time of a full clear: Cold Plains, Dark Wood at the bot's kill rate; the model needs 14 minutes; log `minute` kill counters (hour 4 section 1.1).
- X4 (survival, U): a hidden-instance drill that the Paladin with the standard kit clears Stony Field at clvl 3 and 4: potions used and deaths.

## 13. Matrix cell updates (proposed text)

- docs/levelplan.md area band table: add rows Tamoe Highland (SME none; model 7-11 at 1,190 to 1,212 XP per min), Barracks 9-12 (1,018), Catacombs 2/3 11-18 (1,265 / 1,629), Tower Cellars (small, 5 to 33 units); mark the rows Cave 1 and Cave 2 "below Stony / Dark Wood per minute" (this doc section 3).
- docs/research/xp-gold-trace.md section 3 gains 27 areas by reference to this doc.

## 14. Dead ends and limits

- 12 seeds, not 150: 150 would take about 3 hours of worker time at 73 s per seed for the 36 levels; 12 seeds give the unit means within the min-max ranges in the table (singular seeds vary by 40 percent in Cold Plains and the Cathedral).
- The time model is the weakest part (calibrated on Blood Moor, Cold Plains and Den; dungeons beyond Cave 1 are extrapolated). The walkable area was not measured: the populated rooms already encode it.
- Inner Cloister: 0 units in 3 of 12 seeds, 8 on average: not a field. Tristram is a quest.
- Champions (flag 4) never appeared in Normal in these populations (0.0 per clear); XP of uniques and minions is inside the per-unit rows.
- The threat table is read from MonStats columns only; skill-level damage, hit chance and blocking were not simulated.

## Tools

`tools/emu/am_env.sh` (tables and emulator launcher on the worker host), `tools/emu/am_batch.sh` (population batches), `tools/emu/am_model.py` (table, csv, safe, sim, bands). The population input (about 1 MB) stays on the worker host (`~/d2d-lab/am18/pop.txt`), it is not committed.
