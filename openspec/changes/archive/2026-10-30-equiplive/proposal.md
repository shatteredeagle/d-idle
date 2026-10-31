## Why

On the Game.exe backend every town equip of a found magic piece fails ("item still on the cursor") and the belt drag of mana potions fails; each failure costs three clicks and a repeat trip (soak24 i4/i5, 2026-10-30).

## What Changes

- log every slot click with point, slot and cursor item
- remember items the game refused and do not retry them until the character levels up
- document the diagnosis in docs/gear-gate.md
