## Why

Work item PALDUR (Issue #2): trace durability loss per hit and per block in game.exe 1.14d vs d2d.
no observable behaviour change: docs and offline emulator tooling only, no code under internal/** or cmd/**.

## What Changes

- Dedicated trace in `docs/research/durability-trace.md`
- Harness in `tools/emu/verify_paldur.py` (seed 0x50414c44, 100,128 cases, 0 mismatches)
- No Go code modified
