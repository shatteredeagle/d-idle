## Why

SKMELEE2 (docs/research/sk-melee2-trace.md) names five deviations between game.exe's player blow and d2d (D1, D2, D4, D5, D6). An independent check with its own harness is needed before any of them is relied on.

## What Changes

- `docs/research/verify-skmelee2.md`: per deviation HOLDS / HOLDS WITH LIMITS / FAILS with own case counts (seeds 11 and 3), commands and the differences from the author's numbers.
- Own harness `tools/emu/vm2_*` (game.exe's FUN_0057d9b0 / FUN_0057b420 / FUN_0057b660 under the emulator; d2d's real make_fighter / player_blow in a probe build).
- No Go code, no game, no slot; nothing pushed or posted.
