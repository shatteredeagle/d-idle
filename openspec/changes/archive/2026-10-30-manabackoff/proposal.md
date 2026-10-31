## Why

Fresha (Sorceress, i5 soak 10:52-11:44) spent two thirds of the hour in "Recovery: farming gold" with no mana potions and logged 207 "out of mana: backing off" lines in 21 bursts (about 7 of 53 minutes). Two causes: the recovery potion floor zeroed mana potions for casters, and a back-off in a big pack (danger 2, 9-17 hostiles) never ends because melee needs pack <= 3 or a weak pack and standing needs danger < 2.

## What Changes

- `resource.backoffMaxSteps` (8): after that many consecutive back-off steps a weak caster swings its weapon (danger < 3, life >= meleeMinLifePct).
- `resource.recoveryManaFloor` (2): a recovering caster still buys up to 2 mana potions after the healing floor.
- log line `out of mana: back-off cap, melee`; `step` field on `out of mana: backing off`.
