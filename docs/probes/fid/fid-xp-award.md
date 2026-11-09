# fid-xp-award (TASK-75, matrix row ST10): XP per kill and the level-difference penalty, d2d against game.exe

> status: finished 2026-11-09 by agent decomp7 (Sonnet). Result: **AC #1 proven: 0 differences in 119,790 grid lines and 100,000 random lines on current upstream main 7557a54d.** AC #2 (live read of XP before and after a kill on game.exe) is NOT done: no live game in this run (U). No deviation, no PR candidate.

Tags: V run (game.exe's own FUN_0057e2f0 under the emulator against d2d's real `rules::kill_exp` compiled from the scratch tree), V read, D, I, U. Worker host. Scripts: `tools/emu/d7_xp.sh`, `tools/emu/d7_kill_exp.cpp`, `tools/emu/xp_award.py` (card decompxp).

## Q1. Level-difference step: 0 differences V run

- Grid: exp in {1, 7, 18, 21, 33, 42, 100, 165, 1000, 12345, 0x7fffff} x clvl 1..99 x mlvl 1..110 = 119,790 lines `exp clvl mlvl award`. game.exe side `xp_award.py grid` (FUN_0057e2f0, eax = mlvl, edx = clvl, ecx = exp, nothing hooked). d2d side: `d7_kill_exp` includes `components/rules/combat.hpp` of upstream main and prints `d2d::rules::kill_exp` for every line. **`diff`: 0 differ** (the old port of `kill_exp` at e36922ea differed in 37,288 lines; fixed in 01337ba3, now confirmed on the real function, not a port).
- Random: upstream's own `kill_exp.py 100000 5` (exp up to 0x300000, clvl and mlvl 1..99, half of the cases within +-12 levels) against the same d2d function through `d7_kill_exp stdin`: **100,000 lines, 0 differ** (V run; upstream's test_combat asserts 400 of the same form).

## Q2. What is NOT covered (honest limits)

- The ExpRatio step (FUN_0057e390, `ratio` mode of `xp_award.py`), the 0x7fffff cap on a monster's exp, 0 at the class's MaxLvl, the item_addexperience percent (FUN_0057e480) are V read only; `rules::kill_award` was not run against them here (U for the whole chain; I that it is equal, as the 2026-10-30 trace found per-monster awards equal: 18 Fallen, 21 Quill Rat, 33 Zombie).
- The party share (FUN_0057e6c0) and the merc share (FUN_0057e860) are other rows (ME04).
- AC #2 (live XP per kill at clvl 4 to 6) needs a game: see the drill recipe.

## Deviations (d2d against game.exe)

None found in the level-difference step (0 of 219,790).

## Bot findings

None new. (docs/research/xp-award-trace.md B-rows stand: the Den gap was composition and counting.)

## PR candidate

No PR candidate: nothing deviates.

## Drill recipe (lab, for AC #2; not run)

On the d2d fork lab binary and on a game.exe instance: a fresh clvl 4 character, read stat XP (`debug stats` verb of docs/forkdbg.md; game.exe via the d2d memory reader), kill one Fallen, one Zombie, one Dark Hunter, read XP again. Expected deltas from the table: 18 / 33 / known values of docs/research/xp-award-trace.md exactly, at equal levels (mlvl within 5 of clvl: no penalty).

## Matrix-cell update (proposed)

ST10: add "UPDATE 2026-11-09 (decomp7, fid-xp-award.md): the level-difference step is 0 differ on upstream 7557a54d, 119,790 grid lines plus 100,000 random lines, real `rules::kill_exp` against FUN_0057e2f0 (V run); ExpRatio, cap and MaxLvl rows V read only; live read open (U)."
