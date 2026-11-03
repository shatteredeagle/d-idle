# verifystateev2

## Why

SKSTATEEV has two passes: a static first pass (docs/research/sk-state-events-trace.md) and a second pass with emulator runs (docs/research/stateevrun-trace.md, card stateevrun). Before their claims are trusted, an independent verifier re-checks them with its own reading, its own model and its own game.exe runs.

## What Changes

- docs/research/verifystateev2.md: own model, run counts, a verdict per claim of both passes, deviations (one new, I), ledger-row assessment, proposed matrix text.
- tools/emu/vse_dis.py (disassembly aid), tools/emu/vse_run.py and vse_run.sh (own harness: gate, resolver, FUN_0057c1e0 and the real chain run natively; game-vs-game absorb oracle).

## Impact

Docs and emulator scripts only. No Go code, spec, ADR or glossary change; no game, no d2d run; nothing pushed or posted.
