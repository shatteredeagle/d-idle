# Tower Cellar 3: the cut-off gate (card cellar3gate)

Audit: `docs/probes/sandbox-postdeploy-audit-2026-11-27.md` problem 4 (V, log lines). Fixture: `internal/nav/cellar3gate_test.go` (two SYNTHETIC maps, the real grid is not in the repo).

## Finding
- V (test, must fail before the fix, `go test -p 1 -run 'BesideTheArrivalStairs' ./internal/nav`): the logged sequence "cut off from the gate, stepping to the nearest connected cell" then "no path ... (cut off, no way out found)" is reproduced with no click sent. `pocketTarget` took the connected cell NEAREST to the hero; that cell lay inside the mouth gate's disc of the arrival stairs (an unwanted exit, `mouthTriggerR` 15); the gate refused the click, `leavePocket` dropped the error, stood still eight rounds and the exit failed.
- Fix: `pocketTarget` prefers the nearest connected cell whose click the gate allows (`MouthClickOK`); `leavePocket` reports a refused click; the error now carries `cause=cut_off` or `cause=exit_trigger`; the quest board parks such a chunk as `no_path` (not `gave_up`) with the exit and cause in its text.
- U: that the real Tower Cellar 3 map is a thin known-grid artefact beside the arrival stairs. It could also be a real door the door search missed; then the grid capture below shows it, and the cause stays `cut_off`.

## Capture (U, needs a live instance; not run)
In a throwaway character at Tower Cellar 3 (level 23 on this client, gate to level 24), standing where the log says (about (45,20) of the level frame):
1. `d2cheat`-free reads only: `debug grid 23` (whole grid) and save the crop around (20..80, 0..60) as `internal/nav/testdata/cellar3.grid` in the format of `catacombs35.grid` (`#` blocked).
2. `debug goto` the hero to (45,20) and `debug objects` within 20 subtiles for doors (presets and snapshot).
3. `debug exits 23` for the stairs up (dest 22) and down (dest 24) positions.
Then replace the synthetic maps in `cellar3gate_test.go` by the crop (see `TestExitToCrossesTheLiveCatacombsTwo`).
