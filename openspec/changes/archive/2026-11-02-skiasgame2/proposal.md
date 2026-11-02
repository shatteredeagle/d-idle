## Why

SKIAS first pass (docs/research/sk-ias-trace.md) read only d2d's side of the swing speed and reproduced it as arithmetic. This second pass runs game.exe's real attack-rate code (FUN_00623f50, attack branch) under the emulator and compares it with d2d's attack_speed / attack_ticks.

## What Changes

- Docs and emulator script only: docs/research/skiasgame2-trace.md, tools/emu/skiasgame2_rate.py, one ledger row (formulas F07) and the regenerated docs/coverage/rule-coverage.md. No Go change, no live game.
