## Why

Riskmap rank 5 `mon_wake` (docs/research/d2d-fidelity-riskmap.md): when an idle monster notices the player and starts acting sets how many monsters the bot meets per minute and how many arrive at once (kills/h, crowd size, deaths), and the original's rule had never been read or run against d2d's. This card traces the wake chain in game.exe 1.14d offline (disassembly, unicorn oracle over the parts upstream's own oracles do not cover) and compares it with d2d's `ai.cpp` / `monsters.hpp`. no observable behaviour change: research, docs and an emulator script only; no Go code, no game instance.

## What Changes

- `docs/research/mon-wake-trace.md`: game.exe's rule with addresses (driver, search step, no-target cadence, scheduler, rally, last mode), d2d's with file:line, the deviations table with the differential results, what stays U.
- `tools/emu/wake_cadence.py`: oracle (runs from upstream's tools/emu) running FUN_005de890 and FUN_005a73e0 in game.exe's own code against a copy of d2d's rules; exhaustive plus random cases.
- Result: a clean negative (no deviation proven); no PR docs, no patch, the upstream worktree branch `fix/mon-wake` left empty.

owns: docs/research/mon-wake-trace.md, tools/emu/wake_*.py, openspec/changes/decompwake/**
exclusive: none (offline; no game instance, no d2d slot, no Go changes)
gates: none (research)
