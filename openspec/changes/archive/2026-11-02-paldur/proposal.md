## Why

Board item PALDUR (paladin-act1-recon section 6 row 7): the chance and the slot choice of durability loss when the player hits and when the player is hit or blocks, in game.exe 1.14d against d2d. The repair price and gate were traced earlier (price-trace); the loss rule was not.

## What Changes

- docs/research/paldur-trace.md: the rule (V read, V run on game.exe's own FUN_0057d3d0 / FUN_00559e30 against d2d's real Fight code), deviations D1.., bot findings B1.., drill recipe, matrix and ledger proposals.
- tools/emu/pal_*: the differential (case generator, native runner, d2d probe, build and run scripts, a call-graph scan).

## Impact

Docs and offline emulator tools. No game, no Go code, no spec, policy, ADR or glossary change. No PR candidate (see the doc).
