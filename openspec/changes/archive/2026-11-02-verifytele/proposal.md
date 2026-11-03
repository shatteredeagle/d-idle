## Why

SKTELE claims game.exe refuses a Teleport hop by a room rule (the caster's room plus its near list), not by a distance. A bot navigation limit and a d2d fix would rest on it, so an independent read and run of the game code checks it first.

## What Changes

Adds docs/research/verify-teleport-room.md and the emulator scripts tools/emu/vt_run.sh and vt_check.py. Docs and emulator scripts only: no Go, no live game, no shared file edited.
