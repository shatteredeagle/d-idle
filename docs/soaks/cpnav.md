# cpnav: Assassin could not reach Cold Plains (TASK-184)

Tags: V = verified live (class pace drill, lockstep d2d, lab verbs off, no memory writes: `writes after start: 0`), S = from code.

## Root cause (V)
Not Assassin-specific: the same refusal appears in the Necromancer and Druid pace logs (other waypoint, same pattern); it is seed geometry. Blood Moor of seed 7 has a corridor on the route to the Cold Plains exit (gate 102,479) that runs 14 to 15 subtiles from the Den of Evil mouth (95,340). The planner priced the exit tile only out to `exitAvoidR` (14), so the path hugged the corridor at about 14.6 where its cost was zero. The mouth gate (`mouthTriggerR`, 15) refuses every click whose target is within 15 of an unwanted exit, so the waypoint (81,344) was refused ("click would cross the trigger area of an unwanted exit"), the walk failed and the planner skipped Cold Plains for 150 s ("leveling area unreachable"). Planner radius (14) below gate radius (15) is the whole bug. It was deterministic: both Assassin runs (ass1, ass2) and the two reproductions below failed on the same click.

## Fix
`internal/nav/nav.go`: `exitPenaltyR` = `mouthTriggerR + 3` replaces `exitAvoidR` as the radius of the planner's exit penalty (the goal exemption and the roam filter keep `exitAvoidR`). Class-free: no class or build data enters nav. The penalty stays a cost, not a wall.

## Tests
`TestGoToKeepsMouthGateDistanceFromUnwantedExit` (internal/nav/portal_exit_test.go) walks an open field past an unwanted exit and asserts the walked path keeps `mouthTriggerR`; before the fix it came within 12.4 and 13.9, after it passes. Nav takes no class, so the test is class-independent by construction; `go test -p 1 ./internal/nav` passes.

## Drill before and after (V)
Command: `labrun <worker> --detach -- docs/probes/classpacing/run-class.sh TAG 6 7 6m assassin_fireblast_normal30`.
- Before (cpb, and the earlier ass1 / ass2): at clvl 5 `level plan: clvl 5 -> area 3`, then `nav mouth gate: click held` on {81 344} and `leveling area unreachable`.
- After (cpc): `level plan: clvl 5 -> area 3 Cold Plains` followed by `Entered Cold Plains`; no `unwanted` line and no `leveling area unreachable` in the log.
Other classes: the change only widens a cost disc around exits that are not the destination; their paths can shift by a few subtiles. Not re-run for them (UNVERIFIED live); the nav package tests pass.

## Not done
The map of seed 7 was not captured as a fixture (the d2d slot ended before a Blood Moor dump); the regression test uses a synthetic field. Whether the corridor can also be passed at 15 or more is shown by the after run (it passed), not by a dump.
