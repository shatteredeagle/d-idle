## Why

The early baseline (tools/d2dsoak/baselines-early.json) holds one Game.exe run with xp_per_hour and deaths_per_hour only, so kills per hour and deaths per 1000 kills cannot be gated on a fresh-character d2d run, and the one-run verdict is always weak. The Game.exe minute lines and kill lines of the Soaky and Fresha soaks (clvl 1 to 7) exist on the worker host and were already read by docs/research/per-kill-time-split.md; the same inputs give the missing fields.

## What Changes

- tools/d2dsoak/baselines-early.json: a run entry per Game.exe fresh-character soak (Soaky clvl 1-7, Fresha clvl 1-7) with xp_per_hour, kills_per_hour, deaths_per_hour and deaths_per_1000_kills; a source line per run (file, time window, clvl window); kills_per_hour and deaths_per_1000_kills gated with rel 0.2 (and abs 1.0 per 1000 kills, fidelity-gate).
- docs/d2d-baselines-early.md: how each number was extracted (`d2dsoak extract` on the original-game logs), what is V and what is U, and the known undercount of kills (death-rule spec: 16 to 37 percent).
- Delta: fidelity-gate ADDED `Early baseline fields`.
- Read-only on the worker host: no game, no d2d slot, no cheat.
