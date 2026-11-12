# Gold priority (TASK-377)

> status: current. Code: `internal/loot/goldprio.go`, `internal/bot/goldprio.go`, hook in `internal/combat/fight.go` (`ClearOpts.Interlude`), seams in `cmd/d2bot/adapters.go`. Knobs: `loot.goldPriority` and `loot.gold*` (docs/knobs.md).

User report: "The bot is still missing gold pickups, it should prioritize picking up currency basically just below surviving."

## What was lost and why (ranked)

Evidence: the sandbox log of the current hero, 56 minutes (`docker logs d2play-20261108`, 2026-11-12 18:48 to 19:44), 541 kills while farming. V by log count unless tagged.

- Piles picked: 42 (`Found ? Gold` lines), +301 gold in total (7 per pile). 20 of 36 farming minutes show `goldPickup 0`.
- Piles seen but never taken cannot be counted: the bot logged a pile only after a `PickUp` call looked at it (U: the new `gold ledger` line fixes this). The causes below come from reading the code (V by code) and from what the log does show.

1. **The loot memory was dead code in production (V).** `looterAdapter` (cmd/d2bot/adapters.go) did not forward `ObserveGround`, `ResetMemory` or `WalkBack`, so the runner's `r.d.Loot.(lootMemory)` assertion failed and the walk-back of docs/lootmemory.md never ran: 0 `loot_memory:` lines in 56 minutes. Every pile farther than the pickup radius (30) when the fight ended was lost.
2. **No pickup during a fight (V by code).** `PickUp` ran only after `Fighter.Clear` returned; a long Clear (a chase, a pack that keeps coming) leaves every drop behind. Nothing in the fight loop looked at the ground.
3. **Area leave skipped the pickup (V by code).** When `emptyAreaLeave` or `exploreGaveUp` ended a roam, and when `exploredEnough` stopped it, `sweepPass` returned before the end-of-leg walk-back; piles seen on the last leg were left.
4. **The roam did not stop for gold (V by code).** `Explore` ends on a hostile only; a pile seen mid-leg waited until the leg ended (minutes) or a fight began.
5. **A failed click forgot the pile (V by code).** After two failed rounds `pickUp` called `forget`, and `WalkBack` dropped an entry after two fails: one hover miss or a hostile walking in lost a far pile for good.
6. **Trip cap and bar for remembered gold.** `loot.memMaxTrips` (6) capped gold like items, and `loot.memMinGold` (20, +10 percent per clvl: 32 at clvl 7) skipped the small piles Normal drops (average 7 here). Not in the log (the memory never ran), by design in the code.
7. **Small-pile bar in `candidates`** (5 log lines, 7 piles `small gold pile`): intentional while gold >= `loot.poorGoldBelow` (300); unchanged. `loot.poorMinSell` (6 to 20) only touches junk items, not gold (V by code: the gold branch returns before the sell checks).

Potions, scrolls, keys, runes share `candidates()`; they get the mid-fight and walk-back paths only where the pickit would also keep them through the old memory (walk-back); the mid-fight grab is gold only (`PickOpts.GoldOnly`). U: scrolls and keys are not remembered by the memory unless the pickit keeps them.

## The fix

Position in the ladder: below every survival action, above fighting the next target (docs/inventory-actions.md ladder is unchanged above it).

1. **Mid-fight grab.** `ClearOpts.Interlude` is called once per fight tick after survival (potions, chicken, retreat), the merc potion, the hard-mode rules, the lost-fight check and the buff upkeep, before a target is chosen. `Runner.goldGrab` takes one pile (`PickUp{GoldOnly, Max 1, Abort}`) when life >= `loot.goldMinLifePct` (60), the nearest hostile is >= `loot.goldSafeDist` (8) away, the pack danger <= `loot.goldMaxDanger` (0, "none") and a wanted pile lies within `loot.goldGrabDist` (20). A hostile closing in aborts the click. A grab that takes nothing pauses itself for 2 s.
2. **Roam stop.** The `Explore` done-callback also returns true when `GoldNear` sees a wanted pile in the clear radius; the next pass picks it up before roaming on (a gold stop never counts as the area explored). A stop that picked nothing pauses for 20 s.
3. **Never leave with known gold.** `goldBeforeLeaving` (pickup, then `WalkBack`) runs before the sweep gives an area up (empty-area leave, explored enough via the cleared path, explore gave up, area cleared) and logs `gold ledger` (seen, picked, gold, abandoned).
4. **Retries.** A failed gold click stays remembered and is retried by later calls until `loot.goldRetries` (4) rounds failed; remembered gold ignores `memMinGold` and the item trip cap (up to 4x `memMaxTrips` trips).
5. **Adapter fix.** `looterAdapter` forwards the memory and gold methods and the `PickOpts` fields (Abort, Max, GoldOnly) and the gold gained.

Data, not code: all numbers are knobs; `loot.goldPriority 0` restores the old behaviour (the adapter fix and the ledger stay). No class literal; tests run Paladin and Sorceress (`internal/loot/goldprio_test.go`, `internal/bot/goldprio_test.go`, `internal/combat/interlude_test.go`).

## Not done / live checks

- U: how `Interact` behaves on a pile mid-fight on d2d (the click walks the hero; GoldSafeDist 8 is a guess, not a measurement).
- Live check: on a pack fight with drops, `gold grab: took a pile mid-fight` appears, `loot_memory:` lines appear, and `gold ledger` shows abandoned 0 or small; compare gold per farming minute with the 56-minute baseline above (301 gold, 20 of 36 minutes at 0).
- Leaving by town portal or stairs from outside `sweepPass` (chores, resupply trips, quest walks) has no gold stop.
