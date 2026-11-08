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

## walksmooth2 (TASK-196, 2026-11-08): the pauses were observation latency, measured live
Tags: V = measured in the user's dev container (hero Doofer, realtime d2d), S = read in code, U = unverified.

### Trace (dev image only)
`nav.trace` lines in the bot log (on when `DIDLE_DEV` is set or `D2BOT_NAV_TRACE=1`; never in a stamped release): `click` (from, to, dist, reason first/newwp/still/retry, snapMs/updMs of the observation it was based on, clickMs of the command, gapMs since the last click), `stall`/`resume` (two polls without movement, pause length), `plan` (ms), `leg` (one per walk: duration, path subtiles, speed, clicks per 100 subtiles, still time, polls, mean/max observe ms, callers), `move` (primeMs, runMs, cmdMs, stam, runBelieved per move command), `snapcallers` (every 30 s: who asked for a full snapshot, count and mean ms) and the existing `d2dtrace` verb table. Code: `internal/nav/trace.go`, `internal/d2denv/snapstats.go`.

### Before (first 40 minutes after the walksmooth deploy, logs; 850 ms per observe)
- Travel legs: 3.6 to 5.9 subtiles/s (walk 9.4, run 14), path/net ratio 2 to 2.5 (the walker clicked waypoints behind it because its position was stale), 10 to 12 clicks per 100 subtiles, click gaps 1.0 to 2.5 s (median about 1.2 s), 54 `nav stuck` lines in 40 minutes (13 to 19 subtiles from the waypoint: the click was long since spent), every walk start cost 2 s.
- Cause (V, `snapcallers` + `d2dtrace`): ONE observation took 500 to 1100 ms (snapshot about 340 ms, world update 16 to 166 ms, measured by `nav.trace click` snapMs/updMs and the leg obsAvgMs 800 to 900). Three readers queued full snapshots on the one control channel: the walker, the safety guard (`bot.guardOnce` -> `poll`, about 2 reads per second) and the minimap feed (`sensesFeedUnits` -> `unitSource.Units`, a second full snapshot INSIDE every second nav observe). The channel answers one line per d2d frame (V: every verb in the `d2dtrace` table averages 16.6 ms, also the eight lines of a prefetch batch, so a snapshot is about 10 frames = 170 ms alone). The walker saw its own position about one second late, so the character reached the click target and stood until the next poll.
- Not the cause (V): plan time (0 to 2 ms), `cmd move` itself (16 ms), the pointer-over-unit effect (`cmd move` is a semantic command, no pointer, S), the stuck ladder (it fires only after the pause already happened).

### Fix
1. `d2denv.Reader.Recent(maxAge)`: the last full snapshot when it is younger than maxAge.
2. `sensesUnits` (`cmd/d2bot/sensesunits.go`) uses the snapshot the nav observe just read (`recentSnapAge` 1 s) instead of a second read.
3. The safety guard (`Runner.pollWith(true)` from `guardOnce`) takes a snapshot younger than `guardPollReuse` (350 ms); the main loop's `poll()` still reads fresh (act then look).
Tests: `TestReaderRecent`, `TestUnitSourceReusesTheWalkersSnapshot`, `TestGuardPollReusesRecentSnapshot` (Sorceress and Paladin), `TestWalkTraceOffAndOn`.

### After (same container, minutes after the deploy)
Observe 400 ms instead of 850 (snapcallers: the minimap reader is gone, the guard drops from about 62 to 47 reads per 30 s with the 150 ms window; 350 ms window added after), travel legs 9.3 to 11.5 subtiles/s, click gaps 0.5 to 0.6 s, 11 to 12 clicks per 100 subtiles, path/net ratio near 1.

### Left over (not fixed here)
- Fork evidence: d2d answers one channel line per frame (16.6 ms each) even for a pipelined batch; a pump that dispatches every complete line per frame would make a snapshot about 17 ms instead of 170 ms. Card for the fork (no upstream posting): `Channel::pump` dispatch all complete lines per frame in realtime (the lab4 build did it for lockstep only, docs/d2d-pace.md).
- Short walks (one waypoint, 5 subtiles) cost about 2 s each: observe, plan, click, observe. Combat approach steps call `GoTo` once per step; a cheaper position-only poll during a walk (one `debug players` instead of a snapshot) would cut the poll to 17 ms. Not done: the guard and the monster signature need the snapshot.

## approachfast (TASK-209, 2026-11-08): position-only polls inside GoTo walks
Tags: V = measured in the dev container (hero Doofer), S = read in code, U = unverified.

### Before (V, 60 minutes of `nav.trace leg` lines, after the walksmooth2 deploy; 476 legs)
- Combat legs shorter than 10 subtiles (28): median 1.08 s, mean 1.49 s, 2.7 polls, observe mean 218 ms, so about 0.6 s of a 1.5 s step was observing; 17 clicks per 100 subtiles (median). Combat legs of 10 subtiles or more (33): median 3.0 s, 9.9 polls. Whole sample: 17 clicks per 100 subtiles, 21 stalls, 22 `nav stuck` lines.
- A full snapshot is still about 150 to 200 ms (`snapcallers`: walker observe mean 170 to 195 ms, max 350 to 700 ms).

### What exists on the channel (S)
No new verb is needed. `d2dworld.World.Update` (the walker's second call in every observe) is already ONE `debug level` line: level id, position, size (one d2d frame, about 17 ms; rooms are refetched only when due). So the position-only poll is the world update alone.

### Change
`Navigator.observeFast` (`internal/nav/walk.go`): between two FULL reads a `GoTo` walk (`walkOpts.fastPoll`, fixed goal, no `stopWhen`, no unit following) calls only `World.Update` and moves the player in a copy of the last full observation (delta of the world position, so the coordinate frame cannot drift). Monster list, life, UI and the guard input are those of the last full read, at most `fullPollEvery` (500 ms) old. A full read is taken at the start of every walk, when the level or map changed, when the position jumped by more than 40 subtiles, when the character stood still for two polls (the stuck ladder and the stall trace need real data), after the panel close and at the start of every stuck-ladder rung. The stuck detector, the arrival test and the stop-short logic work on the polled position unchanged. Roam, explore, exit and unit walks keep full reads. `Navigator.NoFastPoll` is the rollback and the test baseline; `nav.trace leg` carries `fastPolls`. Class-free: no class, skill or build data involved.
Tests (`internal/nav/approachfast_test.go`): full snapshots per approach step Paladin melee (6 subtiles, walk speed) 5 to 1, Sorceress ranged (12 subtiles, run speed) 8 to 2; the stuck detector still returns `ErrStuck` for both; one full read per 500 ms on a long walk; `go test -race ./internal/nav` clean.

### Fork card (not done here, private fork only)
`Channel::pump` dispatches one complete line per d2d frame (V: every verb averages 16.6 ms, also the eight lines of a pipelined batch), so a snapshot costs about 10 frames. Request: dispatch every complete line of the input buffer per frame in realtime (as the lab4 build already does for lockstep, docs/d2d-pace.md); a snapshot would drop from about 170 ms to about 20 ms and make the whole fast-poll split unnecessary. No upstream posting.

### After (V, same container, 11 minutes after the deploy of main 0.0.261108211221; `nav.trace leg`)
- 121 legs, 766 polls of which 131 were position-only (17 percent; most legs are short, so their first poll is the full one). No `nav stuck` line, 7 stalls (before: 21 in 476 legs).
- Combat legs under 10 subtiles (15 samples, few): median 1.0 s (before 1.08 s), observe mean 167 ms (before 218). Combat legs of 10 subtiles or more (15): observe mean 127 ms (before 220), median duration unchanged (2.95 s: speed bound). Clicks per 100 subtiles 16.0 over all legs (before 17.1).
- Reading: the long approach legs gain (observe about 40 percent cheaper); a one-waypoint step is dominated by its first, full observe plus the fighter's own snapshot just before it, so the 2 s per step does not go away with fast polls alone. Next steps, not done: (1) start a `GoTo` from the fighter's snapshot (`Reader.Recent`, about 100 ms old) instead of reading again; (2) the fork card above (one line per frame), which makes every full read about 20 ms.
