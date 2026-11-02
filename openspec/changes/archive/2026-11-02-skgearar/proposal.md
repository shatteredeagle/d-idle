## Why

Board item SKGEARAR: game.exe adds attack-rating terms from gear (stats 115, 116, 120, 123, 124, 179) that d2d has no input for (sk-melee2-trace.md D3).

## What Changes

- docs/research/sk-gearar-trace.md: rules with address tags, deviations D1..D5, bot findings, limits.
- tools/emu/sk_gearar.py (runs FUN_0057d8a0 natively against a written reading) and sk_gearar_dis.py (disassembly aid).
- No Go code, no game, nothing pushed.

## Impact

Docs and emulator scripts only. Shared-file lines are in merge-lines.txt.
