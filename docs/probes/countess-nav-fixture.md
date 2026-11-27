# Tower Cellar 5 walk fixture (cards countess3b, countessnav, 2026-11-27)

> status: offline regression done with SYNTHETIC maps; the recorded grid is still to be captured (live-only, needs a lab d2d).

- V (git log -p of `internal/nav/mouthgate.go`): fc75613fa, 6acf2d3e3, ba7d581d9, 5c0416d06, fb1fdb303 and the read-area gate (4d063a52f) came after the refusal of the first click west of the Tower Cellar 5 arrival; `mouthgoal_test.go` already covered the single waypoint (121,70) on an open field.
- V (offline): `internal/nav/cellar5walk_test.go` runs a whole `GoTo` from the arrival to the Countess stand on two synthetic corridor maps (level 25 sized 150 x 150, stairs (130,60), arrival (125,72), stand (21,69); map B mirrored). The walk is planned, 14 clicks, closest approach 13.0 to the stairs (inside the 15 bound, outside the 10 core), no `ErrMouthGate`. Must-fail pair: a goal reachable only through the stairs branch is not reached and the hero never enters the core; with the gate disabled that test fails (checked by hand). The positive test does NOT fail when the goal rule of `gateClick` is disabled: the detour rewrite covers the corridor now, so it guards the whole walk, not that rule.
- U: the real maze. Only a recorded grid shows whether the real corridor needs the goal rule.

## Live capture still needed (not run: no game, no lab d2d in this card)

On the worker host with the private fork binary (`--headless --lab-verbs`, as `tools/probes/gridwhole.py` starts it), seed 5, Paladin or any class: over the control socket send `debug goto 25`, `debug level`, `debug grid 5`, `debug presets` (the Countess is a monster preset: `debug levelmonsters`), and save the replies. Crop the grid around x 15..140, y 50..90 into `internal/nav/testdata/cellar5.grid` ('#' blocked, '.' free) and load it like `TestGoToThePastDoorInTheLairBesideTheStairs` does with `lair37.grid`, with exit (130,60) to level 24 and the hero at the arrival the log gives. Then swap the synthetic maps in `cellar5walk_test.go` for it.
