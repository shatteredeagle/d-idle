## Why

Gate 1 run gate1-20261103-0338 failed the belt rule on a clvl 11 window (minute lines 333-339: 0 healing potions, 3 to 11 gold in the pack, "Recovery: farming gold"). The merc log of the same visits shows 1418 gold in the stash. The town package banks the surplus above the keep line (300 + 50 x clvl) and has no withdraw chore, so the shop and every purse test of the trip policy saw a poor character (docs/research/beltfix5.md).

## What Changes

- Town: new chore `withdraw` before the shop. With no healing potion in belt or pack, gold in the stash and less pack gold than the keep line, it takes the difference out (d2d MoveGold, withdraw direction already verified in docs/d2d-town.md).
- Bot: with no healing potion the resource input and `choresDue` count pack plus stash gold, so poverty mode, recovery, the futile/floor tests and the emergency potion chore no longer treat the banked gold as absent.
- No spec change: the proposed belt-rule delta ("a window counts once gold covers one healing potion") is written up with numbers in the doc, not applied.

## Impact

internal/town/basicsfix.go, internal/town/chores.go, internal/bot/goal.go, internal/bot/resource.go and their tests.

Overlap (declared): internal/town/chores.go is listed for fixprogress and internal/town/basicsfix.go, internal/bot/goal.go, internal/bot/resource.go and internal/bot/fakes_test.go are not in docs/OWNERS; this card adds one chore entry, two small helpers, one gold adjustment and one fake field. No signature changes.
