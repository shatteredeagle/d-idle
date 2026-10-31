## Why

ADR 0009 turns the replicated soak into a fidelity check that finds gaps to fix and can fire the exit trigger; it no longer gates making d2d primary. Run it with the existing d2dsoak harness (docs/d2d-soak.md) against the Game.exe baselines already recorded; no new Game.exe runs.

## What Changes

- run 8 seeds by 2 virtual hours on the lab fork
- compare the four gated metrics with the recorded baselines
- list gaps with attribution per the new rule (engine vs bot)
