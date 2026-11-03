## Why

PALSKILL (docs/research/palskill-trace.md) says Smite, Zeal and Sacrifice are built in d2d and lists deviations D1 to D9. An independent check with its own generator, model and seeds is needed before any is relied on.

## What Changes

- `docs/research/verifypalskill2.md`: per claim VERIFIED / HOLDS WITH LIMITS / NOT VERIFIED / WRONG, own case counts (seeds 5 and 23), commands.
- Own harness `tools/emu/vp2*` (game.exe's FUN_00646ca0, FUN_006449f0, FUN_0057b420 under the emulator, d2d's real rules through the existing pal2_test probe).
- No Go code, no game, no slot; nothing pushed or posted.
