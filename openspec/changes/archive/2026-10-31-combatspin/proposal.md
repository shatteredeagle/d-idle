## Why

NAVSTALL's watchdog (40 fight-loop iterations without the clock moving) fired once in the m2early baseline (seed 2): a break-away (no potion, no portal, life falling) chose a flee point inside a cave mouth's trigger area. `Fighter.move` skipped it without sleeping, `abortFight` reported the tick handled, and the loop re-picked the same point 40 times. The failed-walk tick of NAVSTALL did not cover this second no-time return of `move`.

## What Changes

- `Fighter.move`: a point skipped because it lies in a cave mouth costs one poll tick, like a failed walk.
- `Fighter.valid` rejects cave-mouth points, so every away-point picker (flee, kite, dodge, crowd step-away) chooses another point instead of one `move` would skip.
- Tests: `internal/combat/combatspin_test.go` (flee with a mouth guard everywhere: watchdog must not fire; skipped move advances time). Doc: `docs/combatspin.md`.
