## Why

Decomp board item 17 (SKDEFENDERS, source SKRESIST): docs/research/sk-resist-trace.md covered a monster as the defender. A hireling or a player that is hit is the other branch of the same function (the "plain monster" test decides it): resistance penalty, cap, floor, the full curse value, flat reductions, absorb, the poison fields. This card traces that branch in game.exe 1.14d, runs it natively under the emulator and compares it with d2d's rules (real `rules::monster_blow`) and with the bot's reading.

## What Changes

- `docs/research/skdefenders-trace.md`: questions Q1.., deviations D1.. (d2d vs game.exe), bot findings B1.., limits, drill recipe, matrix and ledger lines.
- `tools/emu/skdef_*`: the oracle (`skdef_resist.py`, `skdef_test.cpp`, runner and build scripts), own prefix.
- `docs/coverage/ledger.tsv` row for the proven rules and the regenerated `docs/coverage/rule-coverage.md`.
- Nothing in Go, no live game, no d2d slot, nothing pushed or posted. No PR candidate (see the trace, section 7).

## Impact

Docs and emulator tooling only.

## Overlap (declared)

docs/coverage/ledger.tsv and the generated docs/coverage/rule-coverage.md are shared by every trace card; this card only extends the note of the existing F04 row (the brief names both files) and regenerates the table with `python3 tools/coverage/coverage.py --write`. Other cards that edit other rows merge by hand-resolving the one line.
