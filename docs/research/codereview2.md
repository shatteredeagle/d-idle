# codereview2: independent review of AREAREACH, CODEFIX and SPECDELTA

> status: current. Offline review (no game, no code change) of `git log a45b9af4..origin/main -- internal cmd` (010e13a7 areareach; 510984c1, a652428b, 0710e303, f7b8133c codefix; d0baa39f, 6c19b2e3 specdelta). STASHFIELD (875cd7cd) is at the base a45b9af4 and was read only where SPECDELTA uses it.

Tags: V = read in the code at origin/main (file:line) or computed from it, or read from data as said; S = from a repo doc; U = not verified (no live run, path not traced to the end). Line numbers are origin/main at the time of review.

Data read for this review (V): levels.txt from the emulator MPQ stack on the worker host (`tools/emu/run-on-worker.sh mpq-table.sh levels`), Normal `MonLvl1Ex`: Crypt (18) 3, Mausoleum (19) 3, Burial Grounds (17) 3, Cave Level 2 (13) 2, Tamoe Highland (7) 8. Hell `MonLvl3Ex`: Crypt 83, Mausoleum 85, Black Marsh 69.

Tests run here: `go test -p 1 -count=1 -run 'TestPickFallbackByXP|TestLongestWalk|TestDarkWoodReachable|TestUnreachableLevelingTargetIsRefused' ./internal/bot/` passes (V). The codefix claim "each test fails when its fix is reverted" was NOT re-checked (U).

## Findings, most severe first

### R1 (high) The walk limit does not hold when a waypoint jump fails or is skipped

`unreachable` (internal/bot/task.go:1017) counts every activated waypoint as a free jump (`w.WPs[a] || IsTown(a)`), but `travelTo` (internal/bot/goal.go:1028-1046) jumps only when (a) `Tune.Nav.UseWaypoints` is on, (b) the jump skips at least `max(2, WpMinDistance)` hops, (c) `wpTripWorthTrying` passes and (d) the waypoint did not fail in the last `wpFailedFor` = 10 min (internal/bot/waypoints.go:35). If any of these is false, travelTo walks the whole route and the 5-hop cap is not enforced.
- Scenario (V from the code, U live): town, Black Marsh (6) active, target Outer Cloister (27) once the Monastery guard opens. Route about 1 -> 2 -> 3 -> 4 -> 5 -> 6 -> 7 -> 27 (7 hops, U exact). longestWalk = 2 (tail after Black Marsh), accepted. One failed waypoint click marks it failed for 10 min, and every pick in those 10 min walks all 7 hops. The same happens with `nav.wpMinDistance` set to 4 (knob range 2 to 8): any jump shorter than 4 hops is walked and not counted.
- Also: longestWalk returns `max(lead, tail)`, not `lead + tail`. Lead 5 plus tail 5 = 10 walked hops passes. The doc comment ("the longest stretch") says so, but AREAREACH and the card describe it as "walks up to 5 hops".
- Smallest fix: build `wp` in `unreachable` from the same predicate travelTo uses (UseWaypoints, `wpTripWorthTrying`, the failed map), apply WpMinDistance to the jump, and compare `lead + tail` (or each stretch, but say so) with walkableHops. Test: one active waypoint marked failed, then the 6-hop pick must be refused.

### R2 (high) Target -> walk -> nav timeout -> same target can loop; at worst about 22 min per round with no skip

Only an error that contains "no exit to that level" skips the target (task.go:920-925). A nav timeout (`nav.ErrTimeout`, "leaving for level N", internal/nav/exit.go:40, deadline 2 x WalkTimeoutSec = 240 s) or "did not arrive within 30s" (goal.go:999) is returned as a plain error. The next slice gets the same target again: `l.stepArea` still holds it (goal.go:814) and `unreachable` still accepts it. Movement bumps progress (internal/bot/observe.go:458-461), so `stuckCheck` (traveling 240 s) does not fire while the character shuffles around a bad exit. The target is skipped only through `fail()` after MaxStepFailures = 5 consecutive failures, then failsafe rung 1 `skipTarget(whyNavStuck)`, skip of 150 s (loop.go:430-438, 549-552).
- Worst case per round (V arithmetic, U live): 5 x (240 s ExitTo + 30 s waitArea) = 22.5 min, then 150 s elsewhere, then the same pick again. Three real-progress events (kills on the way or in the 150 s fallback, `realResetMin` = 3, loop.go:496) reset the ladder (loop.go:532-534), so rungs 2 to 4 (town, new game, restart) may never come. Any successful action in between (a town trip) resets `failures` (loop.go:366), so the 5-failure count may never be reached either. Then the walk fails forever. Not proven that the task runner's LevelTo errors reach `do`/`fail` (U).
- The new rule makes this more likely: before, routes over 3 hops from a field were refused; now 4 to 5 hop walks from Den, Crypt or Cave Level 2 cross 3 to 4 area borders per try.
- Smallest fix: in LevelTo (task.go:919-926), skip the target for `navSkip()` on every travel error that is not a context error, a death or `errAreaGated` (not only "no exit"), and keep a per-target fail count that doubles the skip (150 s, 300 s, 600 s). Test: a mover whose ExitTo always times out must see the pick change after one failure and not come back within the skip.

### R3 (medium) Longer walks with no exposure gate: empty belt, dead merc, low life

`unreachable` and `routeAllowed` (internal/bot/geargate.go:130-153) check bans, the Monastery and tower guards and the gear gate per hop. They do not check the belt, the merc or life. `travelTo` waits for the merc only after a waypoint jump (goal.go:1050). The kit trip (`kitTrip`, kitgate.go:96-131) applies only from KitGateLevel up, in Normal act 1, and only when the purse pays the potion; the recovery kit gate (goal.go:774) only in recovery.
- Scenario (V code, U live): clvl 10 Sorceress in town, 0 healing potions, 20 gold (under KitPotionGold), merc dead, no waypoint active. Pick Black Marsh (enter 10). Old rule: refused (5 hops, no waypoint). New: longestWalk 5, accepted, and the bot walks Blood Moor -> Cold Plains -> Stony Field -> Dark Wood -> Black Marsh without potions or merc. Intermediate areas in the act 1 chain have monster levels at or under the target (Normal 1, 2, 4, 5, 6, V levels.txt), so no hop is above the target, but the walk is 4 fights with no healing.
- Smallest fix: accept a walk over 3 hops only when the old short-walk conditions hold for danger: at least `KitMinHP` healing carried or a portal source, and the merc alive when the plan says the build needs one. Otherwise keep the old refusal. Test: the same pick refused with 0 potions and accepted with a full kit.

### R4 (medium) "travelTo activates every waypoint it passes" is overstated

`touchWaypointHere` (waypoints.go:112-131) runs once on arrival in each area. It needs the waypoint within `wpTouchMaxDist` = 70 subtiles of the entry point and no hostile within ClearRadius, and it allows 3 tries per area. A waypoint further from the entrance is not touched on the way (sweeps touch it later, sweepPass goal.go:1113). So the next pick from a dungeon still faces the long lead walk, and R1/R2 apply again.
- Smallest fix: doc text only ("activates a waypoint near the entry of each area it passes"), or a travelTo detour to the waypoint when the area has one and it is not active (more than a few lines).

### R5 (medium) XP fallback replaces plan order whenever the route guard is closed, and prefers Burial Grounds (Blood Raven) over Cave Level 1

Pick runs the XP fallback when the step that would be picked with NO exclusions is excluded (plan.go:385-393). The route guard, the tower guard, the Den gate and MercGoldFarm all go through `Exclude` (routeguard.go:125-142). From clvl 10 to 17 in Normal, the unexcluded preference is a guarded step (Tamoe Highland 7 from 10, then Outer Cloister 27, Jail 29, Catacombs 2 35; plandata progression-normal.json), so while the guard is closed the XP rank is the normal path, not a rare fallback. That is in the doc comment ("guarded"), but it means every Normal act 1 pick from clvl 10 to 17 now depends on the areaXP estimates. With install data the result is good (clvl 10 to 14: Black Marsh, 52 and 32.3 XP per kill), so this is a scope note, not a defect.
- Burial Grounds: with the traced table (recoveryxp.go:68-71) Burial Grounds (40.2) beats Cave Level 1 (35.2) and Cold Plains (25.6) at the same unit level 2. Scenario (V computed): Sorceress clvl 9, Stony Field and Dark Wood on nav skip. Old fallback = last in-window step in order = Cave Level 1 (9). New = Burial Grounds: 40.2 x 62.1 percent = 25.0 against 21.9. The recovery picker leaves Burial Grounds out on purpose (recoveryxp.go:13: Blood Raven, BRKILL kit gate). The BRKILL gate is on the quest boss action (task.go:592-608), not on a leveling sweep there (U whether a sweep engages her). And when `recoveryArea` finds nothing and the kit gate is off, target0 falls through to this Pick (goal.go:770-777), so a recovering character can get there too.
- Smallest fix: let Pick take the same "XP-safe" list as recovery (skip areas with a boss gated on a full kit unless `FullKit`), or add 17 to Exclude in target0 when the Blood Raven kit gate would refuse. Test: clvl 9 with 4 and 5 excluded must not return 17 without a full kit.

### R6 (medium) Missing monster level: the EnterLvl-3 proxy is on a different scale and wins every fallback offline

`stepXP` (recoveryxp.go:86-94) uses `EnterLvl - 3` when neither the install data nor `normalMLvl` (recoveryxp.go:20) has the area, and rates it with `areaXPPerKill` (10 x mlvl), not the traced table. Crypt (18, enter 10) gets ml 7 and Mausoleum (19, enter 11) ml 8. Real Normal levels are 3 and 3 (V levels.txt).
- Scenario (V computed, clvl 13, no install data: offline tests, a missing MPQ cache): Mausoleum 80.0, Crypt 70.0, Black Marsh 52 x 62.1 percent = 32.3, Dark Wood 47 x 43.0 percent = 20.2. Mausoleum wins although the real estimate is 30 x 5.08 percent = 1.5. With install data the order is right (Black Marsh first at clvl 13 to 14), so this matters for offline tests and sims more than live, but a sim built on it overstates the gain (xphshort.md's +1100 XP/h, S).
- Mixed scales: traced rows (exp per unit) and the fit (10 per level, "0.45x to 1.15x", recoveryxp.go:53-55) are compared in one ranking. An untraced step can win or lose by about 2x.
- Smallest fix: add 7: 8, 18: 3, 19: 3 (V levels.txt) and 20: 7 (U: MonLvl1 column, MonLvl1Ex read empty) to `normalMLvl`. Make `stepXP` return -1 for an unknown level, so the area ranks last instead of being guessed high. Test: Pick with no install data at clvl 13, 5 and 6 open, Mausoleum in the plan: must pick 6.

### R7 (low) Ties and difficulties other than Normal

- From clvl 25, `xpLevelPct` for a monster above the character is `100 x clvl / mlvl` (recoveryxp.go:37-38), so `areaXPPerKill` = 10 x clvl for EVERY area above the character (V: 740.0 for mlvl 75 to 85 at clvl 74, exact in float64). In Nightmare and Hell the rank does not tell +1 from +11: ties go to the later step (`>=`, plan.go:463), the same as the old order, so there is no regression, but the "by XP" ranking adds nothing there and is not the safer choice it looks like. Hell example: Mausoleum 85 and Crypt 83 tie with Black Marsh only when below the character, otherwise the latest in-window step wins.
- Below the character the fallback prefers the highest monster level in the window. It never leaves the entry and window set (same filters as Pick, plan.go:446), and target0 still runs `EntryAllowed` on the result (goal.go:848-856). worthFloor applies only when no in-window step exists (plan.go:467), as before. There is no cap like `RecoveryMaxAreaAbove`, but no step outside the plan window can come out (V).
- Smallest fix: none needed for safety. For clarity, say in the PickOptions.XP comment that from clvl 25 the rank is the plan order.

### R8 (low) CODEFIX side effects

- F1 + AREAREACH (V code, U live): `chainHome` (internal/resource/town.go) is 4, 5, 6, 9, 10, 17. Cave Level 2 (13), Crypt (18) and Mausoleum (19) are not in it, and those are exactly the dungeons the new reach rule and the XP rank send the character to at clvl 10 to 18. With no scroll and no healing potion there, TownTripTrigger has no method (no ByWalk) and `walkChainToTown` returns ErrNoPortal, so the bot fights on. Smallest fix: add 13, 18, 19 to chainHome only after a walk-home drill passes from them (U each), or keep them out of the XP rank while `HasTP` is false.
- F5 (V code): after a withdrawal the whole `PotionFloor` drops to 0 when the purse reaches RecoveryKitGold (chores.go:432-436). The visit may then spend the withdrawn gold past the kit (the rest of the belt). Tests check the floor lift, not the spend after it. Smallest fix: lower the floor to `purse - RecoveryKitGold` instead of 0.
- F2, F3, F4, F8: no problem found (V read). F3 `manaReserve(cur)` uses the live snapshot per purchase, correct.

### R9 (low) SPECDELTA

`BeltWindow.Funded` uses `TotMax`, the best minute of the window (internal/d2dsoakstat/invariants.go). A window poor for 30 minutes and funded in its last minute counts as 31 funded minutes. That matches the delta spec ("at some minute", archive/2026-11-03-specdelta/specs/soak-gates/spec.md:21, 32) and is the strict side, so no fix. An old archive without stashGold counts stash 0, also as the spec says. `hasHealing` ignores rejuvenation potions (from before this range, not new).

## Test coverage

- TestPickFallbackByXP: happy path only (one exclusion set, clvl 11, Normal). Its expected pick is Burial Grounds (R5). There is no test for ties, the Floor branch of pickByXP, a missing level (R6), Nightmare or Hell, or MercGoldFarm or guard exclusions as triggers.
- TestLongestWalk and TestDarkWoodReachableByWalkingTheChain prove the counting and the acceptance. Only one refusal case exists (the old 6-hop test). There is no test for a failed waypoint, `UseWaypoints` off or WpMinDistance (R1). No test runs LevelTo with a timing-out mover to prove the target changes (R2). No danger test (R3).

## A Gate 1 re-run should show

- `grep -c "leveling area unreachable\|action failed" bot-0.log` per target: no target with 5 or more failures in a row (R2).
- No "walking" of more than 5 area changes between two waypoint uses in `EvAreaChange` order (R1).
- No leveling sweep in Burial Grounds (17) with healing under KitMinHP (R5).

## Shared-file lines for the merger

See openspec/changes/codereview2/merge-lines.txt.
