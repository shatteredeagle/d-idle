# ROAMLEG: a mouth-gate hold is not a failed roam target

> See also docs/roamspacing.md: the roam grid is now the knobs `nav.roamSpacing`/`nav.roamCover` (default 16/10, was 7/5).

> status: offline fix, tests pass, verified live in DRILLS23 (docs/probes/drills/README.md): 0 failed legs (was 50), 11 step-aways per run (was 62).

## Finding (V, DRILLS21 in docs/probes/drills/README.md, logs `2026-11-01-drills21-*.txt`)
After fixmouthloop2 a held step-away makes the outer walk return `ErrMouthGate`. `ErrMouthGate` wraps `ErrNoPath` (internal/nav/mouthgate.go), so the roam leg switch folded it into the failed-target case: 10 failed legs and 10 step-aways per episode, 5 episodes, 50 `roam leg failed` lines per DW.m3 run.

## The loop (V, read in internal/nav/roam.go at origin/main c8df777)
`explore` calls `roam` when the level is fully known (explore.go:46). `roam` loops: `done()` check, deadline (`ExploreLimit`), observe, guard, coverage points (`c.pts`, built once per level, `seen`/`failed` per point), pick the nearest unseen, unfailed point by flood hops, then `c.legs++` and one `n.walk` to it (`walkOpts{stopWhen, timeout: 90s}`). The result switch:
- `nil`: mark the point seen.
- `ErrStuck`, `ErrNoPath`, `ErrTimeout`: `c.failed[pick] = true`, log `roam leg failed`, next point.
- anything else: return it.
The loop ends with `ErrExplored` when no point is left (or the leg cap is hit). Callers: `Clear` and the sweep treat `ErrExplored` as "area explored" (internal/bot/goal.go, task.go), exit.go and waypoint.go as "nothing found here".

## What a mouth-gate hold means
walk.go steps the character away from a guarded cave mouth once per walk; the gate (mouthgate.go) rewrites the clicks at most `gateMaxRewrites` times and then refuses (`ErrMouthGate`) because every click would cross the trigger area of an unwanted exit. The character stands inside the guarded zone and the gate will not move it nearer the exit. That is a property of WHERE THE CHARACTER IS, not of the target: the next target starts the same step-away from the same spot and is held the same way. Marking the target failed was wrong twice: it burned a step-away per target, and it threw away good targets (the coverage memory outlives the episode).

## Design
In the roam leg switch, `ErrMouthGate` is matched before the `ErrNoPath` case (it still wraps it): the target is NOT marked failed, `coverage.held` is counted, one debug line `roam: held by the mouth gate, ending the episode` is logged, and roam returns `ErrRoamHeld`. `ErrRoamHeld` wraps `ErrExplored`, the result the old code reached after failing every target, so no caller changes: the sweep falls through to what it did before (area cleared, then the leftover hunt through the Fighter, which has its own hold back-off in internal/combat/mouthhold.go). Callers that care can test `errors.Is(err, nav.ErrRoamHeld)`; `Navigator.RoamHeld()` reports the count. Not changed: which points are sampled, the exit-near filter, when the bot leaves an area, the gate semantics, `ErrStuck`/`ErrNoPath`/`ErrTimeout` handling, wanted exits (the gate and the step-away are not engaged, existing TestWantedExitNoStepAway).
Trade-off (U): the first hold ends the episode where the old code needed about 10 failed legs; the terminal result is the same, the cost is 1 leg instead of 10. The next episode retries the same target.

## Tests (internal/nav/roamleg_test.go, V offline)
- every leg `ErrMouthGate`: new code runs 1 leg, 0 failed targets, ends with `ErrRoamHeld` (also `ErrExplored`); second episode retries (legs 2, failed 0). The old code ran 320 legs and failed 320 targets in the same sim (the NoPath test below is that old path).
- every leg `ErrNoPath`: plain `ErrExplored`, targets marked failed (320 legs), held 0.
- the existing roam, mouth-step and wanted-exit tests pass unchanged.

## Live recipe (not run)
On the worker host, hidden slot 0, lab build, `d2drill` as in DRILLS21: `-group F -only DW.m3 -label roamleg-m3N -v`, three invocations. Expected (U): `roam leg failed` lines 50 -> a handful (only non-gate failures), `stepping away from a cave mouth` per run 62 -> about 20 or fewer, mouth bounces stay 0, `walk did not finish` stays 0, D3 12/12 and D2 11/11 unchanged. Count `roam: held by the mouth gate` lines (debug level) to see episodes ended.

## Shared-file lines for the merger
- docs/BACKLOG.md: ROAMLEG: roam leg held by the mouth gate no longer marks its target failed; the episode ends with ErrRoamHeld (wraps ErrExplored). Offline done (docs/roamleg.md); live proof pending (DW.m3 x3).
- docs/INDEX.md: docs/roamleg.md (roam leg and the mouth gate).
- CHANGELOG: roamleg: a mouth-gate hold ends the roam episode in one leg instead of failing ten targets.
