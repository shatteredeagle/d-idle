# stateevrun: SKSTATEEV second pass with runs

## Why

The first pass of board item SKSTATEEV (`docs/research/sk-state-events-trace.md`) was read-only: every claim V read, nothing run. A PR or a bot rule cannot rest on that. This pass runs game.exe's real swing gate, swing resolver and damage applier under the emulator with counted cases and says which first-pass claims hold, hold with limits or are wrong.

## What Changes

- `docs/research/stateevrun-trace.md` (trace with V run counts, deviations D1.., bot findings, drill recipe, matrix updates).
- `tools/emu/ser_events.py` and `tools/emu/ser_run.sh` (the harness and its worker-host entry).
- Docs and emulator scripts only: no Go, no live game, no spec delta (`skip_specs: true`). No overlap with another card.
