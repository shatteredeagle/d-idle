# codefix: fixes for the belt-fix review

Review: docs/research/codereview1.md (branch agent/codereview1). Tags: V = verified by a test run here or by reading the code, U = not verified live. All offline; no game was used.

## 1. What changed per finding

- F1 (high). `resource.NearTownArea` is the short list again (2, 8, 3), so the fight abort, pre-engage and the potion-low trip policy keep their old behaviour (V: tests). New `resource.ChainHomeArea` (short list plus 4, 5, 6, 9, 10, 17) feeds `Input.ChainHome`; `TownTripTrigger` accepts the walk only when ChainHome is set and no healing potion (belt, rejuvenation, pack) is carried; the kit trip does the same (`k.HP == 0`). Decision others must know: a potion-low trip with 1 potion in Black Marsh is "town too far" again.
- F2 (high). `Runner.noteTripEnd` stores the gold on the policy's basis (stash included while no healing potion), used at trip end in goal.go. A visit that bought nothing is futile on the next check (V).
- F3 (medium). `Town.manaReserve(s)` reserves the scroll price only without portal source and the potion price only while healing carried is under kitMinHP.
- F4 (medium). `walkChainToTown` returns ErrNoPortal outside `ChainHomeArea` (Catacombs, Andariel's Lair, Tamoe and so on).
- F5 (medium). When a withdrawal happened in the same Chores round and the purse is at or above recoveryKitGold, the shop visit ignores `PotionFloor` (`Town.withdrew`).
- F8 (low). `afterCut` keeps the cut-cast memory when the context is already dead.

## 2. Tests (each fails when its fix is reverted by hand; V)

F1 `TestChainHomeIsNotNearTown`, `TestChainHomeTripOnlyWithoutHealing`, `TestEscapeInChainFieldLeavesGame`, `TestNearTownCoversEveryAct` (updated), `TestChainFieldTripPolicy`; F2 `TestStashGoldDoesNotDefeatFutileGuard`; F3 `TestManaReserveOnlyWhileKitIncomplete`; F4 `TestToTownNoChainWalkOutsideTheRoute`; F5 `TestRecoveryFloorLiftedByWithdrawal`; F8 `TestPortalCutMemorySurvivesDeadContext`. The BELTFIX3/4/5 and HIREFIX tests pass unchanged except `TestNearTownArea`, which pinned the widened list. Town tests need the game tables (the worker host has them; they skip on a small box).

## 3. Open (not done)

F6 (withdraw only when the purse cannot pay the kit, one healing count), F7 (in-town futility latch), F9 (no change needed), F10 (no change needed). F6 and F7 are more than a few lines with a test.

## 4. Not verified live; what a Gate 1 re-run should show

- Trips in areas 4, 5, 6, 9, 10, 17 only with 0 healing potions; no "walking the level chain" lines from other areas.
- No repeated `town trip` lines with `emergency=true` and an unchanged `gold+stashGold`.
- More mana potions per visit once the kit is complete; the Sorceress XP/h of runs 3 to 5 against 1 and 2.
- After a stash withdrawal in recovery, more than 2 healing potions in the belt.
