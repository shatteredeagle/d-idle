# Explored share: when a zone counts as explored (bot.exploredPct)

> status: CURRENT. Tags: V = read from the code in this worktree and covered by a unit test, U = not run in a live game.

## Where "explored => cleared => move on" is decided (V, code read)
1. `internal/nav/roam.go` `roam()` visits the level's coverage points (`nav.roamSpacing`, `nav.roamCover`) nearest first and returns `ErrExplored` when no unvisited, unfailed point is left (or the leg cap is hit). `Navigator.Coverage()` returns visited and total.
2. `internal/bot/goal.go` `sweepPass`: `Mover.Explore(ctx, done)` ends (ErrExplored is success); when nothing hostile is near the bot sets `l.cleared[area] = true` ("area cleared") and `logSweep`. Quest runs only finish if `sweepEndsQuest` agrees.
3. The planner (`goal.go`, `l.cleared[a]`) then excludes the area and takes the next one or a fresh game; `task.go` `Clear` loops while `cleared` is false.
Other paths that also set `cleared` are unchanged: `emptyAreaLeave` (no kill for a while), `exploreGaveUp` (two explore failures).

## The knob (V)
`bot.exploredPct` (percent, default 100, range 30..100, `internal/env/tuning_bot.go`, listed in `docs/knobs.md`). `sweepPass` adds `exploredEnough(area)` to the `done` callback of the roam: once visited/total reaches the threshold the roam ends, and the existing code marks the zone cleared. 100 (and 0 or anything above 100) means every point, exactly the old behaviour; below 30 is raised to 30. Quest and boss runs (`keepRoaming`) ignore it. A Mover without `Coverage()` or with 0 points (no fully known map) is never cut short. No class or build is named anywhere (V: one rule, tested with a fake Mover; the watch part is tested for both a sorceress and a paladin rig).

## Display (V unit-tested, U in the browser)
- `/api/hero` gains `explored: {visited, total, pct, threshold}` (additive, `watch.hero/1` unchanged; omitted in town, with 0 points or without a source). Source: `Runner.Explored()`, bound in `cmd/d2bot` next to the stats source.
- Activity panel header (existing panel): "Blood Moor - Explored 63% (moves on at 80%)"; the bracket shows only below 100.
- Logs: the `minute` line has `explored` (percent, -1 without coverage); `sweep: coverage ...` has `exploredPct`; "explored enough" is logged when the threshold ends a roam.

## Not done
No live run; the percentage in the UI is computed from the same points the roam uses, so it can read 100 only when every point is visited or failed-and-skipped counts as unvisited (failed points keep the zone short of 100 even when the roam gave up: U).
