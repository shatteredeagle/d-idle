## Why

Two user decisions of 2026-11-03 change how Gate 1 is judged. A fresh clvl 1-2 character with 10 to 16 gold cannot buy a 30 gold potion, so a belt window there is poverty, not a bot fault; but the clvl 11 window of gate1-20261103-0338 held 1418 gold in the stash (a bot defect, fixed by beltfix5) and must still fail. A single seed can never give a conclusive XP verdict, so a run inside the tolerance must not fail on XP alone.

## What Changes

- Delta spec (soak-gates): MODIFIED "Gate 1 belt and portal" (a window over 5 minutes counts only when pack plus stash gold covers a Minor Healing Potion, 30 gold, at some minute; otherwise a reported poverty window) and MODIFIED "Gate 1 deaths and progression" (single seed inside the tolerance with d2dsoak INCONCLUSIVE passes with the note "one seed, inconclusive").
- internal/d2dsoakstat: new `BeltFundedMin` drives the BELT invariant; `BeltWindowMin` stays the raw longest window; the report line shows both and tags each window "counts" or "poverty, not failing". Old archives without stashGold: stash = 0.
- tools/gate-judge.py (ops repo): XP rule `xp_gate1`, with a unit test.
- Re-judge of the recorded Gate 1 runs: docs/research/specdelta-rejudge.md.
- docs/decisions.md: two index lines (no ADR).

## Impact

internal/d2dsoakstat, openspec/specs/soak-gates (via the delta), docs. Observable change: the BELT invariant and the d2dsoak report line.

## Overlap (declared)

internal/d2dsoakstat/invariants.go and invreport.go were last edited by the finished cards stashfield and specmin (the stale card soak-minutes-virtual still lists the same paths); this card builds on them.
