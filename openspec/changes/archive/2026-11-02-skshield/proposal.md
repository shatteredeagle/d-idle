## Why

Board item 15 (SKSHIELD, backlog SKMELEE2): SKMELEE and SKMELEE2 left open whether a carried shield's damage, or any enhanced damage on gear other than the weapon, reaches the player's stats 21, 22, 17 and 18 in game.exe 1.14d (the carrier flag 0x40000000, FUN_00627910 and friends), and whether a barehanded player with a shield in the off slot gets the shield's stats. d2d ignores all of it.

## What Changes

- Trace doc `docs/research/skshield-trace.md`: the equip and hand-toggle path read in game.exe (FUN_004c0d20, FUN_00627910, FUN_00626e10, FUN_00535d10, FUN_0063bef0 / bf90, FUN_00557ab0 / FUN_005563d0), the same code run under the emulator on 382 scenarios and on every Weapons.txt / Armor.txt row (item maker), d2d and bot findings, a PR decision, a drill recipe, matrix cell text.
- Emulator tools `tools/emu/skshield_*.py`: the equip and merge run, the item base-damage run, a re-compare of SKMELEE2's kept files, a retracted model kept as a negative control, and reading aids (disassemble, xref, scan, function starts, full listing).
- No Go code, no game, no slot, no d2d change; nothing pushed or posted.

## Overlap (declared)

Corrections to docs/research/sk-melee2-trace.md and docs/research/sk-melee-trace.md are given as proposed text in the trace, not edited here. Shared-file lines are in merge-lines.txt.
