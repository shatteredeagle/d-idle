# fid-teleport (TASK-90, matrix row SK-TELE / TS): Teleport destination, refusal and mana, d2d against game.exe

> status: finished 2026-11-09 by agent decomp7 (Sonnet). Result: **destination rule (range by room, blocked target snap) is 0 differ on current upstream main 7557a54d against the real d2d build: 40,000 cases in two seeds.** The mana on a refused hop (D4 of docs/research/sk-teleport-trace.md) is NOT compared by a run; both sides are V read and d2d still pays (U for the run). AC #2 proven (table below), AC #1 and #3 open.

Tags: V run (game.exe's FUN_005ca360 / FUN_00554ea0 under the emulator against the real `test_tele_spot` binary), V read, D, I, U. Worker host. Script `tools/emu/d7_tele.sh` (upstream `sk_tele_spot.py` / `sk_mana.py`, merged).

## Q1. Destination rule, game.exe against the real build: 0 differences V run

`sk_tele_spot.py gen 20000 {1,2} --no-units` (5 x 5 rooms of 40 x 40 subtiles, caster in the centre room, Levels Teleport column 0 / 1, collision words) against `test_tele_spot --tele-spot`: **20,000 cases each, 20,000 same, 0 differ.** The own checks of `test_tele_spot` print ok. So the trace's D1 (range = caster's room or near list, 25,120 of 100,000 differed before) and D2 (blocked target lands on the nearest free spot, ring first then |dx|+|dy| out to 50, 1,323 differed) are fixed upstream (3fbb6803, 57d7c39b) and equal.

Limits (honest): units (0x1000) are excluded (`--no-units`: d2d keeps units in the Crowd, so trace D3 "monster at the target" is not covered here, U); Teleport 2's clear-line test is hooked to "clear" on the game side and covered by the upstream commit f8445a7d (sizes 2 at both ends, FUN_006229f0) without a differential (U); the near list is the 3 x 3 touching rooms (a hook); irregular levels are not covered.

## Q2. Mana on a refused hop (D4): V read both sides, not run

- game.exe: FUN_0056f7f0 stores the do's return in [ebp-0xc] (0x56f96a); at 0x56fa3d a 0 return jumps to 0x56f8b3 and returns 0 before the payment at 0x56fa70 (FUN_0056bfe0), for a skill without a missile (0x56f97a: no missile, ax < 0). Do 27 (FUN_005ca360) returns 0 on every refusal. So a refused Teleport costs nothing (V read, 0x56f7f0..0x56fa70).
- d2d: a skill with no srvstfunc is owed (`Fight::owe`, tick.cpp:127) and paid by `settle_cast()` at strike.cpp:1018 BEFORE the do runs; the Teleport case then `break`s on a refusal (strike.cpp:845-860) without refunding. So d2d takes the mana on a refused hop (V read of both). Not run on the real build: `test_cast_mana` covers Fire Bolt, Charged Bolt, Frost Nova and Frozen Armor only.
- `sk_mana.py` check (40,000 cases, seed 1) compared 71,638 of 130,840 rows: equal 71,316, different 322, all "game casts, d2d refuses" and in skills 41, 230 and 266, none in Teleport (54: startmana 0, minmana 1, manashift 8, mana 24, lvlmana -1, srvdo 27, usemanaondo false). V run.

## Proof table of D1/D2 (AC #2)

| id | rule | before | now (7557a54d) | tag |
|---|---|---|---|---|
| D1 | range: only the caster's room or its near list | 25,120 of 100,000 differ | 0 of 40,000 | V run |
| D2 | blocked target: nearest free spot | 1,323 of 100,000 | 0 of 40,000 | V run |
| D3 | unit at the target (mask 0x1000) | 4,805 of 100,000 | not run (--no-units) | U |
| D4 | mana on a refused hop | 24 points at level 1 | still paid (V read) | V read, differential not built |

## Deviations (d2d against game.exe)

| id | what | tag | differential |
|---|---|---|---|
| D4 | d2d pays the mana of a refused Teleport, game.exe does not | V read (both) | could be built: extend `test_cast_mana` (tests/game, Rig with a target and a Levels Teleport 0 or out-of-room point) to skill 54 and take the mana at the action frame; game side: call FUN_0056f7f0 with the do (table 0x7322b0 entry 27) hooked to return 0 / 1 and read stat 8 after; oracle: mana unchanged on 0. About 100 lines of C++ plus an emu script. Not built here. |
| D3 | units at the target | V run earlier (4,805 of 100,000 against a line copy) | `sk_tele_spot.py gen` without `--no-units` against a d2d side that fills the Crowd |
| D7 | arrival bookkeeping (pets, ring, warp spot) | V read | no |

## Bot findings

B1/B2 (docs/research/sk-teleport-trace.md): the bot's 18-subtile hop is the backend's screen limit, not the skill's, and a refused hop costs 24 mana on d2d that game.exe would not take. B2 stands (D4). No code change.

## PR candidate

No PR candidate: D4 is only V read; the existing differential (`test_cast_mana`) does not run Teleport, so mismatches-before cannot be shown.

## Drill recipe (lab, not run)

D4: lab binary, a Sorceress with Teleport (clvl 18, `docs/forkcheats.md` setup on a throwaway), full mana; aim at a wall cell more than 50 subtiles from any free spot (or in a town room) and cast; `debug stats` before and after: game.exe expects mana unchanged, d2d minus 24 at level 1. AC #3 of the task (live read) is this drill.

## Matrix-cell update (proposed)

SK-TELE row: "UPDATE 2026-11-09 (decomp7, fid-teleport.md): destination rule 0 differ on 7557a54d (2 x 20,000 cases, real build, units excluded; V run); refused-hop mana still paid by d2d (V read, no run); units at target not covered (U)."
