## Why

ORIGPROBES measured 25-30 % fewer monsters outdoors on d2d than on Game.exe and different champion/rare/superunique counts (docs/orig-spawns.md). UPSTREAMGAPS item 3 showed d2d's generator equals game.exe per room, so the gap sits in the measurements or in the rolls. This card traces game.exe's room-activation, density and unique/champion rules offline (unicorn harness, disassembly) and says what is apples to apples.

## What Changes

- `docs/research/spawn-density-trace.md`: the rules (addresses, numbers, V/S/U), which explains the density and the champion gaps, the analytic table vs measured, unknowns.
- `tools/emu/spawn_bosses.py`: oracle hook script (runs from upstream's tools/emu) tracing FUN_005be020 and FUN_005a0320.
- `docs/research/upstream-pr-candidates-spawn.md`: PR candidate or the reason there is none.

owns: docs/research/spawn-density-trace.md, docs/research/upstream-pr-candidates-spawn.md, tools/emu/spawn_*.py, openspec/changes/decompspawn/**
exclusive: none (offline; no game instance, no Go changes)
gates: none (research)
