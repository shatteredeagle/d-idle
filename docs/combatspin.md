# combatspin: the one spin that survived NAVSTALL

Finding: GENINV offline validation, m2early baseline seed 2 (d2d lab6, main 35ce4c2): one real `combat no progress, holding {"iterations": 40}` line.

## What the log shows (VERIFIED by the bot log, 200 lines before the firing)
- State clearing, area 2, life about 35%, no healing potion, no portal, hostiles near: `combat abort fight: flee` (Safety AbortFight, Flee) alternating with `combat move skipped: the point is in a cave mouth`, the same target point 40 times within 15 ms of wall time, then the watchdog line; the fight then continued and ended normally.

## Root cause (VERIFIED by log + code reading)
- `abortFight` picked a flee point with `awayPoint(..., f.valid)`; `valid` only asked PathTo, so a point inside a cave mouth's trigger area (MouthGuard, d2d only) was accepted.
- `Fighter.move` skipped such a point and returned nil without sleeping (NAVSTALL only added the tick to failed walks), `abortFight` returned handled, the loop re-read the snapshot (unchanged clock) and chose the same point again.
- The watchdog turned it into one forced sleep per 40 iterations, which is why the run survived.

## Fix (internal/combat/fight.go)
- `move`: a skipped cave-mouth point costs one poll tick.
- `valid`: rejects cave-mouth points, so all away-point pickers choose another point.
- Same pattern elsewhere: the only other no-time return of `move` was this one; other callers go through `move`.

## Tests
- `combatspin_test.go`: flee with a mouth guard everywhere (before the fix the watchdog fires), skipped move advances time (before: 0s).

## Dead ends
- None: the first reading of the log was the cause.
