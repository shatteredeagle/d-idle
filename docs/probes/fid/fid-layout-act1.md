# fid-layout-act1 (TASK-80, matrix row MA04): layouts of the other Act 1 areas, d2d against game.exe

> status: finished 2026-11-09 by agent decomp8 (Sonnet). Result: **0 differences in every Act 1 level of Levels.txt (ids 5, 6, 7, 9 to 39) on current upstream main 83fc7ba1: tiles, preset units, room seeds and the populated-room lists equal for map seeds 1..30, and for the two 32-bit seed ranges (10 + 10 seeds) of the earlier trace.** 34 levels x 4 kinds x 30 seeds = 4,080 seed comparisons plus 13,497 populated rooms, and 340 range comparisons plus 9,184 rooms, all equal. AC #1 and #2 proven; AC #3 (same-save first-load list on a pinned seed, scenario 8) NOT done (needs a saved character or a live game, U); AC #4 recorded below (the Lut Gholein mismatch is gone on main, in the kinds compared). Nothing posted, no PR candidate.

Tags: V run (game.exe's generator and population pass in the emulator, upstream's `diff_drlg.py`, against the real `drlg-dump` built from upstream main 83fc7ba1), V read, D, I, U. Worker host. Scripts: `tools/emu/d8_setup.sh`, `d8_build.sh` (scratch tree and build), `d8_ss.sh` (one level), `d8_ss_act1.sh` (all levels, seeds 1-30), `d8_ss_ranges.sh` (the two 32-bit ranges), `d8_dump.sh` (is a compare non-empty), `d8_more.sh` (Lut Gholein). Method of docs/research/seedspawn-trace.md section 2, which covered levels 1, 2, 3, 4 and 8 only.

## Q1. Do the other Act 1 levels equal game.exe seed for seed? Yes, V run

Kinds of `diff_drlg.py`: `tiles` (rooms and tile picks), `units` (preset units after game.exe brought the rooms up), `seeds` (each room's seeds), `monsters` (game.exe's population pass `FUN_0052d0f0` per room, class/position/mode/leader, compared as populated rooms and as seeds). Levels are Levels.txt ids (the `Name` column; the in-game name in brackets where it is usual, I). Maps seeds 1..30; "ranges" are 1967900452..61 and 4062638700..09 (the two map ids seen live in joined games, seedspawn-trace.md).

| id | Levels.txt name | tiles | units | seeds | monsters: populated rooms equal | ranges (tiles, units, seeds, monsters) | differ |
|---|---|---|---|---|---|---|---|
| 5 | Wilderness 4 (Dark Wood) | 30/30 | 30/30 | 30/30 | 1,061 of 1,061 | 20/20 (10 + 10) each | 0 |
| 6 | Wilderness 5 (Black Marsh) | 30/30 | 30/30 | 30/30 | 1,039 of 1,039 | 20/20 (10 + 10) each | 0 |
| 7 | Wilderness 6 (Tamoe Highland) | 30/30 | 30/30 | 30/30 | 1,237 of 1,237 | 20/20 (10 + 10) each | 0 |
| 9 | Cave 2 | 30/30 | 30/30 | 30/30 | 883 of 883 | 20/20 (10 + 10) each | 0 |
| 10 | Cave 3 | 30/30 | 30/30 | 30/30 | 1,139 of 1,139 | 20/20 (10 + 10) each | 0 |
| 11 | Cave 4 | 30/30 | 30/30 | 30/30 | 744 of 744 | 20/20 (10 + 10) each | 0 |
| 12 | Cave 5 | 30/30 | 30/30 | 30/30 | 745 of 745 | 20/20 (10 + 10) each | 0 |
| 13 | Cave 2 Treasure | 30/30 | 30/30 | 30/30 | 193 of 193 | 20/20 (10 + 10) each | 0 |
| 14 | Cave 3 Treasure | 30/30 | 30/30 | 30/30 | 175 of 175 | 20/20 (10 + 10) each | 0 |
| 15 | Cave 4 Treasure | 30/30 | 30/30 | 30/30 | 122 of 122 | 20/20 (10 + 10) each | 0 |
| 16 | Cave 5 Treasure | 30/30 | 30/30 | 30/30 | 157 of 157 | 20/20 (10 + 10) each | 0 |
| 17 | Graveyard (Burial Grounds) | 30/30 | 30/30 | 30/30 | 309 of 309 | 20/20 (10 + 10) each | 0 |
| 18 | Crypt 1 A | 30/30 | 30/30 | 30/30 | 223 of 223 | 20/20 (10 + 10) each | 0 |
| 19 | Crypt 2 A (Mausoleum) | 30/30 | 30/30 | 30/30 | 218 of 218 | 20/20 (10 + 10) each | 0 |
| 20 | Tower 2 | 30/30 | 30/30 | 30/30 | 0 of 0 | 20/20 (10 + 10) each | 0 |
| 21 | Crypt 3 A | 30/30 | 30/30 | 30/30 | 118 of 118 | 20/20 (10 + 10) each | 0 |
| 22 | Crypt 3 B | 30/30 | 30/30 | 30/30 | 86 of 86 | 20/20 (10 + 10) each | 0 |
| 23 | Crypt 3 C | 30/30 | 30/30 | 30/30 | 117 of 117 | 20/20 (10 + 10) each | 0 |
| 24 | Crypt 3 D | 30/30 | 30/30 | 30/30 | 80 of 80 | 20/20 (10 + 10) each | 0 |
| 25 | Crypt 3 E | 30/30 | 30/30 | 30/30 | 178 of 178 | 20/20 (10 + 10) each | 0 |
| 26 | Monastery (Gate) | 30/30 | 30/30 | 30/30 | 118 of 118 | 20/20 (10 + 10) each | 0 |
| 27 | Courtyard 1 (Outer Cloister) | 30/30 | 30/30 | 30/30 | 290 of 290 | 20/20 (10 + 10) each | 0 |
| 28 | Barracks | 30/30 | 30/30 | 30/30 | 491 of 491 | 20/20 (10 + 10) each | 0 |
| 29 | Jail 1 | 30/30 | 30/30 | 30/30 | 257 of 257 | 20/20 (10 + 10) each | 0 |
| 30 | Jail 2 | 30/30 | 30/30 | 30/30 | 332 of 332 | 20/20 (10 + 10) each | 0 |
| 31 | Jail 3 | 30/30 | 30/30 | 30/30 | 349 of 349 | 20/20 (10 + 10) each | 0 |
| 32 | Courtyard 2 (Inner Cloister) | 30/30 | 30/30 | 30/30 | 43 of 43 | 20/20 (10 + 10) each | 0 |
| 33 | Cathedral | 30/30 | 30/30 | 30/30 | 194 of 194 | 20/20 (10 + 10) each | 0 |
| 34 | Catacombs 1 | 30/30 | 30/30 | 30/30 | 262 of 262 | 20/20 (10 + 10) each | 0 |
| 35 | Catacombs 2 | 30/30 | 30/30 | 30/30 | 296 of 296 | 20/20 (10 + 10) each | 0 |
| 36 | Catacombs 3 | 30/30 | 30/30 | 30/30 | 363 of 363 | 20/20 (10 + 10) each | 0 |
| 37 | Catacombs 4 | 30/30 | 30/30 | 30/30 | 217 of 217 | 20/20 (10 + 10) each | 0 |
| 38 | Tristram | 30/30 | 30/30 | 30/30 | 337 of 337 | 20/20 (10 + 10) each | 0 |
| 39 | Moo Moo Farm | 30/30 | 30/30 | 30/30 | 1,124 of 1,124 | 20/20 (10 + 10) each | 0 |

Sum: 170 result lines on seeds 1..30 (34 levels x 5 lines: tiles, units, seeds, monsters rooms, monsters seeds), 340 on the ranges, **0 differ**; every line is "N/N". Level 20 (Tower 2) has no populated rooms on either side (why it has none was not traced; I that it is a preset-only level); its three other kinds equal. A compare with an empty dump would also say "match", so one dump per kind was looked at: `d8_dump.sh 1 6` prints 7,992 lines (Black Marsh tile grids), level 9 3,636 lines (54 rooms of 8 x 8 preset tiles), level 17 2,751 lines: the dumps are real, not empty (V run).

Before: MA04 said "0 verified / 102 ported" for the outdoor levels (the matrix, 2026-11-06) and seedspawn-trace.md covered 5 levels. Now 34 more are verified; the monster counts are the number of rooms with spawns (not units).

## Q2. What this does not prove (limits)

- Layout, presets, room seeds and the spawn list only. Per-unit state after creation (life roll on the world stream, the unit seed one step behind) stays seedspawn-trace.md D1/D2 (open, not touched here).
- Nightmare and Hell (`DIFFICULTY=`) were not run here (U): the generator reads the same Levels.txt columns, but the population differs by difficulty.
- Quest-linked unique placement (superuniques via `FUN_005a4f50`) is the boss-mods card, not layout.
- The `game` kind (whole game in sequence, container drops) was not re-run for these levels: seedspawn-trace.md did it for levels 1, 2, 3, 4, 8 (30/30).

## Q3. Same-save check (AC #3): not done

Needs the scenario-8 character (Stony Field first load) read on game.exe and on d2d, i.e. a saved character and a game: U. Recipe below.

## Q4. Lut Gholein (AC #4)

The act 2 town (level 40) was an open mismatch in docs/research/lutgholein-trace.md (room seeds of d2d shifted by two rooms; the tile edge ownership was a compare-order artifact). On main 83fc7ba1: `tiles`, `units`, `seeds` 30/30 and `layout` (the outdoor rectangles and Lut Gholein's file, levels 2 and 40) 30/30 (V run, `d8_more.sh`, `d8_ss_all.sh`). So the seed shift is not visible any more in the seed kind. The live cell-for-cell comparison of that trace (20 rooms around the player, in the client's room order) was not repeated (U). Recorded as: closed in the offline kinds, live check open.

## Deviations (d2d against game.exe)

None found in the compared kinds. The open items are those of seedspawn-trace.md (D1 life roll stream, D2 unit seed one step behind) and difficulty above Normal (U).

| id | what | tag | differential |
|---|---|---|---|
| D1 | unit seed one step behind after the life roll (seedspawn D2) | V run (earlier, 14,605 of 14,605) | `ss_unitseed_run.sh` exists in this repo's tools/emu for levels 2, 3, 4, 8; could be extended to the 34 levels above with their `ss_seed_test` binary against the current main |
| D2 | Nightmare / Hell population | U | `DIFFICULTY=1 d8_ss.sh ...` (the env var of `diff_drlg.py`); not run here for time |

## Bot findings

None new: the bot reads levels through the backend, not through the generator.

## PR candidate

No PR candidate: nothing deviates.

## Drill recipe (lab, not run)

AC #3: lab binary (docs/d2d-fork.md), `--seed N` with the pinned seed of scenario 8, enter Stony Field from Cold Plains, `debug` the monster and object list (docs/forkdbg.md); on game.exe the same save with the same map id via the d2s field 0xAB; compare class, position and mode per room against `ss_run.sh ... monsters`.

## Matrix-cell update (proposed)

MA04: "UPDATE 2026-11-09 (decomp8, fid-layout-act1.md): all 34 Act 1 levels other than 1 to 4 and 8 (ids 5-7, 9-39) equal game.exe on upstream 83fc7ba1: tiles, preset units, room seeds and populated rooms, seeds 1..30 and the two 32-bit ranges, 0 differ (V run); Nightmare/Hell and the same-save first load not run (U); Lut Gholein now equal in tiles, units, seeds, layout (30/30)."
