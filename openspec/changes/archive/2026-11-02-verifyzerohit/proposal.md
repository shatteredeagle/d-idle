# verifyzerohit

## Why

SKZEROHIT has two passes: a static first pass (docs/research/sk-zero-hit-trace.md) and a second pass with emulator runs (docs/research/zerohitrun-trace.md, branch agent/zerohitrun). Before their claims and the ledger row of the second pass are trusted, an independent verifier re-checks them with its own reading, its own model and its own game.exe runs.

## What Changes

- docs/research/verifyzerohit.md: verdict per claim of both passes, own model, run counts (seeds 101, 202, 303), the gates both passes missed, the event-2 dispatch pointer, ledger-row assessment.
- tools/emu/vzh_run.py (own harness: FUN_0057cee0, FUN_0057cb00, the enchant event walk and FUN_005a29a0 run natively) and tools/emu/vzh_xref.py (pointer search in game.exe's image).

## Impact

Docs and emulator scripts only. No Go code, spec, ADR or glossary change; no game, no d2d run; nothing pushed or posted.
