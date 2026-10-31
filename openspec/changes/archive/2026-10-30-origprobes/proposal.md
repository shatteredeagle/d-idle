## Why

The d2d side of the spawn-count and casts-per-kill probes is merged (docs/d2d-spawns.md, docs/d2d-casts.md, docs/d2d-tick-rate.md). The original game is the acceptance target, so the same columns are measured on Game.exe (instance i0, Testy) for the comparison. The user approved the first verification probes on 2026-10-30.

## What Changes

- cmd/origcasts: census of the monster table while exploring whole levels (spawns), casts per kill with the bot's real Fighter (casts), a mana-regeneration frame-rate estimate (mana)
- docs/probes/orig-spawns.csv, docs/probes/orig-casts.csv, docs/orig-spawns.md, docs/orig-casts.md with the d2d-versus-original tables and caveats
