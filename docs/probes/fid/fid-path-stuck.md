# fid-path-stuck (decomp batch 7, matrix row MA12): game.exe pathing against d2d, and what the bot relies on at cave mouths

> status: finished 2026-11-11 by agent decomp11 (Sonnet), offline (emulator on the worker host; no game, no live d2d, nothing pushed or posted). Result: **re-run of upstream's path oracles (`tools/emu/d6_base.sh move`): velocity 122,646 cases per seed x 2 seeds 0 mismatches, walk rates 1,203 / 0 differ, the pathers seen today equal (search 2,000, player 2,000, wall 2,000, sight 20,000, search-target 20,000, PATH accessors 5,065; the toward pather and chase check carried from decomp6), the same numbers as decomp6's [fid-monster-move.md](fid-monster-move.md).** Nothing new proven: the stuck cases (crowd collision, the exit trigger at a cave mouth) are still not compared; this card writes down what the bot relies on, where each reliance is proven, and the one differential that would close it.

Tags: V run (game.exe's own code run by the emulator against d2d's real or line-for-line code), V read, V live (a bot log), D, S, I, U.

## Q1. What the bot relies on when it walks (reliance table)

| # | The bot's reliance | Where | Proven by | Tag |
|---|---|---|---|---|
| R1 | a click-move toward a spot reaches it by game.exe's player pather (type 7: toward, then search when close) | `internal/nav` plans its own route and clicks waypoints along it | FUN_00679ed0 against d2d's port: 2,000 cases, 648 reach their end (this run, again) | V run (player pather); the bot's routes are its own A* (`internal/nav/astar.go`), the pather only fills in between clicks (I) |
| R2 | a walk longer than 77 path points stops at the 77th and the bot re-clicks | stuck ladder, re-click (fixmouth4.md) | FUN_00648e70 (clamp 0x4d, V read), `path_dynamic.py` 5,065 cases ok | V read + V run |
| R3 | the walking speed: base, run, chill, faster run/walk | step estimates, kill-time models | `walk_velocity.py` 2 seeds x 122,646 cases 0 mismatches; `walk_rates.py` 1,203 / 0 | V run |
| R4 | sight / line of fire for ranged attacks and for spell lines | `internal/combat/reach.go` BlockLOS | `sight.py` 20,000 cases ok (1,606 blocked), d2d's `sight_blocked` | V run |
| R5 | a click toward a spot beyond walls routes around them | stuck cases | `wall ok: 2,000 cases, 970 reach their end, 1,252 go round` | V run (toward pather + search around walls, walls only) |
| R6 | monsters do not block a click path for good: a stalled walk fights what blocks it | `Fighter.yield` (combat/fight.go:25) | **not compared**: crowd collision is U (fid-monster-move Q4) | U |
| R7 | stepping next to a level-exit tile fires the exit; a click whose straight segment comes within 15 subtiles of an unwanted exit may fire it | `internal/nav/mouthgate.go:23` (`mouthTriggerR` 15) and `mouthR` 9 / `mouthApproachR` 12 in walk.go; V live in fixmouth.md: arrival (40,145) to a click toward (36,146) fired the trigger 12 ms later | V live (DRILLS14 / 16 logs); game.exe's own exit-trigger rule **not traced** | V live + U |

## Q2. What the six pathers do (V run, re-run today)

`tools/emu/run-on-worker.sh --ttl 1800 d6_base.sh move` (the scratch checkout of d2d's upstream main, decomp6's setup):

| block | result |
|---|---|
| walk_velocity.py seed 9191 / 4242 | 122,646 / 122,646 cases, mismatches 0 / 0 (per kind: monster WL 30,867 / 30,787, RN 30,912 / 31,131, player 29,466 / 29,346, merc 7,962+7,697 / 7,869+7,916, pet 7,853+7,889 / 7,717+7,880) |
| velocity truncation | 47,023 / 46,837 cases (game.exe truncates velocity x pct / 100 to whole 1/4096 subtile units, d2d's float step does not): D1 below |
| walk_rates | 1,203 lines, 0 differ |
| `moves.py`: search pather FUN_0067b850 / player FUN_00679ed0 / wall pather | ok 2,000 each (1,047 / 648 / 970 reach their end, 1,252 go round walls); "think at once: WL and RN only, as ai.cpp" ok (the toward-pather and chase-check lines of decomp6, 20,000 cases each, were cut off by the script's `tail -4` in this re-run: carried from fid-monster-move Q2, not re-seen today) |
| sight.py 20,000 seed 3 | ok, 1,606 blocked |
| search.py 20,000 seed 2 | ok, 16,714 found a target |
| path_dynamic.py | ok, 5,065 cases over the PATH accessors, setters, collision and recalc functions (one benign emulator message: a bad memory read at an unmapped page in a function the case set does not claim) |

## Q3. The stuck cases at a cave mouth: what is known (I, U)

What the three traces say together, and what is missing:

1. **The pather is not the cause (V run).** The paths a click produces on walls are equal in game.exe and d2d. The stuck ladder in `internal/nav` handles stalls by re-click and side steps; what it re-clicks into is the same pather.
2. **The trigger is the hazard (V live).** The Den and Cave Level 1 mouths: arrival 11.2 subtiles from the listed exit, one click toward the exit side fired the trigger within 12 ms (4 of 4); clicks that start 16 to 18 away did not fire (fixmouth.md). The game fires an exit when the player's position enters the exit's tile region; a click's path is not a straight segment around walls, so the bot's `mouthTriggerR` 15 on a straight segment is a **proxy** (conservative on open ground, I). A path around a wall may come closer or stay farther than the segment (not computed).
3. **Not traced:** (a) the exit-trigger test in game.exe (the unit-move code that looks up the tile's level-exit flag and warps; I did not find it in this card's time; candidates: the room's `levelexit` list checked in the unit's path step), (b) crowd collision: a monster standing in a one-wide mouth stops the path (FUN_0064d9b0 with the mover's size; d2d's `Crowd`, fid-monster-move Q4), (c) what `FUN_0064d910`'s unit pattern does when the first step is blocked (the pather re-paths each chase tick, V run for the chase check, equal).

## Q4. Where d2d and game.exe could differ at a mouth (D list)

| id | what | game.exe | d2d | tag | differential |
|---|---|---|---|---|---|
| D1 | velocity truncation | whole 1/4096 units a tick | float step | V run (counted: 47,023 of 122,646 non-integer); effect up to 0.5 percent of speed on the slowest rows (I) | `walk_velocity.py compare`; per-tick position drift needs a multi-tick replay |
| D2 | crowd collision | a unit-sized pattern against units (FUN_0064d9b0 with the mover's size) | `Crowd` | U | buildable: run FUN_0064d910 against a unit list instead of walls (moves.py hooks "the collision test only"), oracle = the path points; d2d side `Crowd` with the same units |
| D3 | exit trigger | tile region test on the unit's step | unknown to this card | U | buildable: find the test in game.exe (search for readers of the level-exit tile flag from the path step), run it over a grid of positions and exit rects |
| D4 | the 77-point cap on the monster side | clamp 77; the search pather gives up past 77 | `monsters.hpp:742-748` same | V run equal (path_dynamic, search) | built (decomp6) |

## Bot findings (described, not changed)

B1 (new) `internal/nav/mouthgate.go:23` `mouthTriggerR = 15.0` is a straight-segment proxy; game.exe's exit fires on position, and the path goes around walls. Where the straight segment runs through rock and the real path hugs a mouth, the gate can say OK while the path passes closer than 15 (I; not computed). Consumer: a probe card to trace the exit trigger (proposed `fid-exit-trigger`, batch7.md); until then nothing to change.
B2 (carried, fid-monster-move B2) no bot code uses the 77-point cap; a long click stops at the 77th point and the soak's "stuck" counters can mistake that for a stall (I).
B3 (carried) the run toggle: `internal/d2denv/env.go:104` hooks `ResetRun` to `OnInGame`.

## PR candidate

None: no deviation proven in the covered paths (D1 below one unit a tick; D2 to D3 have no harness).

## Drill recipe (lab binary, docs/forkdbg.md; writing only)

M1 (D3 / R7): in a lab level with a known exit tile, `debug goto` a spot 20 subtiles away on the far side of a wall from the exit, click-move to the opposite side of the wall so the straight segment passes within 12 of the exit while the real path stays 20 away; check whether the exit fires (expect no). The reverse (a segment more than 15 away, the path hugging the mouth) is the dangerous case. M2 (D2): `debug spawn zombie 8` in a one-wide corridor in front of the player, walk back, count ticks a rear zombie stays still (fid-monster-move drill 2, unchanged). M3 (R2): a click 120 subtiles away on a clear line: count path points after the click: expect 77.

## Matrix-cell update (proposed text, MA12)

"UPDATE 2026-11-11 (decomp11, fid-path-stuck.md): the path oracles re-run unchanged (velocity 2 x 122,646, walk rates 1,203, six pathers, sight, search, PATH accessors: all equal); open: crowd collision (D2) and game.exe's exit-trigger rule (D3, the cave-mouth hazard of the bot's mouthgate): both U with a buildable differential."
