# fid-monster-move (TASK-82, matrix row MA12): monster velocity, pathing, the 77-point path, d2d against game.exe

> status: finished 2026-11-08 by agent decomp6 (Sonnet). Result: **the velocity deviation of the earlier trace (F08, monsters moving at Run in RN, chill adding instead of multiplying, floor 10 instead of 25, 75 percent start) is MERGED upstream and the oracle now shows 0 mismatches over 122,646 cases (two seeds); the monster pathers and the chase check compare equal on 2,000 random cases each; the 77-point cap is read and run.** No new deviation, no PR candidate. Open (U): the 63 untouched PATH functions, monsters blocking each other (crowd collision), the drawn walk frames.

Tags: V run (game.exe's own code run by the emulator and compared with d2d's real or line-for-line code), V read, D, I, U. Worker host, scratch tree = upstream main a5b970d5 plus the TASK-83 candidate commits (they touch missile hits only). Scripts `tools/emu/d6_base.sh move`.

## Q1. Walk and run velocity of a monster, player, merc and pet (AC #1) V run

`walk_velocity.py gen 100000 {9191, 4242}` (FUN_00623f50: velocity base, stat 0x43 deltas, chill coldeffect, the 25 floor, velocitypercent) against `test_walk_velocity` (d2d's real rules). Cases: every MonStats row in WL and RN (where MonStats2 has the mode) x deltas x chill, every class x walk / run x faster run/walk x velocitypercent, merc and pet rows.

| Seed | Cases | Mismatches | per kind (cases) |
|---|---|---|---|
| 9191 | 122,646 | **0** | monster RN 30,912, monster WL 30,867, player 29,466, merc WL/RN 7,962 / 7,697, pet WL/RN 7,853 / 7,889 |
| 4242 | 122,646 | **0** (every kind 0) | monster RN 31,131, WL 30,787, player 29,346, merc 7,869 / 7,916, pet 7,717 / 7,880 |

Notes from the run: monsters start at stat 0x43 = 75 (d2d: 8bf07f12, merged); `monster-velocity-truncated` 47,023 cases (seed 9191): game.exe truncates velocity x pct / 100 to whole 1/4096-subtile units, d2d's float step does not. That residual is below one unit (1/4096 subtile) a tick; for the slowest rows (Velocity 1 at 75 percent = 192 units a tick) it is up to about 0.5 percent of the speed (I; arithmetic, not run per tick). Earlier trace deviations M1 (RN at Run), M2 (chill adds coldeffect), M3 (floor) = the old 425,665 mismatches: **gone**; the commits are in the upstream log (monster base velocity, `walk_velocity.py`, `walk_rates.py`).

Walk and run animation rate: `walk_rates.py` against `test_combat --walk-rates`: 1,203 lines, **0 differ** (V run).

## Q2. The movement step and the pathers (AC #2) V run

`moves.py` (upstream's) runs game.exe's functions natively and compares with the port of `rules/monsters.hpp`:

| Function | What | Result |
|---|---|---|
| FUN_00679c80 | the toward pather, path types 2 / 5 / 6 / 0xd (a monster's walk and run) on random walls, starts, ends, near and steps | `paths ok: 20000 cases, 10259 short of their end` |
| FUN_006503f0 -> FUN_00650350 | the chase check: re-path, stop distance, budget, target kind | `chase ok: 20000 cases, 7657 stop, 6175 go on, 6168 re-path` |
| FUN_005a8030 | which modes think at once when they end | "think at once: WL and RN only, as ai.cpp": ok |
| FUN_0067b850 | the search pather (path type 1) with and without a unit to walk to | `search ok: 2000 cases, 1047 reach their end` |
| FUN_00679ed0 | a player's path (type 7: toward, then search when close) | `player ok: 2000 cases, 648 reach their end` |
| wall pather (toward + search around walls) | | `wall ok: 2000 cases, 970 reach their end, 1252 go round` |

plus `sight.py 20000 3` (**ok, 20,000 cases, 1,606 blocked**), `search.py 20000 2` (**ok, 20,000 cases**, 16,714 found a target), `path_dynamic.py` (**ok: 5,065 cases** over the 70 dynamic-path accessors, setters and wrappers of PATH/Path.cpp). The step per tick (velocity x unit vector, no acceleration) is the documented rule in walk-speed-trace.md Q2 (V, earlier). "Port and compare the 6 ported ones" is therefore met: the six functions of the table (rows 1 to 5 plus the wall pather) are compared, every block `ok` (V run; 20,000 + 20,000 + 2,000 + 2,000 + 2,000 cases).

## Q3. The 77-point path (AC #3) V read + V run

- FUN_00648e70 (read with `disasm.py 648e70`): `if (path != null) { n = min(arg, 0x4d); path[0x91] = path[0x90] = n }`, an unsigned compare against 0x4d (77): a path's distance and maximum distance never exceed 77. V read.
- `path_dynamic.py` lines for the path point count (offset 0x28): the setter clamps to 77 on the high side only (signed compare); cases 0, 0x4D, 0x4E, 0xFF and random go through the setter at 0x648e70 (V run, 5,065 cases ok).
- The search pather (FUN_0067b850) rebuilds the route from its parent chain and gives up past 77 points (`moves.py:162-169`, game.exe's behaviour); d2d's port (`components/rules/monsters.hpp:742-748`, "none past 77 or with no step") returns no path in the same cases (V run: search ok, 2,000 cases). So a monster's route is cut at 77 points in both; a longer walk is a series of re-paths (the chase check's re-path rule, block 2). The riskmap's "path stops at 77 subtiles" line (`town.cpp:3224`) now sits in `apps/d2d/town/ambient.cpp:185-192` (an NPC's walk, cut along the line: "same line for a flyer", I).

## Q4. Stuck cases (not run) U

Monsters blocking each other and the player (game.exe tests a unit-sized collision pattern, FUN_0064d9b0 with the mover's size; d2d uses its `Crowd`), a monster in a one-wide corridor behind another, and a path whose target is unreachable are NOT compared; the six pathers above are compared on walls only. Differential buildable: game.exe's FUN_0064d910 hooked to answer from a unit list instead of walls (moves.py hooks "the collision test only") against `Crowd`; oracle = the path points. Until then "stuck rate" in the soak is a bot-side number.

## Deviations (d2d against game.exe)

| id | what | tag | differential |
|---|---|---|---|
| D1 | float step against game.exe's truncated velocity (47,023 of 122,646 cases non-integer) | V run (counted), I (size) | yes: `walk_velocity.py compare` already flags it; per-tick position drift would need a multi-tick replay |
| D2 | crowd collision and the unit-sized pattern in a path | U | buildable (above) |

## Bot findings (described, not changed)

B1 The walk-speed-trace B1 (run belief after a new game) is fixed in the bot: `internal/d2denv/env.go:104` hooks `ResetRun` to `OnInGame` (`internal/d2dinput/run.go:70`). The run toggle belief itself stays UNVERIFIED live there.
B2 No bot code uses the 77-point cap: `internal/nav` plans its own routes; a click-move longer than 77 path points stops at the 77th point in game.exe and in d2d (the bot re-clicks), which the soak's "stuck" counters can mistake for a stall (I).

## PR candidate

None: 0 mismatches after the merged fix; no proven deviation left in the covered paths.

## Drill recipe (lab binary, docs/forkdbg.md)

1. Q3: `debug goto` an open outdoor level, click-move to a spot 120 subtiles away on a clear line: count the path points (`dump player` path fields) after the click: expect 77 on both, the unit stops at the 77th point and the bot re-clicks.
2. Q4: `debug spawn zombie1 8` in a one-wide corridor in front of the player, then walk back: count the ticks a rear zombie stays still; compare with game.exe on an instance with the same layout.
