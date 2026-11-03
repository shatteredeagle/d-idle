## Why

Gate 1 (ms3) failed on two windows: seed 17 had a 17 virtual minute no-kill stall (minutes 299-315 in Stony Field, then Burial Grounds) and seed 19 a belt window 158-175. docs/research/codereview2.md found two high bugs in the walk and target code (R1: the walk limit does not hold when a waypoint jump fails or is skipped; R2: a target whose walk times out is picked again).

## What Changes

- R1: `longestWalk` (internal/bot/task.go) takes the jump rules of `travelTo` (UseWaypoints, the failed-waypoint map, the minimum jump of max(2, nav.wpMinDistance) hops); without a usable jump the whole route counts as walked.
- R2: `LevelTo` skips the target on every travel error except a context error, a death or `errAreaGated`, for navSkip x 1, 2, 4, 8 on repeats (`targetFailSkip`, internal/bot/areaskip.go); a good walk resets the count.
- Findings and proposed cards for the seed 19 window and the seed 17 stall are in the report, not fixed here.

## Impact

Leveling picks that rely on a waypoint jump are refused sooner when the jump cannot happen, and a failing exit no longer repeats for up to 22 minutes. No spec number changes.
