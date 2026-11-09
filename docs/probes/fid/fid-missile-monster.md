# fid-missile-monster (TASK-84, matrix row MA10): monster missiles and ranged attackers, d2d against game.exe

> status: finished 2026-11-09 by agent decomp7 (Sonnet). Result: **AC #1 partly proven.** The launch record (physical and element of every Act 1 and 2 MissA / MissS1 / MissC row) is 0 differ on current upstream main 7557a54d against the REAL d2d build (60,000 cases, 36 rows, 3 difficulties). The to-hit gate is merged upstream; the block gate (can-block = physical != 0) is still open on d2d (V read of combat.hpp, game.exe side from the earlier run). The "371 untested rows" count is NOT closed: only the 36 Act 1 / 2 launch rows are run (U for the rest).

Tags: V run (game.exe's real code under the emulator against the real d2d test binary), V read, D, S (existing proof), I, U. Worker host, scratch tree = upstream main 7557a54d. Scripts `tools/emu/d7_mm.sh` (uses upstream `mm_diff.py` / `mm_launch.py`, merged).

## Q1. Launch record, game.exe against the real build: 0 differences V run

`mm_diff.py game ~/mmtab 60000 20261109` (tables from the 1.14d patch stack) writes the cases and game.exe's record (FUN_0059fa30 -> 0059f900 -> 0064b860 / 0064ac60, stats 0x15 / 0x16 and 0x30..0x3b read with FUN_00625480); `test_monster_missile --monster-missile cases.txt` is d2d's real `monster_shot_phys` / `monster_shot_stats`. Result: **cases 60,000, rows 36, d2d lines missing 0, physical differ 0, element compared 27,947 differ 0, at the game's own missile levels (1 / 4 / 8) 1,491 cases 0 differ.** This confirms the merged missilediff fix (D1 level brackets, D2 HitShift, D3 row element and poison) on current main with a new seed (before: 44,456 of 120,000 differed on 23e2baec, docs/research/missilediff.md).

## Q2. To-hit gate (trace deviation b): fixed upstream V read

game.exe rolls to-hit for a MissA missile only when the row's ToHit (+0x18b) is set (94 of 142 rows have none). Upstream 125443ba ("Monster missiles roll to hit by their ToHit"): `stats.no_hit_roll = !info.to_hit` (components/game/ai.hpp:800, 834; fight/deaths.cpp:190). Read at 7557a54d. No numbers run in this card for it (the earlier 100,000 case roll run was against a port, S).

## Q3. Block gate (trace deviation a): STILL OPEN on d2d, V read

game.exe `005ad730` -> `0057dfb0(.., missile 1, can-block = record +8 != 0)`: a missile with no physical part cannot be blocked (25 of 49 Act 1 / 2 rows; Fallen Shaman fire, Skeleton Mage bolts, Bighead lightning, the spits and pots). d2d `rules::monster_blow` (components/rules/combat.hpp:923-925) rolls `defender.block` for every missile whatever its physical part: no gate (read at 7557a54d). Earlier numbers (S, monster-attack-trace.md): 10,503 of 100,000 block cases differ, all vs a reading of game.exe's gate, d2d side a port. **Tag: game.exe side V read + S, d2d side V read; no differential against the real build was run here (U).** Effect: only a shield build blocking Shaman fire or a mage bolt; the Sorceress and a shieldless Paladin are unaffected; low.

## Q4. Soaky death, remaining rows

- The 371 untested of 684 missile rows: not run (U). The run covers the rows that d2d's monster MissA path uses; skill-sourced rows go through `andariel_missile` (ai.cpp:403) and are not on this path.
- Soaky's death has no d2d twin: not examined (U).

## Deviations (d2d against game.exe)

| id | what | tag | differential |
|---|---|---|---|
| D1 | block gate: d2d may block a missile with no physical part, game.exe never | V read (both sides) + S | could be built: `test_monster_missile` style test over `monster_blow` with a defender block > 0 and a record of max 0; game side FUN_0057dfb0 with the can-block argument 0/1 and a fixed seed (oracle `mon_missile.py block` of the earlier trace); not built here |

## Bot findings

B1 (unchanged): no per-missile threat model; the dodge reacts to any hostile missile within 4 subtiles (monster-missile-trace.md B1-B4). Nothing new.

## PR candidate

No PR candidate: no d2d deviation was proven with the existing differential in this run (D1 is a V read; the record is equal).

## Drill recipe (lab, not run)

D1: lab binary, a Paladin or Sorceress with a shield and Block above 0, a Fallen Shaman (fire bolt, no physical): `debug spawn` one at range, stand and count blocked bolts over 200 casts (`debug combat log` of docs/forkdbg.md); game.exe expects 0 blocks, d2d some.

## Matrix-cell update (proposed)

MA10: add "UPDATE 2026-11-09 (decomp7, fid-missile-monster.md): launch record 60,000 cases x 36 rows 0 differ against the real build on 7557a54d (V run); to-hit gate merged (125443ba); block gate still open (V read, no run on the real build); 371-row count not closed (U)."
