# codefix67: F6 and F7 of codereview1

Offline only; not verified live (U). Findings: docs/research/codereview1.md F6, F7; background docs/research/codefix.md section 3.

## Finding and fix

- F6a (V, code): `Town.needWithdraw` fired whenever the purse was under the keep line, even when it paid the kit. Now it also needs `Gold < kitCost()` = `KitTPGold + max(HPFirst,1) * KitPotionGold` (existing tuning, no new knob). The amount stays the keep-line top-up.
- F6b (V, code): `healingCarried` missed pack rejuvenation, which the bot side (`resInput`) counts. It now counts it; the bot side has one helper `healingOf(resource.Input)`.
- F7 (V, code): `Runner.noteTownFutile` runs at the end of a trip. If the character is in town, carries no healing and the policy gold (stash included while no potion) did not grow by `idleTripGold` since the trip began (`trip.startBasis`), the latch is set. `needChores` then drops the emergency in-town round (`townLatched`) until gold grows, a potion appears, or the character leaves town. No threshold or policy change; no class literals.

## Tests

- `internal/town/codefix67_test.go`: `TestNoWithdrawWhenPurseAlreadyPaysTheKit` (Sorceress default, Barbarian with HPFirst 3), `TestNoWithdrawWithPackRejuvenation`.
- `internal/bot/codefix67_test.go`: `TestInTownFutilityLatch` (Sorceress recovery purse, Barbarian with stash gold).
- Verified to fail without the fix by hand reverts: F6 tests failed with the old `needWithdraw` (withdrawal of 250 / need true); F7 test failed with the latch check disabled. Town tests need the game tables (run on the worker host).

## Live recheck recipe (P5 stash drill, not run)

1. Character clvl about 11 in town, empty belt and pack, 600 gold, 1400 stash gold: run the town chores via the d2d town verbs; expect no stash round and a shop purchase.
2. Same with 6 gold: expect one withdrawal to the keep line, then potions in the belt.
3. 6 gold, stash empty, shop cannot sell: expect one round, then no further `town trip ... emergency=true` line while gold stays the same.
4. One rejuvenation potion only in the pack, 6 gold, stash 1400: expect no withdrawal.
