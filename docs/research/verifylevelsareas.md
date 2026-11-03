# VERIFYLEVELSAREAS: independent check of levelsareas-trace.md

Tags as in AGENTS.md. Verifier: Sonnet, own run of `tools/emu/run-on-worker.sh lv_area.py` (the author's script re-run; it only calls `FUN_0061dca0`, so a second harness would call the same function) and own read of the d2d source in the lab checkout (read only).

| Claim | Verdict | Why |
|---|---|---|
| Q1 game rule of `FUN_0061dca0` | VERIFIED (counts) | Re-run: 1,128 cases, 6 mismatches, all id 0 (game 1, table 0 / 0 / 0 Ex, 0 / 25 / 50 classic). Ids 137..140 return 1. |
| Q2 "1,088 of 1,088 equal d2d" | HOLDS WITH LIMITS | The oracle in `lv_area.py` is the Levels.txt column, not a d2d run. d2d loads the same columns (gamedata_load.cpp:682 V read, world.cpp:425/1329/1597 index by clamped difficulty, 1 past the table, V read). So this is game vs table plus a source read of a plain copy; no d2d binary was run on a level. Honest tag: V run (game vs table) + V read (d2d copy). It is not an arithmetic re-derivation of a d2d formula, but it is also not a d2d-vs-game differential. |
| Q3 table census | NOT RE-RUN (D tag accepted) | Pure data statements; not needed for any PR. |
| Q5 chest/object item level uses the area column | HOLDS WITH LIMITS | d2d side read (world.cpp:433, 1601); the game consumer `FUN_00558200` was not run in this card (S: treasure-trace.md). |
| Q7 / D3 / D4 counts, wander | NOT VERIFIED | Correctly marked U. |
| D1 id 0 harmless | VERIFIED (counts) | 6 cases, reproduced. |
| D2 classic vs Ex | HOLDS WITH LIMITS | Game both columns differ (run); d2d has no classic path (read). |
| "No PR candidate" | AGREED | |

## Gaps the first pass missed (U, not claimed fixed)
- G1: d2d also reads `area_level[id][3]` (classic MonLvl1) for NPC trap level (gamedata.cpp:560, 568) and `[0]+1` (gamedata.cpp:351). The game's trap and top-level rules use other functions; they are not in the 1,088 and need their own differential (run the game's trap-level function, same ids).
- G2: world.cpp:1597 indexes `[diff]` where `diff` is clamped to 0..2 (line 1595), so the game's difficulty 3 value of 1 is unreachable in d2d; harmless, I.
- G3: Nothing here compares a d2d-run area level (e.g. drop-dump on a level) with the game. Low value: the table is copied unchanged.

## Ledger rows
Earned: `levels` rows 1..136, aspect "area monster level (MonLvl1..3Ex)", game function vs table equal, d2d copy read. Tag the row "game=table, d2d=copy (read)", not TESTED-vs-d2d-run. Density, uniques, wander, traps stay untested.

## Shared-file lines for the merger
See `openspec/changes/verifylevelsareas/merge-lines.txt`.
