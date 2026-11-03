## Why

The melee PR component (undead and Blunt bonus) rests on the rule in docs/research/alignment2-trace.md: the undead block of FUN_0057b7d0 applies to a player or any unit with alignment 2. An independent re-check with its own harness was requested.

## What Changes

- docs/research/verify-alignment2.md: claim-by-claim verdicts (VERIFIED / HOLDS WITH LIMITS / NOT VERIFIED / WRONG).
- tools/emu/vfal_run.py: own emulator harness (random seeds, real stat-list code, decoy in ECX, native spawn map over 734 MonStats rows, raw call-site scan).

## Impact

Docs and an emulator script only. No spec, ADR or glossary change. Offline: no game, no d2d slot.
