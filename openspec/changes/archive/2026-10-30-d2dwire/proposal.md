## Why

The d2d backend flag exists but the bot behind it was a partial stack. The whole behaviour stack (combat, town, leveler, task, merc, liveness) has to run over the d2d seams so the replicated soak can drive the real bot, with socket ownership rules so two agents never drive one d2d.

## What Changes

- d2bot --backend d2d builds the same stack as the Game.exe path (cmd/d2bot/d2dwire.go)
- socket ownership safety: --d2d-attach, owner marker, per-instance default socket
- heartbeat liveness for a hung d2d feeds the failsafe ladder
- docs/d2d-wire.md describes what runs and what is left off (loot not wired)
