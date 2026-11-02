## Why

Decomp item PALAURA (open parts of docs/research/pal-skill-trace.md sections 3, 6 (D5, D6) and 10): which units an aura reaches (the aurafilter words 0x12003 and 0xa783 and the area search FUN_0056b7e0), what Holy Fire, Shock and Freeze do to enemies per pulse, and what Prayer's per-pulse mana and state 85 (`nomanaregen`) really do.

## What Changes

- Trace in game.exe 1.14d (disassembly plus emulator, game side): the area search shape, every bit of the filter FUN_0056b3e0 (MonStats flag bits decoded by data), the aura callback FUN_005cedc0, the friendly do FUN_005cf010 and the enemy do FUN_005cf3a0 / FUN_005d0920 with their roll and resist, Prayer's heal, mana payment and state 85, and the regeneration function FUN_005806f0 with the state.
- In this repo: `docs/research/palaura-trace.md` (Q, deviations D1..D9, bot findings B1..B3, drill recipe P1..P4, matrix text), `tools/emu/palaura_*` (own prefix), shared-file lines in `merge-lines.txt`. No Go change, no live game, no d2d slot, no PR (none proven by an existing d2d differential); `docs/coverage/ledger.tsv` is unchanged (the runs are game side only).
