# Stale target area, border-loop bans and blocked journey steps (TASK-398)

> status: fix landed on branch agent/staletarget (not deployed; no live run). Tags: V verified (how is said), S secondary, U unverified. Evidence: docs/probes/hero14-analysis-2026-11-14.md (branch agent/hero14) and the hero14 container log.

## Timeline from the log (V, grep over the 56 MB log, counts only)

- 19:04:29Z a save point re-enters the game (Rogue Encampment, clvl 14). 19:04:34Z the quest step `quest.search_for_cain/to-wood` is refused: `area not allowed now: Blood Moor (2): banned after repeated deaths` (first of 47,075; then Stony Field 42,572). Retry at 0.08 s, then outcome `blocked` every 0.2 s: 89,647 lines.
- The bans came from `area skipped ... reason border-loop` (997 Blood Moor, 788 Stony Field, 2,880 s each), not from deaths (deaths 0).
- Root cause (V, reproduced on origin/main with a test, see below): `ob.targetArea` is set by the leveling slice (goal.go target, task.go setTargetArea) and never reset when the slice ends. The quest step then waypoints into Dark Wood (area 5), state Clearing; `replanCause` sees `Player.Area != targetArea`, returns "exploring crossed an area border" on every guard tick and calls `noteCrossing`; five stamps in a minute trip the breaker. Reproduction: 20 guard ticks in Dark Wood with target 2 give 20 cancels and a `border-loop` ban on area 2.
- The refusal text was always "banned after repeated deaths" (`routeAllowed` and `areaAllowed` ignored the stored reason), and `routeBanned` only forgave nav-fault bans, so a walk that a waypoint would have made without touching Blood Moor was refused.
- A blocked group left `Plan.Stuck`, which `taskStep` turned into `selfPause` (262 min paused, 54 min "planning blocked"), with nothing farmed.

## Fix

1. `staleOrCrossing` (staletarget.go): an arrival from a town is a jump, so the leftover target is dropped (log `stale target area dropped`), nothing is cancelled. A real crossing stamps once per arrival. `noteCrossing` ignores a second stamp for the same target within `borderLoopGap` (10 s). `taskStep` also clears the target at the start of every quest tick (a clearing step sets its own).
2. `banReason`: refusals and the orders map name the real reason (`skipped (border-loop, not a death)`). `routeAllowed` ignores non-death bans on hops the waypoint trip jumps over (`jumpedHops`); `routeBanned` never lets `border-loop` block an intermediate hop; while a quest step walks (`questTravel`) a border-loop ban on its own area is ignored too, and `World.Banned` leaves border-loop areas out: the quest step wins, log `quest step wins over the border-loop ban`. Deaths (`death-streak`, `repeat-site`) still block everything.
3. Spin guard (`spinGuard`): 5 identical refusals in a row, then wait 1 s, 2 s, ... up to 60 s, one log line per doubling.
4. Blocked journey (`journeyBlocked`): a blocked plan or a danger refusal starts a clock (log `journey blocked: <reason>`, status field `journeyBlocked`, why code `journey_blocked`). Before `bot.questBlockedFarmMin` (10) the loop waits with the spin guard. After it, the hero farms the level plan with the same pick as the level steps (`LevelTo(level+1)`: build and class data from the plan table, no skill or class literal), and retries the quest step after 10 min, 20, 40 ... (cap x64); each retry lifts the group's block (`Executor.Unpark`). A step that runs clears everything.

Build-specific: none. The level pick is the plan table's (`docs/levelplan.md` bands); tests run Sorceress and Paladin.

## Tests (internal/bot/staletarget_test.go, internal/task/executor_unpark_test.go)

Stale target dropped on town arrival, real crossing counted once, `noteCrossing` rate limit, border-loop ban does not refuse the route / death ban still does / real reason, quest step wins, spin guard backoff, blocked journey farms then retries with a growing wait. V: the stale-target scenario fails on origin/main (20 re-plans, ban on area 2).

## Not verified (U)

No live run: a clvl 15 Paladin in town with Blood Moor and Stony Field banned by `border-loop` must take the waypoint to Dark Wood and stay (kills above 100 in 10 minutes, `action cancelled` 0, `area skipped` 0). Waypoint hops from a field to a field (not from town) with a stale target are covered only by the per-tick target reset of quest steps and by `LevelTo` (which already sets target 0).
