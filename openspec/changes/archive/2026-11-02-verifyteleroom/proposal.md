## Why

Board item VERIFY-TELEPORT-ROOM: an independent re-check of the Teleport room rule claimed in docs/research/sk-teleport-trace.md (SKTELE): game.exe refuses a hop whose target lies outside the caster's room and its near list.

## What Changes

- docs/research/verifyteleroom-trace.md: per claim HOLDS / HOLDS WITH LIMITS / FAILS with case counts, seeds, commands, differences from the author's numbers, limits, drill recipe, matrix cell text.
- tools/emu/vtr_room.py, vtr_dis.py, vtr_cmp.sh: own emulator harness (player-typed unit, native near-list read, irregular rooms), disassembly helper, rerun of the author's numbers.

## Impact

Docs and emulator scripts only. No Go, no spec, ADR or glossary change. No ledger change (row 54 is already claimed by SKTELE's line).
