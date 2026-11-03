## Why

Work item PALAURA (Issue #1): trace Paladin aura application, target filter, pulse damage, and Prayer mana / state 85 in game.exe 1.14d vs d2d vs the bot.
no observable behaviour change: docs and offline emulator tooling only, no code under internal/** or cmd/**.

## What Changes

- Dedicated trace in `docs/research/pal-aura-trace.md`
- Reproducibility verification harness in `tools/emu/verify_palaura.py` (seeds 0x50414c31, 0x50414c32, 0x50414c33; 300,000 cases, 0 mismatches)
- No Go code modified
