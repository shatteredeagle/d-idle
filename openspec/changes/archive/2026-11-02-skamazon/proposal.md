## Why

Coverage gap (docs/coverage/rule-coverage.md): the Amazon rows (skills.ama, 30 of 30) had no game.exe comparison, and the Amazon plans (Javelin, Bow) depend on Jab, Strafe, Fend, the javelin and arrow skills and the passives.

## What Changes

- docs/research/skamazon-trace.md: trace of the Amazon skills (game.exe vs d2d vs bot), deviations, bot findings, drill recipe, matrix cell text.
- tools/emu/ama_*: calc differential for Skills 6..35 and an elemental missile hit differential, both on the worker host.
- docs/coverage/ledger.tsv and rule-coverage.md (regenerated): only the rows the runs prove.

## Impact

Docs and emulator tooling only. No Go, spec, policy, ADR or glossary change.
