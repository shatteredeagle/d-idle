# Loot memory

> status: current (TASK-320). Code: `internal/loot/memory.go`; hooks in `internal/bot/observe.go` and `goal.go` (`walkBackLoot`).

Problem: `PickUp` only looks at the ground within `loot.pickRadius` (30 subtiles) of the hero at call time. Gold and items that dropped while the hero chased a pack, walked a leg or fled stayed behind and were forgotten (V by reading `pickup.go candidates`).

## What it does

1. **Remember** (`Looter.ObserveGround`, called from `Runner.trackAlive` every observed tick while alive and outside town). Each ground unit id is judged once with the normal pickit (`EvaluateWith`, `passes` with the tuned strictness, the gold-pile bar). Taken units are stored with id, class, name, position, area, first-seen time, gold amount or sell value. Trashed items (`SetIgnore`) never enter.
2. **Forget**: the unit is absent from the ground list while the hero is within 50 subtiles of its spot (picked by us, the merc, a pursuit, or despawned); `PickUp` picked it; `PickUp` failed it twice; a walk-back found it still there twice; a new game (`ResetMemory`, called where the observer counts `newGames`). Eviction at `loot.memCap`: lowest value first, then oldest. Entries survive zone changes inside one game and are only fetched while the hero is in their area.
3. **Walk back** (`Looter.WalkBack`, via `Runner.walkBackLoot`), at quiet moments: in `sweepPass` after the pickup when nothing hostile is within the clear radius, and again after a roam leg returns, before the area counts as cleared. Candidates are beyond `loot.pickRadius` and within `loot.memMaxDist` (straight line), worth the walk (gold >= `loot.memMinGold` * (1 + 0.1 * (clvl - 1)); items the pickit keeps or stashes, or sell value >= `loot.valuableMinGold`), nearest first, at most `loot.memMaxTrips` trips per visit of an area. Each trip is `PickUp{Center: spot, Radius: 6}`, so room, ignore, hover confirm and the abort rule are the normal ones. The abort is the sweep's own `hostile()` test (a hostile within the clear radius ends the walk-back). One Info line: `loot_memory: going back for <gold> gold / <item>` (fields `drops`, `area`) and a summary `loot_memory: N trips, picked M, gold +G`.

Knobs: `loot.memEnabled` (default 1, class-generic), `memMinGold`, `memMaxDist`, `memMaxTrips`, `memCap` (docs/knobs.md).

## Verification

- V (offline, `mem.Fake`-style sim, Sorceress and Paladin): drop beyond radius then fetched; near drop taken by normal pickup is not duplicated; item that left the ground forgotten, a far absent one kept; cap/eviction; area change keeps and does not fetch cross-area; reset; trashed never enters; min value scaled by clvl and max distance; max trips; no walk-back with a hostile; off switch. `memory_test.go` (needs the game tables: `D2_MPQ_DIR`, they skip without).
- U (live): that d2d's ground list keeps far items while their room is loaded and drops them when it unloads (the 50-subtile "visible" rule guesses this); that straight-line distance is a good proxy for walking distance; how often walk-backs cross rooms that are still dangerous. Live check: after a pack fight, the hero walks back for a pile dropped more than 30 subtiles away (log line above).
- Not done: exit/town departure hooks (the cleared-area point covers the common case); the watch senses `loot_memory` payload and map markers (skipped: the map panel is being changed by another agent).
