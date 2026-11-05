# codereview3: independent review of GATE1FIX, RECOVERYKIT and XPFALLBACK

> status: archived 2026-11-05 (cleanup B): finished or superseded, kept for the record. Old path: `docs/research/codereview3.md` (now a stub). Earlier status: current. Offline review (no game, no code change) of the three merges that the running Gate 1 ms4 relies on: 29910a4e gate1fix, fd48c320 recoverykit, 75c9ef5b xpfallback. Context: docs/research/codereview2.md (R1 to R9).

Tags: V = read in the code at origin/main (file:line) or run here, U = not verified (no live run, path not traced to the end). Line numbers are origin/main at 7b7c0f90.

Test run here (V): `go test -p 1 -short -count=1 -v -run TestMatchesDocs ./internal/plandata/` reports SKIP ("docs/plan/data not present"), see C4.

## Findings, most severe first

### C1 (medium) A target skip bans every target routed through it, also the ones a waypoint would reach (gate1fix + routeBan)

`LevelTo` (internal/bot/task.go:920-926) now skips the target on any travel error (before: only "no exit"), for navSkip x 1, 2, 4, 8 = 150, 300, 600, 1200 s (NavSkipSec 150, internal/env/tuning_bot.go:74). The skip is an `avoid` entry, and two older rules treat every `avoid` entry as a ban on the walk:
- `routeBan` (internal/bot/goal.go:654-677) walks `Mover.Route(cur, target)`, the exit graph, and skips any target whose route crosses an avoided area, for as long as that area's skip (V). The route is the exit graph, so it does not care that `travelTo` would jump over the area by waypoint.
- `routeAllowed` (internal/bot/geargate.go:143-149) refuses the walk with `errAreaGated` and the text "banned after repeated deaths" (V).
- The skip also lands on the target when an earlier hop failed: an error "exit to Stony Field: ..." while walking to Black Marsh skips Black Marsh, not Stony Field (V task.go:924; the hop is only in the error text).

Scenario (V code, U live): Normal clvl 8, Black Marsh (6) waypoint active, target Stony Field (4). The walk times out at the Cold Plains to Stony Field exit (nav.ErrTimeout) a 3rd time in the run: Stony Field skipped 600 s. The next picks Dark Wood (5), Black Marsh (6) and Tamoe (7) are all "route crosses a banned area" for those 600 s, although town -> Black Marsh is one waypoint jump. The planner is left with Blood Moor, Cold Plains and the Den, or (with C2) up to 1200 s.

Fix: in `routeBan` and in the `r.banned(h)` branch of `routeAllowed`, ignore a hop whose `avoidWhy` is `whyUnreachable` or `whyNavStuck` and that is not the target (a target skip is not a danger ban); or check the hops actually walked (the `longestWalk` jump) instead of the whole exit route. Test: Stony Field skipped `whyUnreachable`, Black Marsh waypoint active, `target()` from town must return 6.

Minute-line signal: after a "leveling area unreachable ... skipSec=600|1200" for areas 3, 4 or 5, minute lines in areas 1 to 3 (or town) for that long with low xp per minute, and bot-0.log "target skipped: the route crosses a banned area" for 5, 6, 7. A route refusal mid-walk logs "banned after repeated deaths" with no death before it.

### C2 (medium) The per-target fail count never decays

`targetFails` (internal/bot/areaskip.go:48-64) is cleared only by a successful `travelTo` to that same target (task.go:927-928). It survives games, hours and recovery. It is not cleared when the character is already in the target (the `s.Player.Area == target` branch, task.go:913, never calls travelTo) or reaches it by another pick.

Scenario (V arithmetic, U live): Dark Wood fails 3 times in the first hour (150, 300, 600 s), then the bot levels elsewhere and enters Dark Wood later only as an intermediate area. Hours later one transient "did not arrive within 30s" skips Dark Wood for 1200 s at once, and with C1 also every target beyond it.

Fix: keep the time of the last failure with the count and restart at 0 when it is older than twice the last skip (or 30 min); clear the count when the snapshot shows the character in the target. Test: three failures, clock + 1 h, one failure must give navSkip, not 8 x navSkip.

Minute-line signal: a "skipSec=1200" (or 600) on the first "leveling area unreachable" line for an area in a stretch with no failure for that area in the previous 30 minutes.

### C3 (low, U) The recovery kit trip does not count a failed way to town

`recKitTrips` grows on a finished trip (internal/bot/resource.go:110-112) and on failed chores (goal.go:479-483), not on a failed `ToTown` (goal.go:453-461). A failed ToTown abandons the trip without `setLastTrip`, so `tripDue` (goal.go:1365-1383) does not hold the next one back, and the trip is emergency (gap / 4). Scenario: recovering, 0 potions, 40 gold, no scroll, in Cold Plains (`ChainHomeArea` true, so `WayHome` is true) and the walk home fails: the same trip is due again on the next tick. Pre-existing for every emergency trip; new is that recovery now asks for one from the field (before, `broke` suppressed field potion trips, goal.go:332-335). The failsafe ladder (`l.failures`) is the only stop (U whether it fires, the abandon resets `l.failures` to 0 at goal.go:458).

Fix: count a failed ToTown of a `RecoveryKitReason` trip in `recKitTrips` as well. Test: a Town fake whose ToTown fails, KitTripCap 2: the third choresDue must not ask for the trip.

Minute-line signal: step alternating "Going to town (... recovery restock: no healing potion, ...)" with no area change to 1 and no gold change, potions 0.

### C4 (low) The plan flag exists only in the embedded copy; the copy check always skips

`xp_fallback_full_kit` is in internal/plandata/progression-normal.json (V lines 30, 81, 89), not in docs/plan/data/progression-normal.json (V grep: 0), which plandata.go calls "the source". `TestMatchesDocs` (internal/plandata/plandata_test.go:9-24) skips the whole test at the first embedded file missing in docs/plan/data, and the `*_endgame90.json` files live in docs/plan/data/builds-endgame/, so it always skips (V run above). Live is not affected: no config sets `planDir` (V grep of cmd, internal, tools), so LoadPlan reads the embedded copy (plan.go:92-103). The next "copy docs/plan/data to plandata" silently drops R5.

Fix: add the three flags to docs/plan/data/progression-normal.json; in the test, look up `*_endgame90.json` under builds-endgame/ and skip only when the directory is missing.

Minute-line signal: none (live reads the embedded copy).

## No findings

- gate1fix R1 (`longestWalk`, task.go:982-999, `unreachable` task.go:1026-1032): matches `travelTo` (goal.go:1050-1078): jump only from an activated waypoint or town, to a target not failed lately, with `UseWaypoints`, at least max(2, WpMinDistance) hops from the start of the jump (V). Where it differs (a later jump from a second waypoint after the first one's jump failed) it refuses more, not less (V).
- gate1fix R2 stop conditions: a guard cancel, a step timeout, a stuck check or a trip interrupt cancels the action context (runAction, loop.go:313-358), so `ctx.Err() != nil` and no skip is charged to the target (V). Skips expire through `banned()` (goal.go:615-625) and `idleBreak` (V).
- recoverykit: no loop on success. A trip that buys a potion makes `Healing > 0` in the field, which ends the rule and resets the count (goal.go:339-341); with the potion drunk the rule fires again at gold >= KitPotionGold, one trip per potion, by design (V). In town the shop opens through line 377 (`p.InTown() && healing == 0`), capped by potionFloor = RecoveryPotionFloor 2 (broke.go:118-123) (V).
- xpfallback R5 to R7: the flag is read with the right key from the progression table (plan.go:256, `rawObj.boolean`), applied in `pickByXP` and in the plan-order pass only when the fallback ran (plan.go:391-407) (V). An unknown level ranks out; no estimate at all keeps the plan order (plan.go:469-477) (V). Ties go to the later step (V).
- Interactions: a target skipped by gate1fix makes the preferred step excluded, so the XP fallback runs, and Burial Grounds, Crypt and Mausoleum stay out of it without a full kit (V). A recovery kit trip during a walk cancels the action context, so it never adds a target skip (V).

## What the running Gate 1 ms4 should be checked for

- C1/C2: in bot-0.log, "leveling area unreachable" with skipSec 600 or 1200 for areas 3 to 5, then "target skipped: the route crosses a banned area" lines; in the minute lines a stretch of the skip's length in areas 1 to 3 or town with low xp per minute.
- C3: minute lines repeating "Going to town (... recovery restock ...)" without reaching area 1.
- Proof the fixes work: no target with 5 or more "leveling area unreachable" in a row; no minute line in 17, 18 or 19 with healing under KitMinHP; in recovery, potions back above 0 within a few minutes of gold reaching 30.
