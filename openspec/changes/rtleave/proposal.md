## Why

RT.leave (P29 leave-game path) is blocked: Fighter.Leave is nil in the d2drill rig, so Act(Chicken) sends an unverified placeholder (docs/retreatlive.md, docs/retreatdrill.md).

## What Changes

Wire Leave in the d2drill rig (Save and Exit through the existing d2d control path, rejoin afterwards), unblock RT.leave with offline tests; the live run follows.

## Overlap (declared)

The d2drill and internal/drill files listed in tasks.md lie inside the card m1-liveproof's `cmd/d2drill/**` and `internal/drill/**` (the drills area); this card touches only the RT.leave hook, its lab side and tests.

## Impact

Offline only (fakes and docs), nothing verified live.
