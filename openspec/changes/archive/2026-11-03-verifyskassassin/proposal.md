# verifyskassassin

## Why

SKASSASSIN (docs/research/skassassin-trace.md) is a static first pass on the 30 Assassin skill rows. Before its claims are trusted, an independent verifier re-checks them with its own disassembly reading, its own model and its own game.exe runs, and compares with d2d where a harness exists.

## What Changes

- docs/research/verifyskassassin.md: verdict per claim, own model of FUN_0056c160 / FUN_0056bfe0, run counts (seed 20261102), the game.exe vs d2d mana differential for ids 251..280, ledger assessment.
- tools/emu/vsa_dis.py (reading aid), tools/emu/vsa_mana.py (own oracle), tools/emu/vsa_diff.sh (game.exe vs d2d's mana_cost).

## Impact

Docs and emulator scripts only. No Go code, spec, ADR or glossary change; no game; nothing pushed or posted.
