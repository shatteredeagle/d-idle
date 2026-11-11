# Roam coverage spacing (TASK-301)

> status: CURRENT. Tags: V = measured (log read-only, or the simulator run in this worktree on the worker host), S = arithmetic from V numbers, U = unverified. Supersedes the first-pass guess `roamSpacing = 7`, `roamCover = 5` of 2026-10-30 ("about the monster wake radius", U, never measured on d2d). Code: `internal/nav/roam.go` (`roamParams`, `NotePosition`), `internal/nav/roamstudy_test.go`.

## Decision

`nav.roamSpacing` 16 and `nav.roamCover` 10 (subtiles, `internal/env/tuning_nav.go`), cover clamped to spacing. The legacy 7/5 pair stays selectable (the consts `roamSpacing`/`roamCover` are kept and tested as the legacy values). The grid is made when a level is first roamed; a changed knob applies from the next level or game.

## Evidence

Sandbox log (V, read-only `docker logs d2play-20261108`, only 29 min of it exists: 2026-11-11 14:23 to 14:52 UTC, a fresh container, clvl 11 Paladin; the older day is gone, so this is a thin sample):

| measure | value |
|---|---|
| Blood Moor grid at 7/5 | 1593 points (one level: 2019 for Cold Plains, Cave sizes not in this window) |
| roam legs in the window | 128 legs, 181 s walking, 1292 subtiles; leg path p10/p50/p90 = 2 / 7.7 / 20.7 subtiles, so the median leg is about one grid step |
| kills per area in the window | Blood Moor 120, Cold Plains 43, area 4 19 (of 190 kills) |
| kill time split (killtime lines, Blood Moor) | walk 281 s, cast 90 s, wait 583 s: walking between points is a third of the time that is not waiting for the next monster |
| time from fight start to first attack (`firstAttackSec`, 72 fights) | median 1.5 s, p90 3.7 s; at the measured walking speed of 12.4 subtiles/s (nav.trace legs) that is a median approach of about 18 and a p90 of about 45 subtiles (S, a proxy: it includes target choice) |
| "followed, turning to fight" | 40 times: the pursuit rule fired, monsters reach the hero on their own |

NOT measurable from the log (U): the distance at which each kill was first targeted (no log line carries it), and `roam: N/M points seen` (that status field is newer than the window). The first-attack proxy above is the only reach evidence, and it says hostiles are met at 18 to 45 subtiles, far above both 7 and 16; `bot.clearRadius` is 30.

## Simulator (V)

`ROAMSTUDY=1 go test -count=1 -v -run RoamStudy ./internal/nav`: nearest-first roam over a level field (hops, 8-neighbour, marks every cell along the path as the real roam marks each observation), then for every reachable walkable cell the distance to the nearest walked cell. Levels: the real Blood Moor of d2d seed 17 (`internal/d2dworld/testdata/seed17`, no exit filter, so 1844 points at 7/5 instead of 1593), a synthetic serpentine cave with dead-end stubs and rooms, a synthetic open field. The real Cold Plains, Den and Cave maps are not in the repo (only seed 17 levels 1 and 2 are recorded): U for them, rerun the command with their grids when recorded. Path is in hops (diagonal = 1, so a fair relative measure but about 20 percent below Euclidean walking).

Blood Moor (real, 77247 walkable cells):

| spacing/cover | points | legs | path hops | cells within 12 | max gap | pockets farther than 20 |
|---|---|---|---|---|---|---|
| 7/5 | 1844 | 1621 | 13070 | 100.0% | 9 | 0 |
| 10/7 | 983 | 842 | 9751 | 100.0% | 11 | 0 |
| 14/9 | 494 | 452 | 6933 | 100.0% | 14 | 0 |
| 16/10 | 394 | 374 | 6577 | 99.8% | 14 | 0 |
| 18/11 | 327 | 301 | 6014 | 99.1% | 16 | 0 |
| 24/14 | 176 | 165 | 5196 | 96.0% | 26 | 4 |

Within 30 and within 20 subtiles of the walked path: 100.0% for every pair up to 18/11 on all three levels (24/14: 99.8% within 20 on Blood Moor). Cave (synthetic) and open field show the same shape: max gap 14 at 16/10 on both; 24/14 reaches 19 on the cave and 14 on the field. Legs and points drop about 80 percent from 7/5 to 16/10, path about 50 percent.

## Why 16/10

- Every walkable cell is within 14 subtiles of a walked cell at 16/10 (max gap, all three levels), less than half of `bot.clearRadius` (30) and under the ~20 pocket limit; 24/14 breaks it (26, four pockets). 18/11 also holds (16) but has less margin for a hero that stops short of a point.
- The first-attack proxy (median 18, p90 45) says monsters are met well beyond 16, so the old grid was finer than needed by about a factor of three in distance.
- Path length falls from 13070 to 6577 hops on Blood Moor (S: roughly half the walking of a full clear).
- Caveat (U): real clears are interrupted by fights and the hero only marks a point when within cover; the simulator walks perfectly. The numbers are the best case; the 14-subtile gap leaves room. Verify with the live status line `roam: N/M points seen` after deploy, and with the area clear time.

## Detours mark coverage (V, tests)

Before this change `coverage.mark` ran only inside `roam()`, so ground passed while fighting, picking up or walking elsewhere stayed unseen and was walked again. `Navigator.NotePosition(level, pos)` (goroutine-safe, `covMu`) now marks on every observation: `Runner.trackAlive` calls it for any Mover that has it (outside town), and `useSensesCoverage` republishes the coverage at once when a point changes, so the overlay cell flips without waiting for the next nav observation. Tests: `TestNotePositionMarksDetourPoints`, `TestRoamKnobsReachTheView`. A kill near a point needs no extra rule: the hero stands within cover of it while fighting there.

## Roam memory per level (TASK roammemory)

Before: one coverage (`n.cov`) rebuilt whenever the level map changed, so a town trip, the Den or a portal trip and back started the roam from zero (V, code). Now `Navigator.covs` maps level number to its coverage; `selectCov(lm)` (called on every observation) makes the current level's entry `n.cov`, so the overlay, `Coverage()` and `NotePosition` follow the level the hero is in and a re-entered level shows its remembered cells at once. `d2dworld` keeps one `LevelMap` per level for the whole game and builds new ones after `Reset`/`ResetNext` (a new game, V `internal/d2dworld/world.go`), so the memory is keyed by that identity: an entry whose map differs is dropped (test `TestRoamMemoryPerLevelAndGame`: A to B to A keeps A, two levels independent, new map value starts empty). Nav has no class or build data, so Paladin and Sorceress behave the same.

Planner side (V, `internal/bot/goal.go`): `l.cleared`, `sweepKills` and the step state are per game already (`syncGames`/`forgetGame` on a new game), and the planner excludes cleared areas (`Exclude`/`Cleared` in the pick). A sweep that does enter a cleared area finds all points seen and ends at once (`ErrExplored`) instead of roaming again. Monsters do not respawn within a game (SME, S), so nothing else needs to clear it.

Persistence across bot restarts: NOT done. The log shows `d2d backend up ... managed:true`: the bot starts and owns the d2d process, so a bot restart is a new d2d process and a new game; there is no game identity that survives it (V log of 2026-11-11 14:23; an unmanaged d2d would differ, U). Points failed stay failed for the rest of the game.
