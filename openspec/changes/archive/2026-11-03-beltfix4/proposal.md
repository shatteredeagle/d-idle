## Why

Gate 1 run gate1-20261103-0053 (bot cbf6905e) failed BELT with 7 virtual minutes (limit 5) and XP/h with 8162 against 9172 (d2dsoak INCONCLUSIVE). The 7-minute window is a fresh clvl 1-2 character with 0 to 16 gold; a second, shorter window (3 minutes, clvl 6) shows a real defect: recovery mode switched the in-town potion restock off and the character left the Rogue Encampment with 185 gold and no healing potion (docs/research/beltfix4.md).

## What Changes

- internal/bot/goal.go `choresDue`: recovery no longer suppresses the emergency potion chore in town when the belt and pack hold no healing potion and the gold buys one; potionFloor still caps the purchase.
- docs/research/beltfix4.md: window classification, XP/h decomposition, Gate 1 re-run advice.

## Impact

internal/bot/goal.go and internal/bot/beltfix3_test.go. No spec delta. Overlap (declared): both files were last edited by the archived beltfix3 card.
