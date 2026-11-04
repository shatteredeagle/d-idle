# fixemptyplan: a goal that snaps onto the player's cell

> status: offline fix, live proof pending. Card: [fixemptyplan](archive/openspec-index.md#fixemptyplan). Evidence: docs/probes/postfix-soak2-analysis.md F4.

## Cause

- VERIFIED (soak 2 log counts and an offline reproduction by the analyst): `combat move failed: no path: empty path` 1,191 times (1,185 in combat), 90 percent from the out-of-mana back-off (3,872 back-offs, 28 percent failed). Cost about 71 s (0.1 percent of kill time).
- VERIFIED OFFLINE (`TestBlockedGoalSnapsOntoStart`): when the goal cell is blocked, `planHard` snaps it to the nearest walkable cell; if that is the player's own cell the plan has no waypoints and no error. `PathTo` returned `(nil, nil)`, so `Fighter.valid` accepted the point; `walk` then failed with `no path: empty path`.

## Semantics

- A goal that snaps onto the start means "already as close as possible": `input.ErrAlreadyThere` (alias `nav.ErrAlreadyThere`). The error also matches `nav.ErrNoPath` and keeps the `empty path` text, so older callers behave as before.
- Back-off (kite, flee, dodge, retreat, out-of-mana step): a zero-length path is a FAILURE to retreat. `Fighter.valid` is false, so `awayPoint` takes its next candidate (3 distances x 10 angles, at most 30 probes, no loop); when none is valid it returns not-found and the caller falls through to cast, melee, potion or escape at once.
- Approach (`validApproach` for firing positions, `assess`): an arrival, so valid / reachable.
- `Fighter.move`: an already-there GoTo is a quiet debug line, not `move failed`; it still costs one tick (lockstep clocks only move on sleeps).
- `exit.go` matches the typed error instead of the error text (same behaviour).
- Everything else (real no-path, partial, unknown map) is unchanged.

## Tests

`go test -p 1 ./internal/nav ./internal/combat ./internal/input`: `TestBlockedGoalSnapsOntoStart`, `TestSnappedStartEdgeCases` (nav); `TestValidApproachVsBackoff`, `TestAwayPointSkipsSnappedCandidates`, `TestMoveAlreadyThereIsQuietTick` (combat). VERIFIED OFFLINE. UNVERIFIED LIVE: the real planner's snapping next to a real wall with a real pack.

## Live recipe

Spawn a pack near a wall or obstacle (`debug spawn`), drain mana (`debug stat`), fight. Expect: no `empty path` in the log, back-offs go to a different cell or are skipped, and the bot falls through to its next option. Compare the back-off failure rate with soak 2 (28 percent).
