# LEVELSAREAS: Levels.txt area rows, d2d vs game.exe vs our bot (first pass)

Tags: **V run** game.exe's real code run under the emulator (`tools/emu/lv_area.py`, `FUN_0061dca0` native) and compared with the table d2d loads; **V read** read in a disassembly or in source; **D** the patched 1.14d Levels table (`tools/emu/mpq-table.sh levels`, census `tools/emu/lv_tbl.py`); **S** an existing proof (named); **I** inferred; **U** unverified. Only the work of this card is claimed V run. The d2d source was read through the lab checkout (read only); no d2d build was made in this pass.

## 1. Answers

| Q | Question | Answer | Where | Tag |
|---|---|---|---|---|
| Q1 | What is the area monster level game.exe uses? | `FUN_0061dca0(levelId, difficulty, expansion)` returns Levels.txt MonLvl{1,2,3}Ex (expansion) or MonLvl{1,2,3} (classic) for difficulty 0..2, and 1 for difficulty 3 and for any id past the table. | FUN_0061dca0; consumers `FUN_00573cb0` (monster level), `FUN_00558200` (item level of non-monster units, treasure-trace.md) | V run |
| Q2 | Does d2d hold the same numbers? | Yes for every real row. d2d fills `GameData::area_level[id] = {MonLvl1Ex, MonLvl2Ex, MonLvl3Ex, MonLvl1}` (gamedata_load.cpp:682) and indexes it by `clamp(difficulty, 0, 2)` (world.cpp:425, 1329, 1597); ids past the table give 1 (world.cpp:425). Game result vs those columns: **1,088 of 1,088** cases equal (ids 1..136 x difficulty 0..3 x expansion 0/1). | lv_area.py | V run (game) + V read (d2d) |
| Q3 | Is the table itself odd? | No. 137 rows (ids 0..136), 125 with a density in some difficulty; in none of them is the area level not monotone (Normal <= NM <= Hell), a density above d2d's 10000 cap, MonUMin above MonUMax, or a populated row without monster slots; no populated Normal row has MonLvl1Ex 0. The classic and Ex Normal columns are equal on all 125 rows. | lv_tbl.py | D |
| Q4 | What do the Nightmare / Hell area levels look like? | Wilderness 1: 1 / 36 / 67; Courtyard 1: 9 / 40 / 70; Catacombs 4: 12 / 43 / 73; Moo Moo Farm: 28 / 64 / 81; Arcane: 14 / 48 / 79; Throne Room: 43 / 66 / 85. | levels.txt | D |
| Q5 | Which drops use the area level? | Non-monster item levels (chests, objects): the area level, `FUN_00558200`; d2d: `chest_tc(act, diff, here, alvl(low), alvl(high))` (world.cpp:433, 580..638, 1601) and the `hdm` loot at world.cpp:1330 use the same column per difficulty. Monster drops use the monster's own level (stat 12). | treasure-trace.md | S + V read |
| Q6 | Monster level in NM / Hell | Already traced and run: pr-nmlevel-verifier.md (V run, 2 seeds); d2d fixed in the maintainer's #170. Not repeated. | monster-stats-trace.md D1 | S |
| Q7 | Density, unique counts, spawn counts | d2d: per room a `game.next() % 100000 > density` roll (density capped at 10000, objgroups.cpp:716..722), uniques from `umin` / `umax` of the difficulty (objgroups.cpp:726, 1113). The game-side population is run by `ms_spawn.py` (monster-stats-trace.md R3..R5: real unit stats of populated levels), but no count-per-level differential of d2d against game.exe exists. | | V read (d2d) / **U** (equality of counts) |

## 2. Run (what was done)

- `tools/emu/run-on-worker.sh mpq-table.sh levels` gives the patched Levels table (141 lines, header plus 140 rows incl. rows after id 136; `denominators.tsv` counts 137).
- `tools/emu/run-on-worker.sh lv_area.py`: `FUN_0061dca0` for ids 0..140 x difficulty 0..3 x expansion 1/0 = 1,128 cases. 1,090 equal the Levels column (what d2d holds); 32 cases are ids 137..140 (no row; game returns 1 = d2d's fallback); **6 differ, all id 0** (the Null row): game returns 1; the table has 0 / 25 / 50 (and Ex 0 / 0 / 0).
- `tools/emu/lv_tbl.py levels.txt`: the census of Q3.

## 3. Deviation list

| # | Rule, game.exe vs d2d | Tag | Differential |
|---|---|---|---|
| D1 | id 0 (Null row): game 1; d2d `area_level[0]` holds the raw column (0 / 25 / 50 classic, 0 Ex). No level uses id 0, so no effect. | V run + V read; harmless | none needed |
| D2 | d2d always reads the Ex (expansion) columns; a classic game would read MonLvl1..3 in game.exe (the 26 vs 36 style difference at id 2). d2d serves LoD only. | V run (game, both columns differ), I (no classic path) | game side exists (lv_area.py expansion 0); d2d has no classic mode |
| D3 | Counts of monsters per level (density roll per room, umin / umax) are not compared with game.exe. | U | buildable: run `ms_spawn.py` for a level and seeds, count units and uniques per difficulty; d2d side needs a probe calling the room population with the same room list and game seed (ms_build.sh style harness) |
| D4 | MonWndr / MonSpcWalk / NumMon: d2d keeps `wander`, `num_mon`; usage not compared. | U | read FUN_ users of Levels +0x.. then compare |

## 4. Bot findings (not changed)

- B1: `internal/bot/geargate.go:155` `areaMLvl` reads the Levels.txt column, which is the area level and equals game.exe's `FUN_0061dca0` (V run here); for Normal it is an upper bound of unit levels, as monster-stats-trace.md B2 says. No new finding in the Levels rows.
- B2: `internal/bot/recoveryxp.go:16-17` fallback levels (see monster-stats-trace.md B2); repeated by reference only.

## 5. PR candidate

No PR candidate: the area-level rows agree (0 mismatches before and after; nothing to fix). D3 is unproven.

## 6. Drill recipe (lab, for later)

- Q7/D3: in a lab Normal game enter Blood Moor with fork verbs from docs/forkdbg.md and count `monsters` in the level once it is populated; compare with the game-side count from `ms_spawn.py` on the same seed (docs/d2d-fork.md for the seed knob). Repeat Nightmare for the NM umin/umax (4..5) and Hell (7..9) in Wilderness 1.

## 7. Matrix-cell updates

Coverage ledger: `levels` rows 1..136, area level aspect only, TESTED-0-DIFF with 1,088 cases. Density, unique counts, wander and spawn counts stay untested (D3, D4).

## Shared-file lines for the merger

See `openspec/changes/levelsareas/merge-lines.txt`.
