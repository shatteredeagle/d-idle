# BELTFIX3: Gate 1 BELT and NOKILL fixes 1 to 4

> status: current. Implements section 6 items 1-4 of docs/research/beltfix2-analysis.md (branch agent/beltfix2). Items 5 and 6 are not done.

Tags: V = verified by a test run here, U = unverified live (no game was used).

## What changed

| # | change | test (fails when reverted, V) |
|---|---|---|
| 1 | `internal/bot/goal.go`: `choresDue` marks 0 healing potions an emergency also in town; `runGoal` uses a 5 s gap (`townShopGap`) for it instead of the 240 s resupply cooldown. Field starts keep the old pacing. | `TestTownArrivalRestocksWithoutCooldown`, `TestTownArrivalNoGoldNoEmergency` (internal/bot) |
| 2 | `internal/town/trip.go`: `walkChainToTown` (navigator `ToArea` to the act 1 town) after the waypoint and the hop chain fail. `resource.walkableHome` widened to Stony Field, Dark Wood, Black Marsh, Cave 1, Underground Passage 1, Burial Grounds so `kitTrip`, the resource trigger and combat's `nearTown` agree. | `TestToTownWalksChainFromDarkWood` (internal/town, needs /tmp/d2mpq: worker host), `TestKitTripNoPortalPotionPrice`, `TestNearTownArea`, `TestNearTownCoversEveryAct` |
| 3 | `cmd/d2bot/d2dwire.go`: a context dying after the click returns "cast cut after the click, source maybe spent"; the next portal call within 8 s waits up to 3 s for that cast's portal and returns success when it opened, before it reads another charge. | `TestPortalCutAfterClickReportsSpent` (cmd/d2bot) |
| 4 | `BackToField` tries the town portal twice before the waypoint and the walk. | `TestBackToFieldRetriesPortalOnce` (needs /tmp/d2mpq) |
| - | Replay of the Gate 1 BELT A window (minute lines 218-254; six rows embedded as area and gold): every field row is now an emergency potion trip, every town meeting restocks. | `TestReplayGate1BeltWindowDecisions` |

Decision for the part of #3 "a trip start while the safety guard is drinking waits for the drink": not implemented as a separate wait. The guard cancels the action with a drink; the cut-cast reuse in the portal verb is what stops the second charge from being burned. U whether that is enough: needs the Gate 1 re-run.

## NOT done

- #5 (mana-only kit trip keeps the potion gold; bank before poverty) and #6 (waypointToTown touching an inactive waypoint).
- `ToTown` in the town package does not itself know about a cut cast made by the fighter; the reuse lives in the portal verb.

## NOT verified live (U)

- The walk home from each of Stony Field, Dark Wood, Black Marsh, Cave 1, Underground Passage 1 and Burial Grounds (only the opposite direction was measured, 14 wall s). `nav.ToArea` tries a waypoint first when it knows one.
- The portal-cut path (whether game.exe/d2d spends a charge on a cut cast; the wait is bounded and costs at most 3 s).
- Widening walkableHome also changes combat: those fields now count as near town for the fight loop.

## Gate 1 re-run: what to grep in bot-0.log

- Every `Entered Rogue Encampment` with an empty belt and gold >= 30 is followed by `town trip` (`emergency=true`) and `shop: bought .. Healing`; no `state recovering -> clearing` straight after.
- `no town portal and no waypoint: walking the level chain to town` followed by `Entered Rogue Encampment`; `waypoint to town failed` must not repeat.
- `d2d portal: the cut cast opened its portal, not casting again`, and no pair of `cancelled, nothing clicked` lines one second apart with the tome charges dropping.
- `back to field: portal in town .. attempt=2` instead of a 14 s walk after a dropped click.
