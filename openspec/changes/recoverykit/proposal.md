## Why

Gate 1 seed 19: a death at minute 155 left gold 0; minutes 158-175 the character farmed in recovery with 0 potions (pack gold 54..73, hp min 53 percent) because recovery suppresses field potion trips until RecoveryKitGold and ends at RecoveryTargetGold 900. The soak invariant counts gold >= 30 as funded, so the belt rule failed for 18 minutes. The user chose this fix over changing the gate rule (2026-11-03).

## What Changes

- resource.RecoveryKitTrip: in recovery, outside town, with no healing potion, a way home and gold >= KitPotionGold (existing knob), a short kit trip is due. bot.choresDue calls it and marks the trip hard and emergency; the existing chores and potionFloor (RecoveryPotionFloor potions) buy the minimal kit, then recovery goes on farming to RecoveryTargetGold.
- Build-specific part: none in the rule; the price is the KitPotionGold knob (internal/env/tuning_resource.go), the potion count RecoveryPotionFloor. No class check.

## Impact

Recovery characters with 0 potions and one potion's gold restock early. U: not run live or in a gate.
