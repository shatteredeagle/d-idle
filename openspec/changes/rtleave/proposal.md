## Why

RT.leave (P29 leave-game path) is blocked: Fighter.Leave is nil in the d2drill rig, so Act(Chicken) sends an unverified placeholder (docs/retreatlive.md, docs/retreatdrill.md).

## What Changes

Wire Leave in the d2drill rig (Save and Exit through the existing d2d control path, rejoin afterwards), unblock RT.leave with offline tests; the live run follows.

## Impact

Offline only (fakes and docs), nothing verified live.
