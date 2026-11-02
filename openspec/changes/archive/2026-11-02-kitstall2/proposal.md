## Why

KITSTALL (docs/kitstall.md) left one question open: on baseline lane i2 a Sorceress lost her starting weapon and a worn pair of gloves around the sell chore (11:33:33 to 11:34:46) and later ended with a Short Bow in the weapon slot.

## What Changes

- Evidence (S, bot log): the Short Staff (class 63) and Leather Gloves (334) are worn in the dump at 11:33:33 and gone in the dump at 11:34:46; the gloves were never in the start kit and were worn with no equip chore line, so the game wears a picked armor or weapon into a free slot. The bow was picked at 11:35:57 with the weapon slot empty and was worn at 11:36:05 with no chore running: the game wore it on pickup.
- loot: `unwear` takes a launcher or throwing weapon off again when the character is not an Amazon (it did that for quest items only).
- town: `sellTo` refuses any unit that is not a plain pack item (worn or start kit) and logs every sale; `sellTo` logs the worn items before and after each vendor (instrumentation for the open question); `foundGear` wears the start weapon again when the slot is empty.
- sim: `AutoWearWeapon` models the game wearing a picked weapon.

## Impact

Code in internal/loot, internal/town, internal/town/sim; tests offline. No spec, ADR or glossary change. How the worn items left the body is still U (the sell code only offers pack items), see docs/kitstall.md.

## Overlap (declared)

kitstall (agent/kitstall, not merged when this card was written) touches internal/town/ammo.go and internal/gear; this card does not.
