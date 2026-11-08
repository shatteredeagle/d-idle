# walksmooth: the stop-and-go walk (TASK-188)

> status: offline fix merged-ready, NOT proven live. Tags: V verified (how stated), S read in code, U unverified.

## Report
The user's first minutes with a Paladin (realtime container, 2026-11-08): "walks 5 steps, waits a second, walks 5 steps, along the wall".

## Evidence from his session log (V, read only: container log of the first 6 minutes)
- Every `nav stuck` line but one (5 of 6) had the character standing 5.0 to 5.8 subtiles from the waypoint it was clicking; the normal waypoint reach is 4 (`min(6, max(2, nav.waypointSpacing * 0.5))`). The walk therefore waited for the stuck detector (`nav.stuckMs` 1200) and recovered with `reclick`.
- Over the first minutes the character covered about 1.4 subtiles per second against 9.4 for a walk (positions in the `status` and `nav stuck` lines; includes fights and scans, so an upper bound on idle share only). `killtime` buckets: 190 of 282 s between the first two kills were `wait`.
- `minute` lines: `stam` equals `stamMax` in every line of five minutes of travel, `runBelieved` true, `runKeys` 1. Running drains about 5 percent of the bar per second outside town (docs/d2d-run.md, V lab), so the Paladin was most likely WALKING (U: not read directly; the toggle is not readable). Walking is 1.5x slower than running.
- Not the cause (S): the exit penalty only affects levels with a non-wanted exit; the start walk goes to the wanted exit.

## Root cause (U for the exact mechanism)
d2d ends a `cmd move` short of its target (docs/d2d-backend.md: "path stops short"), here by 5 to 6 subtiles, and the nav only advanced to the next waypoint inside 4 subtiles, then re-sent the same target (a no-op) until the stuck ladder fired. Each such stop costs 250 ms to 1.2 s. Second contributor: the run belief can start wrong.

## Fix (class-free, no new knob)
1. `internal/nav/walk.go`: a standing character (two polls without moving) counts a waypoint within `0.8 x nav.waypointSpacing` as reached and clicks the next one at once (`stopReachSpacings`). The stuck ladder keeps working from the normal waypoint (`walkState.base`) so recovery behaviour is unchanged.
2. Same file, `lookahead`: the click goes to the farthest waypoint within `2.5 x nav.waypointSpacing` that the planned path leads to in a straight line (intermediate waypoints within 2 subtiles of the chord), that is in line of sight and that the mouth gate allows. The next move is issued before the previous one ends. The old behaviour re-clicked every 250 ms while a delayed command was pending (28 vs 15 clicks per 100 subtiles in the sim).
3. `internal/d2dinput/run.go`: if the belief says running, the character has moved continuously for 4 s outside town and stamina never left the maximum, press the run key once more (at most 2 times per game). Town is excluded (no drain there).

## Tests and before/after (offline sim, NOT live)
`go test -p 1 ./internal/nav -run TestWalkIsSmooth` (sim with a stop-short and command-lag model, walking speed 9.4 and running speed 14), `go test ./internal/d2dinput -run RunBelief`.
Before the change the stop-short cases end in `ErrStuck` (3 of 5 cases fail; the rest pass); after, pauses over 500 ms per 100 subtiles are 0.00 and the longest pause 0.1 s in all five. The ghost-pillar recovery sweep (21 start rows) went from 13 to 14 passes.

## Not done / needs the game
- The live timing of `cmd move` in realtime was not measured: a realtime debug run on a worker slot was refused by the permission system in this session. Recipe: `docs/probes/walksmooth/run.sh w1 7 4m` on the worker (realtime, debug log: `nav click` lines carry position and waypoint) and compare click intervals with `nav stuck`.
- Whether the Paladin really walked (U): read `stam` in the minute lines after the update; with the heal it must drop below `stamMax` during travel.
