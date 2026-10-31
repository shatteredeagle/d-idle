## Why

LUTGHOLEIN (docs/research/lutgholein-trace.md, archived card lutgholein) showed Lut Gholein's room seeds in d2d's `drlg-dump` two level-RNG steps behind game.exe (emulator: 0/20 seeds equal, 20/20 equal with a two-room shift) but left the mechanism untraced (U). The user's PR standard (docs/process/upstream-pr-standard.md) needs the game.exe function that takes the two steps, the d2d fix, a differential at 0 mismatches, a regression over the other preset levels and an independent verifier before a PR may be drafted.

## What Changes

- tools/emu/lut_trace.py: every write to the level seed (level +0x1c4) while game.exe generates Lut Gholein under upstream's emulator, with the writing function and call stack; tools/emu/lut_seeds.py: room-seed comparison of two dumps (shift 0 vs shifted)
- the cause and the fix, PR-ready on a local branch of the upstream clone on the worker host (never pushed)
- docs/research/lutgholein-proof.md (the proof, V/S/U), docs/research/pr-lut-gholein-draft.md (their template), docs/research/pr-lut-gholein-evidence-pack.md (PROOF complete/incomplete, commands not run)
- no code in internal/ or cmd/; nothing posted upstream
